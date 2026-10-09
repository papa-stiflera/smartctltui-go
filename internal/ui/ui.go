package ui

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/papa-stiflera/smartctltui-go/internal/smart"
)

// ---------- styles ----------

var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7c3aed")).
			Bold(true).
			Padding(0, 1)
	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6b7280")).
			Padding(0, 1)

	healthOK   = lipgloss.NewStyle().Foreground(lipgloss.Color("#34d399")).Bold(true)
	healthBad  = lipgloss.NewStyle().Foreground(lipgloss.Color("#f87171")).Bold(true)
	healthWarn = lipgloss.NewStyle().Foreground(lipgloss.Color("#fbbf24")).Bold(true)
	tempHot    = lipgloss.NewStyle().Foreground(lipgloss.Color("#fb923c"))
	tempOK     = lipgloss.NewStyle().Foreground(lipgloss.Color("#60a5fa"))

	selBg     = lipgloss.Color("#1e293b")
	selFg     = lipgloss.Color("#7c3aed")
	selBgDim  = lipgloss.Color("#3b4256")
	selFgDim  = lipgloss.Color("#e5e7eb")
	rowBg     = lipgloss.Color("#1f2937")
	rowBgAlt  = lipgloss.Color("#111827")
	borderCol = lipgloss.Color("#374151")

	attrValOK = lipgloss.NewStyle().Foreground(lipgloss.Color("#34d399"))
	attrValW  = lipgloss.NewStyle().Foreground(lipgloss.Color("#fbbf24"))
	attrValB  = lipgloss.NewStyle().Foreground(lipgloss.Color("#f87171")).Bold(true)

	statusBar = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6b7280")).
			Background(lipgloss.Color("#111827"))
)

// ---------- messages ----------

type loadedMsg struct {
	device string
	info   *smart.DiskInfo
	err    string
}

type refreshMsg struct{}

// ---------- model ----------

type model struct {
	devices    []string
	current    int
	info       *smart.DiskInfo
	err        string
	loading    bool
	lastUpd    time.Time
	width      int
	height     int
	refreshSec int
	mock       bool
}

// Init implements tea.Model.
func (m model) Init() tea.Cmd {
	return m.refresh()
}

// Start loads the current device.
func (m model) refresh() tea.Cmd {
	if len(m.devices) == 0 {
		return nil
	}
	dev := m.devices[m.current]
	if m.mock {
		nvme := strings.HasPrefix(dev, "/dev/nvme")
		d := smart.Mock(dev, nvme)
		return func() tea.Msg {
			return loadedMsg{device: dev, info: d}
		}
	}
	return func() tea.Msg {
		d, err := smart.Run(dev)
		if err != nil {
			return loadedMsg{device: dev, err: err.Error()}
		}
		return loadedMsg{device: dev, info: d}
	}
}

// Update implements tea.Model.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case loadedMsg:
		if msg.err != "" {
			m.err = msg.err
			m.info = nil
		} else {
			m.info = msg.info
			m.err = ""
		}
		m.loading = false
		m.lastUpd = time.Now()
		return m, tea.Tick(time.Duration(m.refreshSec)*time.Second, func(t time.Time) tea.Msg {
			return refreshMsg{}
		})

	case refreshMsg:
		return m, m.refresh()

	case tea.KeyMsg:
		if msg.Type == tea.KeyRunes {
			if len(msg.Runes) == 1 {
				switch msg.Runes[0] {
				case 'q':
					return m, func() tea.Msg { return tea.Quit() }
				case 'r':
					m.loading = true
					return m, m.refresh()
				}
			}
			return m, nil
		}
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, func() tea.Msg { return tea.Quit() }
		case tea.KeyUp, tea.KeyPgUp:
			if m.current > 0 {
				m.current--
			}
			m.loading = true
			return m, m.refresh()
		case tea.KeyDown, tea.KeyPgDown:
			if m.current < len(m.devices)-1 {
				m.current++
			}
			m.loading = true
			return m, m.refresh()
		case tea.KeyEnter:
			switch m.refreshSec {
			case 10:
				m.refreshSec = 5
			case 5:
				m.refreshSec = 30
			default:
				m.refreshSec = 10
			}
			return m, m.refresh()
		}
	}
	return m, nil
}

// ---------- view ----------

// availWidth returns the usable inner width (default 60 before first resize).
func (m model) availWidth() int {
	if m.width > 44 {
		return m.width - 4
	}
	return 60
}

func (m model) View() string {
	var b strings.Builder
	w := m.availWidth()

	// header
	b.WriteString(titleStyle.Render(" SMARTCTLTUI "))
	b.WriteString("\n")
	b.WriteString(subtitleStyle.Render(fmt.Sprintf(" SMART disk monitor  ·  %d device(s)  ·  auto-refresh %ds",
		len(m.devices), m.refreshSec)))
	b.WriteString("\n")
	b.WriteString("\n")

	if len(m.devices) == 0 && m.err != "" {
		b.WriteString(healthBad.Render(" " + m.err))
		b.WriteString("\n")
		return b.String()
	}

	// device list
	for i, dev := range m.devices {
		line := fmt.Sprintf(" %2d  %s", i+1, dev)
		line = padRow(line, w)
		if i == m.current {
			line = lipgloss.NewStyle().
				Foreground(selFg).
				Background(selBg).
				Bold(true).
				Render(line)
		} else {
			line = lipgloss.NewStyle().
				Foreground(selFgDim).
				Background(selBgDim).
				Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("\n")

	if m.loading {
		b.WriteString(subtitleStyle.Render("  Loading…"))
		b.WriteString("\n")
		return b.String()
	}
	if m.err != "" {
		b.WriteString(healthBad.Render(fmt.Sprintf("  Error: %s", m.err)))
		b.WriteString("\n")
		return b.String()
	}
	if m.info == nil {
		return b.String()
	}
	info := m.info

	// info card
	border := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(borderCol)
	card := border.Render(lipgloss.JoinVertical(lipgloss.Top,
		lipgloss.JoinHorizontal(lipgloss.Left,
			lipgloss.NewStyle().Width(24).Render(infoLine("Device", info.Device, w-12)),
			lipgloss.NewStyle().Render(infoLine("Model", info.Model, w-37)),
		),
		lipgloss.JoinHorizontal(lipgloss.Left,
			lipgloss.NewStyle().Width(24).Render(infoLine("Capacity", info.Capacity, w-12)),
			lipgloss.NewStyle().Render(infoLine("Serial", info.Serial, w-37)),
		),
		lipgloss.JoinHorizontal(lipgloss.Left,
			lipgloss.NewStyle().Width(24).Render(infoLine("Firmware", info.Firmware, w-12)),
			lipgloss.NewStyle().Render(infoLine("Type", info.Rate, w-37)),
		),
		m.healthLine(),
		m.tempLine(),
	))
	b.WriteString(card)
	b.WriteString("\n")
	b.WriteString("\n")

	// attributes table
	b.WriteString(subtitleStyle.Render(" SMART attributes"))
	b.WriteString("\n")
	b.WriteString(headerRow(w))
	b.WriteString("\n")
	for i, a := range info.Attributes {
		b.WriteString(attrRow(a, i, w))
		b.WriteString("\n")
	}

	// status bar
	b.WriteString("\n")
	statusText := fmt.Sprintf(
		" up/down: select device · R: refresh · PgUp/PgDn: navigate · Enter: interval · Q: quit  |  updated %s",
		m.lastUpd.Format("15:04:05"))
	b.WriteString(statusBar.Render(padRow(statusText, m.width)))

	return b.String()
}

func infoLine(k, v string, max int) string {
	if v == "" {
		v = "—"
	}
	v = truncate(v, max)
	return lipgloss.NewStyle().Render(lipgloss.JoinHorizontal(lipgloss.Left,
		lipgloss.NewStyle().Width(11).Foreground(lipgloss.Color("#6b7280")).Render(k),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#d1d5db")).Render(v)))
}

func (m model) healthLine() string {
	var s string
	switch m.info.Health {
	case "PASSED":
		s = healthOK.Render("✓ PASSED")
	case "FAILED":
		s = healthBad.Render("✗ FAILED")
	default:
		s = healthWarn.Render("? UNKNOWN")
	}
	return lipgloss.NewStyle().Render("  Health: " + s)
}

func (m model) tempLine() string {
	if m.info.Temperature < 0 {
		return "  Temp: —"
	}
	style := tempOK
	if m.info.Temperature > 45 {
		style = tempHot
	}
	return style.Render(fmt.Sprintf("  Temp: %d°C", m.info.Temperature))
}

func headerRow(w int) string {
	h := lipgloss.NewStyle().Foreground(lipgloss.Color("#6b7280"))
	return h.Render(padRow(fmt.Sprintf(
		" %-4s %-24s %-5s %-5s %-5s %-8s",
		"ID", "Attribute", "Val", "Worst", "Thresh", "Raw"), w))
}

// truncate shortens s to n visible chars (runes), appending "…" when cut.
func truncate(s string, n int) string {
	if n <= 1 {
		return ""
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

// padRow pads s with spaces to width w visible chars (truncating if too long).
func padRow(s string, w int) string {
	if w <= 0 {
		return s
	}
	if len([]rune(s)) >= w {
		return truncate(s, w)
	}
	return s + strings.Repeat(" ", w-len([]rune(s)))
}

func attrRow(a smart.Attribute, i int, w int) string {
	bg := rowBg
	if i%2 == 1 {
		bg = rowBgAlt
	}
	valStyle := attrValOK
	switch a.Severity() {
	case 1:
		valStyle = attrValW
	case 2:
		valStyle = attrValB
	}
	// Build the row and pad it to the terminal width so the background
	// fills the full line (preventing long raw values from overflowing).
	row := fmt.Sprintf(" %-4d %-24s %4d %4d %4d %-8s",
		a.ID, a.Name, a.Value, a.Worst, a.Threshold, a.Raw)
	row = padRow(row, w)
	return lipgloss.NewStyle().Background(bg).Render(valStyle.Render(row))
}

// Run is the program entry point.
func Run(mock bool) {
	p := tea.NewProgram(NewModel(mock), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "bubbletea:", err)
		os.Exit(1)
	}
}

// NewModel builds the initial model and discovers devices.
func NewModel(mock bool) model {
	m := model{refreshSec: 10, mock: mock, width: 80, height: 24}
	if mock {
		m.devices = []string{"/dev/sda", "/dev/nvme0", "/dev/nvme1"}
		return m
	}
	devs, err := smart.DevicesFromScan()
	if err != nil || len(devs) == 0 {
		m.err = fmt.Sprintf("failed to get device list: %v", err)
	} else {
		m.devices = devs
	}
	return m
}
