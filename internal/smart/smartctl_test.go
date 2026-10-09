package smart

import (
	"strings"
	"testing"
)

// TestSeverity checks the three-tier classification relative to threshold.
func TestSeverity(t *testing.T) {
	cases := []struct {
		name string
		attr Attribute
		want int
	}{
		{"below threshold is bad", Attribute{Value: 10, Threshold: 11}, 2},
		{"equal threshold is not bad", Attribute{Value: 11, Threshold: 11}, 1},
		{"under 80 is warn", Attribute{Value: 79, Threshold: 0}, 1},
		{"at 80 is ok", Attribute{Value: 80, Threshold: 0}, 0},
		{"high is ok", Attribute{Value: 200, Threshold: 0}, 0},
	}
	for _, c := range cases {
		if got := c.attr.Severity(); got != c.want {
			t.Errorf("Severity() = %d, want %d (%s)", got, c.want, c.name)
		}
	}
}

const sataOutput = `smartctl 7.2 2020-04-30 [x86_64-linux-gnu]
Copyright (C) 2002-20, Bruce Allen, Christian Brueckner
All rights reserved.

=== START OF INFORMATION SECTION ===
Model Family:     Western Digital Green
Device Model:     WDC WD20EZRX-00D8PB0
Serial Number:    WD-WMC4M0875073
Firmware Version: 80.00A80
User Capacity:    2,000,398,934,016 bytes [2.00 TB]
Sector Sizes:     512 bytes logical, 4096 bytes physical
Device is:        In smartctl database [for details use: -P show]
ATA Version is:   ACS-2 (minor revision not indicated)
SATA Version is:  SATA 3.0, 6.0 Gb/s (current: 6.0 Gb/s)
Local Time is:    Mon Oct  3 13:16:17 2016 CEST
SMART support is: Available - device has SMART capability.
SMART support is: Enabled

=== START OF READ SMART DATA SECTION ===
SMART overall-health self-assessment test result: PASSED
Power Cycle Min/Max Temperature:        13/41 Celsius

General SMART Values:
Offline data collection status: (0x00) Offline data collection activity was never started.
Auto Offline Data Collection: Disabled.
Self-test execution status: (0x00) The previous self-test routine completed without error or no previous self-test has ever been run.
Total time to complete offline collection: (65535) seconds.

SMART Attributes Data Structure revision number: 16
Vendor Specific SMART Attributes with Thresholds:
ID# ATTRIBUTE_NAME          FLAG     VALUE WORST THRESH TYPE      UPDATED  WHEN_FAILED RAW_VALUE
  5 Reallocated_Sector_Ct    0x0033   200   200   140    Pre-fail  Always       -       0
  9 Power_On_Hours           0x0032    87    87    000    Old_age   Always       -       58412
 12 Power_Cycle_Count        0x0032   100   100    000    Old_age   Always       -       1164
194 Temperature_Celsius      0x0022   100   100    000    Old_age   Always       -       36 (Min/Max 13/41)
195 Hardware_ECC_Recovered   0x000a   200   200    000    Old_age   Always       -       0
197 Current_Pending_Sector   0x0022   100   100    000    Old_age   Always       -       8
`

const nvmeOutput = `smartctl 7.1 2019-12-30 [x86_64-linux-gnu]
Copyright (C) 2002-19, Bruce Allen, Christian Brueckner
All rights reserved.

=== START OF INFORMATION SECTION ===
Model Family:   Unknown (based on SN prefix)
Device Model:   Samsung SSD 970 EVO 500GB
Serial Number:  S1HNNR4B000000
Firmware:       RWB20100
PCIe Generation: 3.2
Local Capacity: 500,107,862,400 bytes [500 GB]
NVMe Version:   1.3

=== START OF ADDITIONAL SMART INFORMATION SECTION ===
SMART overall-health self-assessment Test Result: PASSED (0x00)

SMART Extended Information
** NVMe Log 0x02 (Smart Error Log) **
Critical Warning:                     0x00
Temperature:                          41 Celsius
    Current Temperature:                    41 Celsius (41 Celsius)
Power Cycle Min/Max Temperature:        28/48 Celsius
Available Spare:                      100%
Available Spare Threshold:           10%
Percentage Used:                      7%
Data Units Read:               55,240,836 [28.2 TB]
Data Units Written:            77,080,093 [39.4 TB]
Host Read Commands:          814,888,291
Host Write Commands:         613,104,390
Controller Busy Time:                3,761
Power Cycles:                        2,917
Power On Hours:                      5,403
Unsafe Shutdowns:                      301
Media and Data Integrity Errors:          0
Error Information Log Entries:          6,554
`

// TestParseOutputSATA verifies parsing of a real SATA smartctl output.
func TestParseOutputSATA(t *testing.T) {
	d := ParseOutput("/dev/sda", sataOutput)

	if d.Device != "/dev/sda" {
		t.Errorf("Device = %q, want /dev/sda", d.Device)
	}
	if d.Model != "WDC WD20EZRX-00D8PB0" {
		t.Errorf("Model = %q", d.Model)
	}
	if d.Serial != "WD-WMC4M0875073" {
		t.Errorf("Serial = %q", d.Serial)
	}
	if d.Firmware != "80.00A80" {
		t.Errorf("Firmware = %q", d.Firmware)
	}
	if d.Capacity != "2,000,398,934,016 bytes [2.00 TB]" {
		t.Errorf("Capacity = %q", d.Capacity)
	}
	if d.Health != "PASSED" {
		t.Errorf("Health = %q, want PASSED", d.Health)
	}
	if d.IsNVMe {
		t.Error("IsNVMe = true for SATA device")
	}
	if d.Temperature != 36 {
		t.Errorf("Temperature = %d, want 36", d.Temperature)
	}
	if d.TempMin != 13 || d.TempMax != 41 {
		t.Errorf("TempMin/TempMax = %d/%d, want 13/41", d.TempMin, d.TempMax)
	}

	if len(d.Attributes) != 6 {
		t.Fatalf("Attributes = %d, want 6", len(d.Attributes))
	}
	byID := make(map[int]Attribute, len(d.Attributes))
	for _, a := range d.Attributes {
		byID[a.ID] = a
	}
	if a := byID[5]; a.Name != "Reallocated_Sector_Ct" || a.Value != 200 || a.Threshold != 140 || a.Flag != "0x0033" || a.Raw != "0" {
		t.Errorf("attribute 5 = %+v", a)
	}
	if a := byID[9]; a.Name != "Power_On_Hours" || a.Value != 87 || a.Raw != "58412" {
		t.Errorf("attribute 9 = %+v", a)
	}
	if a := byID[194]; a.Name != "Temperature_Celsius" || a.Value != 100 || a.Raw != "36 (Min/Max 13/41)" {
		t.Errorf("attribute 194 = %+v", a)
	}
}

// TestParseOutputSATAFailed verifies the FAILED health verdict.
func TestParseOutputSATAFailed(t *testing.T) {
	text := strings.Replace(sataOutput, "test result: PASSED", "test result: FAILED", 1)
	d := ParseOutput("/dev/sda", text)
	if d.Health != "FAILED" {
		t.Errorf("Health = %q, want FAILED", d.Health)
	}
}

// TestParseOutputNVMe verifies NVMe-specific fields and synthesized attributes.
func TestParseOutputNVMe(t *testing.T) {
	d := ParseOutput("/dev/nvme0", nvmeOutput)

	if !d.IsNVMe {
		t.Error("IsNVMe = false for NVMe device")
	}
	if d.Model != "Samsung SSD 970 EVO 500GB" {
		t.Errorf("Model = %q", d.Model)
	}
	if d.Firmware != "RWB20100" {
		t.Errorf("Firmware = %q", d.Firmware)
	}
	if d.Capacity != "500,107,862,400 bytes [500 GB]" {
		t.Errorf("Capacity = %q", d.Capacity)
	}
	if d.Health != "PASSED" {
		t.Errorf("Health = %q, want PASSED", d.Health)
	}
	if d.Endurance != "7%" {
		t.Errorf("Endurance = %q", d.Endurance)
	}
	if d.Temperature != 41 {
		t.Errorf("Temperature = %d, want 41", d.Temperature)
	}
	if d.TempMin != 28 || d.TempMax != 48 {
		t.Errorf("TempMin/TempMax = %d/%d, want 28/48", d.TempMin, d.TempMax)
	}

	if len(d.Attributes) != 14 {
		t.Fatalf("Attributes = %d, want 14", len(d.Attributes))
	}
	byID := make(map[int]Attribute, len(d.Attributes))
	for _, a := range d.Attributes {
		byID[a.ID] = a
	}
	if a := byID[1]; a.Name != "Endurance_Percentage" || a.Value != 93 {
		t.Errorf("attribute 1 = %+v, want Endurance_Percentage value 93", a)
	}
	if a := byID[2]; a.Name != "Temperature" || a.Value != 41 {
		t.Errorf("attribute 2 = %+v", a)
	}
	if a := byID[5]; a.Name != "Available_Spare" || a.Value != 100 {
		t.Errorf("attribute 5 = %+v", a)
	}
	if a := byID[12]; a.Name != "Unsafe_Shutdowns" || a.Value != 301 {
		t.Errorf("attribute 12 = %+v", a)
	}
}

// TestParseOutputUnknownHealth verifies the fallback for missing health section.
func TestParseOutputUnknownHealth(t *testing.T) {
	d := ParseOutput("/dev/sda", "Device Model: XYZ\nSerial Number: ABC\n")
	if d.Health != "UNKNOWN" {
		t.Errorf("Health = %q, want UNKNOWN", d.Health)
	}
	if d.Model != "XYZ" || d.Serial != "ABC" {
		t.Errorf("Model/Serial = %q/%q", d.Model, d.Serial)
	}
	if len(d.Attributes) != 0 {
		t.Errorf("Attributes = %d, want 0", len(d.Attributes))
	}
}

// TestParseOutputDedupesAttributeIDs verifies duplicate IDs are dropped.
func TestParseOutputDedupesAttributeIDs(t *testing.T) {
	text := sataOutput + "  9 Power_On_Hours           0x0032    90    85    000    Old_age   Always       -       58413\n"
	d := ParseOutput("/dev/sda", text)
	if len(d.Attributes) != 6 {
		t.Fatalf("Attributes = %d, want 6 after dedupe", len(d.Attributes))
	}
	var count int
	for _, a := range d.Attributes {
		if a.ID == 9 {
			count++
		}
	}
	if count != 1 {
		t.Errorf("attribute ID 9 appears %d times, want 1", count)
	}
}

// TestParseScanLines verifies extraction of device paths from --scan output.
func TestParseScanLines(t *testing.T) {
	text := "00:00:00 /dev/nvme0 [NVM [2,500 TB] [built-in]]\n00:00:10 /dev/nvme0n1\n00:01:00 /dev/sda [SAT]\n\n"
	got := ParseScanLines(text)
	want := []string{"/dev/nvme0", "/dev/nvme0n1", "/dev/sda"}
	if len(got) != len(want) {
		t.Fatalf("ParseScanLines = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("device %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestParseScanLinesEmpty(t *testing.T) {
	if got := ParseScanLines(""); len(got) != 0 {
		t.Errorf("ParseScanLines(\"\") = %v, want empty", got)
	}
}

// TestMock verifies the demo mode data shape for both device types.
func TestMock(t *testing.T) {
	sata := Mock("/dev/sda", false)
	if sata.Health != "PASSED" || sata.Model != "SAMPLE DISK" {
		t.Errorf("Mock sata = %s/%s", sata.Health, sata.Model)
	}
	if sata.IsNVMe || len(sata.Attributes) != 5 {
		t.Errorf("Mock sata: IsNVMe=%v, attrs=%d", sata.IsNVMe, len(sata.Attributes))
	}

	nvme := Mock("/dev/nvme0", true)
	if !nvme.IsNVMe || nvme.Rate != "NVMe" || len(nvme.Attributes) != 14 {
		t.Errorf("Mock nvme: IsNVMe=%v, Rate=%s, attrs=%d", nvme.IsNVMe, nvme.Rate, len(nvme.Attributes))
	}
	if nvme.Temperature != 34 {
		t.Errorf("Mock nvme Temperature = %d", nvme.Temperature)
	}
}
