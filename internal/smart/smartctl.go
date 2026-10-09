package smart

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// DiskInfo is the parsed view of a disk's SMART status.
type DiskInfo struct {
	Device      string
	Model       string
	Serial      string
	Firmware    string
	Capacity    string
	Rate        string
	Health      string // "PASSED" | "FAILED" | "UNKNOWN"
	Temperature int    // current temperature in C, -1 if unknown
	TempMin     int
	TempMax     int
	Attributes  []Attribute
	// NVMe-specific extras (empty on SATA drives)
	Endurance               string // percentage used
	DataRead                string
	DataWritten             string
	AvailableSpare          string
	AvailableSpareThreshold string
	HostReadCmds            string
	HostWriteCmds           string
	ControllerBusyTime      string
	PowerCycles             string
	PowerOnHours            string
	UnsafeShutdowns         string
	MediaErrors             string
	ErrorLogEntries         string
	IsNVMe                  bool
}

// Attribute is one SMART attribute row.
type Attribute struct {
	ID         int
	Name       string
	Flag       string
	Value      int
	Worst      int
	Threshold  int
	Type       string
	Updated    string
	WhenFailed string
	Raw        string
}

// Severity classifies the attribute state relative to its threshold.
func (a *Attribute) Severity() int {
	if a.Value < a.Threshold {
		return 2 // bad
	}
	if a.Value < 80 {
		return 1 // warn
	}
	return 0 // ok
}

// Run executes smartctl on device and parses the output.
func Run(device string) (*DiskInfo, error) {
	args := []string{"--attributes", "--health", "--info", device}
	cmd := exec.Command("smartctl", args...)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		if msg := errb.String(); strings.Contains(msg, "Permission denied") {
			return nil, fmt.Errorf("insufficient permissions: run smartctltui with sudo")
		}
		return nil, fmt.Errorf("smartctl: %v: %s", err, strings.TrimSpace(errb.String()))
	}
	return ParseOutput(device, out.String()), nil
}

// ParseOutput parses the output of smartctl --attributes --health --info for a single device.
func ParseOutput(device, text string) *DiskInfo {
	d := &DiskInfo{Device: device, Temperature: -1, TempMin: -1, TempMax: -1}

	// key-value fields
	for _, line := range strings.Split(text, "\n") {
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		switch key {
		case "Device Model", "Model Number":
			if d.Model == "" {
				d.Model = val
			}
		case "Serial Number":
			if d.Serial == "" {
				d.Serial = val
			}
		case "Firmware Version", "Firmware":
			if d.Firmware == "" {
				d.Firmware = val
			}
		case "User Capacity", "Total NVM Capacity", "Local Capacity":
			if d.Capacity == "" {
				d.Capacity = val
			}
		case "Rotation Rate":
			d.Rate = val
		case "Temperature":
			v := strings.TrimSpace(strings.TrimSuffix(val, "Celsius"))
			v = strings.TrimSuffix(v, "°C")
			if t, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
				d.Temperature = t
			}
		case "Current Temperature":
			if t, err := strconv.Atoi(strings.TrimSpace(strings.TrimSuffix(val, "Celsius"))); err == nil {
				d.Temperature = t
			}
		case "Power Cycle Min/Max Temperature":
			if parts := strings.Split(strings.TrimSuffix(val, "Celsius"), "/"); len(parts) == 2 {
				d.TempMin, _ = strconv.Atoi(strings.TrimSpace(parts[0]))
				d.TempMax, _ = strconv.Atoi(strings.TrimSpace(parts[1]))
			}
		case "Percentage Used":
			d.Endurance = val
			d.IsNVMe = true
		case "Data Units Read":
			d.DataRead = val
		case "Data Units Written":
			d.DataWritten = val
		case "Power On Hours":
			d.PowerOnHours = val
			if d.Rate == "" {
				d.Rate = "NVMe"
			}
		case "Available Spare":
			d.AvailableSpare = val
		case "Available Spare Threshold":
			d.AvailableSpareThreshold = val
		case "Host Read Commands":
			d.HostReadCmds = val
		case "Host Write Commands":
			d.HostWriteCmds = val
		case "Controller Busy Time":
			d.ControllerBusyTime = val
		case "Power Cycles":
			d.PowerCycles = val
		case "Unsafe Shutdowns":
			d.UnsafeShutdowns = val
		case "Media and Data Integrity Errors":
			d.MediaErrors = val
		case "Error Information Log Entries":
			d.ErrorLogEntries = val
		}
	}

	if strings.Contains(text, "NVMe") {
		d.IsNVMe = true
	}

	lower := strings.ToLower(text)
	if strings.Contains(lower, "smart overall-health self-assessment test result:") {
		if strings.Contains(lower, "result: passed") {
			d.Health = "PASSED"
		} else {
			d.Health = "FAILED"
		}
	} else {
		d.Health = "UNKNOWN"
	}

	// attribute rows:  ID NAME FLAG VALUE WORST THRESH TYPE UPDATED WHEN_FAILED RAW
	reAttr := regexp.MustCompile(`^\s*(\d+)\s+([A-Za-z0-9_]+)\s+0x[0-9A-Fa-f]{4}\s+(\d+)\s+(\d+)\s+(\d+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(.*)$`)
	reFlag := regexp.MustCompile(`0x[0-9A-Fa-f]{4}`)
	for _, line := range strings.Split(text, "\n") {
		m := reAttr.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		id, _ := strconv.Atoi(m[1])
		val, _ := strconv.Atoi(m[3])
		worst, _ := strconv.Atoi(m[4])
		thr, _ := strconv.Atoi(m[5])
		d.Attributes = append(d.Attributes, Attribute{
			ID:         id,
			Name:       m[2],
			Flag:       reFlag.FindString(line),
			Value:      val,
			Worst:      worst,
			Threshold:  thr,
			Type:       m[6],
			Updated:    m[7],
			WhenFailed: m[8],
			Raw:        strings.TrimSpace(m[9]),
		})
	}

	// temperature from raw attributes when not in the info section
	if d.Temperature < 0 {
		for _, a := range d.Attributes {
			n := strings.ToLower(a.Name)
			if strings.Contains(n, "temperature") && strings.Contains(n, "c") {
				raw := strings.TrimSpace(a.Raw)
				if m := regexp.MustCompile(`^\s*\d+`).FindString(raw); m != "" {
					if t, err := strconv.Atoi(strings.TrimSpace(m)); err == nil {
						d.Temperature = t
					}
				}
				if m := regexp.MustCompile(`\((\d+)/(\d+)\)`).FindStringSubmatch(a.Raw); m != nil {
					d.TempMin, _ = strconv.Atoi(m[1])
					d.TempMax, _ = strconv.Atoi(m[2])
				}
				break
			}
		}
	}

	// dedupe by ID (some drives report the same attribute twice, e.g. 170/232)
	seen := map[int]bool{}
	uniq := make([]Attribute, 0, len(d.Attributes))
	for _, a := range d.Attributes {
		if seen[a.ID] {
			continue
		}
		seen[a.ID] = true
		uniq = append(uniq, a)
	}
	d.Attributes = uniq

	// NVMe: synthesize attribute rows from the health-log key-value fields
	if d.IsNVMe {
		// parse the leading integer from strings like "55 240 836 [28,2 TB]" or "3 761"
		// (smartctl separates digit groups with spaces / U+00A0 / U+202F)
		reNum := regexp.MustCompile(`^[0-9 ]+`)
		firstNum := func(s string) int {
			clean := strings.ReplaceAll(strings.ReplaceAll(s, "\u00a0", " "), "\u202f", " ")
			m := reNum.FindString(clean)
			if m == "" {
				return -1
			}
			d := strings.ReplaceAll(m, " ", "")
			if d == "" {
				return -1
			}
			v, _ := strconv.Atoi(d)
			return v
		}
		add := func(id int, name, raw string, thr int) {
			v := firstNum(raw)
			if v < 0 {
				return
			}
			d.Attributes = append(d.Attributes, Attribute{
				ID: id, Name: name, Flag: "0x0032",
				Value: v, Worst: v, Threshold: thr,
				Type: "NVMe", Updated: "Always", WhenFailed: "-", Raw: raw,
			})
		}
		if d.Endurance != "" {
			if p := firstNum(d.Endurance); p >= 0 {
				d.Attributes = append(d.Attributes, Attribute{
					ID: 1, Name: "Endurance_Percentage", Flag: "0x0032",
					Value: 100 - p, Worst: 100 - p, Threshold: 0,
					Type: "NVMe", Updated: "Always", WhenFailed: "-", Raw: d.Endurance,
				})
			}
		}
		if d.Temperature >= 0 {
			d.Attributes = append(d.Attributes, Attribute{
				ID: 2, Name: "Temperature", Flag: "0x0032",
				Value: d.Temperature, Worst: d.Temperature, Threshold: 0,
				Type: "NVMe", Updated: "Always", WhenFailed: "-", Raw: fmt.Sprintf("%d°C", d.Temperature),
			})
		}
		if d.DataRead != "" {
			add(3, "Data_Units_Read", d.DataRead, 0)
		}
		if d.DataWritten != "" {
			add(4, "Data_Units_Written", d.DataWritten, 0)
		}
		if d.AvailableSpare != "" {
			add(5, "Available_Spare", d.AvailableSpare, 0)
		}
		if d.AvailableSpareThreshold != "" {
			add(6, "Available_Spare_Threshold", d.AvailableSpareThreshold, 0)
		}
		if d.HostReadCmds != "" {
			add(7, "Host_Read_Commands", d.HostReadCmds, 0)
		}
		if d.HostWriteCmds != "" {
			add(8, "Host_Write_Commands", d.HostWriteCmds, 0)
		}
		if d.ControllerBusyTime != "" {
			add(9, "Controller_Busy_Time", d.ControllerBusyTime, 0)
		}
		if d.PowerCycles != "" {
			add(10, "Power_Cycles", d.PowerCycles, 0)
		}
		if d.PowerOnHours != "" {
			add(11, "Power_On_Hours", d.PowerOnHours, 0)
		}
		if d.UnsafeShutdowns != "" {
			add(12, "Unsafe_Shutdowns", d.UnsafeShutdowns, 0)
		}
		if d.MediaErrors != "" {
			add(13, "Media_Data_Integrity_Errors", d.MediaErrors, 0)
		}
		if d.ErrorLogEntries != "" {
			add(14, "Error_Log_Entries", d.ErrorLogEntries, 0)
		}
	}
	return d
}

// DevicesFromScan lists devices via `smartctl --scan`.
func DevicesFromScan() ([]string, error) {
	cmd := exec.Command("smartctl", "--scan")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return ParseScanLines(out.String()), nil
}

// ParseScanLines extracts device paths from the output of smartctl --scan.
// Recent smartctl prefixes each line with a timestamp, e.g.
// "00:00:00 /dev/sda [SAT]", so the first token starting with "/" is the device.
func ParseScanLines(text string) []string {
	var devs []string
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		dev := ""
		for _, f := range fields {
			if strings.HasPrefix(f, "/") {
				dev = f
				break
			}
		}
		if dev == "" {
			dev = fields[0]
		}
		devs = append(devs, dev)
	}
	return devs
}

// Mock returns sample DiskInfo data for testing without root.
func Mock(device string, nvme bool) *DiskInfo {
	d := &DiskInfo{Device: device, Model: "SAMPLE DISK", Serial: "MOCK12345",
		Firmware: "1.2.3", Capacity: "1 TB", Health: "PASSED", Temperature: 34}
	if nvme {
		d.Rate = "NVMe"
		d.IsNVMe = true
		d.Endurance = "4%"
		d.DataRead = "28,2 TB"
		d.DataWritten = "39,4 TB"
		d.AvailableSpare = "100%"
		d.AvailableSpareThreshold = "5%"
		d.HostReadCmds = "814 888 291"
		d.HostWriteCmds = "613 104 390"
		d.ControllerBusyTime = "3 761"
		d.PowerCycles = "2 917"
		d.PowerOnHours = "5 403"
		d.UnsafeShutdowns = "301"
		d.MediaErrors = "0"
		d.ErrorLogEntries = "6 554"
		d.Attributes = []Attribute{
			{ID: 1, Name: "Endurance_Percentage", Flag: "0x0032", Value: 96, Worst: 96, Threshold: 0, Type: "NVMe", Raw: "4%"},
			{ID: 2, Name: "Temperature", Flag: "0x0032", Value: 34, Worst: 34, Threshold: 0, Type: "NVMe", Raw: "34°C"},
			{ID: 3, Name: "Data_Units_Read", Flag: "0x0032", Value: 55240836, Worst: 55240836, Threshold: 0, Type: "NVMe", Raw: "55 240 836 [28,2 TB]"},
			{ID: 4, Name: "Data_Units_Written", Flag: "0x0032", Value: 77080093, Worst: 77080093, Threshold: 0, Type: "NVMe", Raw: "77 080 093 [39,4 TB]"},
			{ID: 5, Name: "Available_Spare", Flag: "0x0032", Value: 100, Worst: 100, Threshold: 0, Type: "NVMe", Raw: "100%"},
			{ID: 6, Name: "Available_Spare_Threshold", Flag: "0x0032", Value: 5, Worst: 5, Threshold: 0, Type: "NVMe", Raw: "5%"},
			{ID: 7, Name: "Host_Read_Commands", Flag: "0x0032", Value: 814888291, Worst: 814888291, Threshold: 0, Type: "NVMe", Raw: "814 888 291"},
			{ID: 8, Name: "Host_Write_Commands", Flag: "0x0032", Value: 613104390, Worst: 613104390, Threshold: 0, Type: "NVMe", Raw: "613 104 390"},
			{ID: 9, Name: "Controller_Busy_Time", Flag: "0x0032", Value: 3761, Worst: 3761, Threshold: 0, Type: "NVMe", Raw: "3 761"},
			{ID: 10, Name: "Power_Cycles", Flag: "0x0032", Value: 2917, Worst: 2917, Threshold: 0, Type: "NVMe", Raw: "2 917"},
			{ID: 11, Name: "Power_On_Hours", Flag: "0x0032", Value: 5403, Worst: 5403, Threshold: 0, Type: "NVMe", Raw: "5 403"},
			{ID: 12, Name: "Unsafe_Shutdowns", Flag: "0x0032", Value: 301, Worst: 301, Threshold: 0, Type: "NVMe", Raw: "301"},
			{ID: 13, Name: "Media_Data_Integrity_Errors", Flag: "0x0032", Value: 0, Worst: 0, Threshold: 0, Type: "NVMe", Raw: "0"},
			{ID: 14, Name: "Error_Log_Entries", Flag: "0x0032", Value: 6554, Worst: 6554, Threshold: 0, Type: "NVMe", Raw: "6 554"},
		}
	} else {
		d.Rate = "Solid State Device"
		d.Attributes = []Attribute{
			{ID: 5, Name: "Reallocated_Sector_Ct", Flag: "0x0032", Value: 100, Worst: 100, Threshold: 0, Type: "Old_age", Raw: "4"},
			{ID: 9, Name: "Power_On_Hours", Flag: "0x0032", Value: 100, Worst: 100, Threshold: 0, Type: "Old_age", Raw: "149h"},
			{ID: 170, Name: "Available_Reservd_Space", Flag: "0x0033", Value: 99, Worst: 99, Threshold: 10, Type: "Pre-fail", Raw: "0"},
			{ID: 190, Name: "Airflow_Temperature_Cel", Flag: "0x0032", Value: 40, Worst: 54, Threshold: 0, Type: "Old_age", Raw: "40 (Min/Max 23/54)"},
			{ID: 233, Name: "Media_Wearout_Indicator", Flag: "0x0032", Value: 95, Worst: 95, Threshold: 0, Type: "Old_age", Raw: "0"},
		}
	}
	return d
}

// EnsureSudo re-execs the current process with sudo if not root.
func EnsureSudo() {
	if os.Geteuid() == 0 {
		return
	}
	args := os.Args[1:]
	// strip existing --sudo flag
	argv := []string{}
	for _, a := range args {
		if a == "--sudo" {
			continue
		}
		argv = append(argv, a)
	}
	cmd := exec.Command("sudo", append([]string{os.Args[0]}, argv...)...)
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "sudo failed:", err)
		os.Exit(1)
	}
}
