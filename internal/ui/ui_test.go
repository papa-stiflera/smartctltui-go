package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/papa-stiflera/smartctltui-go/internal/smart"
)

func TestTruncate(t *testing.T) {
	cases := []struct {
		s    string
		n    int
		want string
	}{
		{"hello", 3, "he…"},
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello", 1, ""},
		{"hello", 0, ""},
		{"", 5, ""},
		{"привет", 4, "при…"},
		{"abc", 3, "abc"},
	}
	for _, c := range cases {
		if got := truncate(c.s, c.n); got != c.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", c.s, c.n, got, c.want)
		}
	}
}

func TestPadRow(t *testing.T) {
	if got := padRow("ab", 5); got != "ab   " {
		t.Errorf("padRow(\"ab\", 5) = %q", got)
	}
	if got := padRow("abcd", 3); got != "ab…" {
		t.Errorf("padRow(\"abcd\", 3) = %q", got)
	}
	if got := padRow("abc", 3); got != "abc" {
		t.Errorf("padRow(\"abc\", 3) = %q", got)
	}
	if got := padRow("ab", 0); got != "ab" {
		t.Errorf("padRow(\"ab\", 0) = %q", got)
	}
	if got := padRow("ab", -1); got != "ab" {
		t.Errorf("padRow(\"ab\", -1) = %q", got)
	}
}

func TestAvailWidth(t *testing.T) {
	if got := (model{width: 80}).availWidth(); got != 76 {
		t.Errorf("availWidth(80) = %d, want 76", got)
	}
	if got := (model{width: 40}).availWidth(); got != 60 {
		t.Errorf("availWidth(40) = %d, want 60", got)
	}
}

func TestInfoLine(t *testing.T) {
	if got := infoLine("Device", "", 50); !strings.Contains(got, "—") {
		t.Errorf("infoLine with empty value = %q, want em dash", got)
	}
	if got := infoLine("Model", "WDC WD20EZRX-00D8PB0", 10); !strings.Contains(got, "…") {
		t.Errorf("infoLine with long value = %q, want ellipsis", got)
	}
	if got := infoLine("Serial", "ABC", 50); !strings.Contains(got, "ABC") {
		t.Errorf("infoLine = %q, want to contain ABC", got)
	}
}

func TestHeaderRow(t *testing.T) {
	h := headerRow(60)
	for _, col := range []string{"ID", "Attribute", "Val", "Worst", "Thresh", "Raw"} {
		if !strings.Contains(h, col) {
			t.Errorf("headerRow missing column %q", col)
		}
	}
}

func TestAttrRow(t *testing.T) {
	a := smart.Attribute{ID: 5, Name: "Reallocated_Sector_Ct", Value: 200, Worst: 200, Threshold: 140, Raw: "0"}
	row := attrRow(a, 0, 60)
	if !strings.Contains(row, "Reallocated_Sector_Ct") {
		t.Errorf("attrRow = %q, want to contain name", row)
	}
	if !strings.Contains(row, "200") {
		t.Errorf("attrRow = %q, want to contain value", row)
	}
}

func TestHealthLine(t *testing.T) {
	if got := (model{info: &smart.DiskInfo{Health: "PASSED"}}).healthLine(); !strings.Contains(got, "PASSED") {
		t.Errorf("healthLine PASSED = %q", got)
	}
	if got := (model{info: &smart.DiskInfo{Health: "FAILED"}}).healthLine(); !strings.Contains(got, "FAILED") {
		t.Errorf("healthLine FAILED = %q", got)
	}
	if got := (model{info: &smart.DiskInfo{Health: "UNKNOWN"}}).healthLine(); !strings.Contains(got, "UNKNOWN") {
		t.Errorf("healthline UNKNOWN = %q", got)
	}
}

func TestTempLine(t *testing.T) {
	if got := (model{info: &smart.DiskInfo{Temperature: -1}}).tempLine(); !strings.Contains(got, "—") {
		t.Errorf("tempLine unknown = %q, want em dash", got)
	}
	if got := (model{info: &smart.DiskInfo{Temperature: 30}}).tempLine(); !strings.Contains(got, "30°C") {
		t.Errorf("tempLine 30 = %q", got)
	}
	if got := (model{info: &smart.DiskInfo{Temperature: 50}}).tempLine(); !strings.Contains(got, "50°C") {
		t.Errorf("templine 50 = %q", got)
	}
}

func TestNewModelMock(t *testing.T) {
	m := NewModel(true)
	if len(m.devices) != 3 {
		t.Fatalf("mock devices = %d, want 3", len(m.devices))
	}
	if m.refreshSec != 10 {
		t.Errorf("refreshSec = %d, want 10", m.refreshSec)
	}
	if !m.mock {
		t.Error("mock flag not set")
	}
	if m.devices[0] != "/dev/sda" || m.devices[2] != "/dev/nvme1" {
		t.Errorf("mock devices = %v", m.devices)
	}
}

func TestUpdateWindowSize(t *testing.T) {
	m := NewModel(true)
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	mm := m2.(model)
	if mm.width != 100 || mm.height != 30 {
		t.Errorf("width/height = %d/%d, want 100/30", mm.width, mm.height)
	}
}

func TestUpdateQuitKey(t *testing.T) {
	m := NewModel(true)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if msg := cmd(); msg != tea.Quit() {
		t.Errorf("q key cmd = %v, want quit", msg)
	}
}

func TestUpdateCtrlC(t *testing.T) {
	m := NewModel(true)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if msg := cmd(); msg != tea.Quit() {
		t.Errorf("ctrl-c cmd = %v, want quit", msg)
	}
}

func TestUpdateRefreshKey(t *testing.T) {
	m := NewModel(true)
	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	if !m2.(model).loading {
		t.Error("r key should set loading")
	}
	msg := cmd().(loadedMsg)
	if msg.err != "" || msg.info == nil {
		t.Errorf("refresh cmd msg = %+v, want loaded info", msg)
	}
}

func TestUpdateNavigation(t *testing.T) {
	m := NewModel(true) // 3 devices
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if got := m2.(model).current; got != 0 {
		t.Errorf("up at first = %d, want 0", got)
	}
	m3, _ := m2.(model).Update(tea.KeyMsg{Type: tea.KeyDown})
	if got := m3.(model).current; got != 1 {
		t.Errorf("down from 0 = %d, want 1", got)
	}
	m4, _ := m3.(model).Update(tea.KeyMsg{Type: tea.KeyDown})
	if got := m4.(model).current; got != 2 {
		t.Errorf("down from 1 = %d, want 2", got)
	}
	m5, _ := m4.(model).Update(tea.KeyMsg{Type: tea.KeyDown})
	if got := m5.(model).current; got != 2 {
		t.Errorf("down at last = %d, want 2", got)
	}
}

func TestUpdateEnterCyclesInterval(t *testing.T) {
	m := NewModel(true)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if got := m2.(model).refreshSec; got != 5 {
		t.Errorf("enter 1: refreshSec = %d, want 5", got)
	}
	m3, _ := m2.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if got := m3.(model).refreshSec; got != 30 {
		t.Errorf("enter 2: refreshSec = %d, want 30", got)
	}
	m4, _ := m3.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if got := m4.(model).refreshSec; got != 10 {
		t.Errorf("enter 3: refreshSec = %d, want 10", got)
	}
}

// TestRefreshMockLoadsInfo verifies that refresh() in mock mode delivers a loaded message.
func TestRefreshMockLoadsInfo(t *testing.T) {
	m := NewModel(true)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = m2.(model) // select /dev/nvme0
	cmd := m.refresh()
	if cmd == nil {
		t.Fatal("refresh() returned nil cmd")
	}
	msg := cmd().(loadedMsg)
	if msg.err != "" || msg.info == nil {
		t.Fatalf("refresh msg = %+v, want loaded info", msg)
	}
	if msg.info.Device != "/dev/nvme0" {
		t.Errorf("device = %q, want /dev/nvme0", msg.info.Device)
	}
	if len(msg.info.Attributes) == 0 {
		t.Error("nvme mock has no attributes")
	}
}

func TestViewSmoke(t *testing.T) {
	m := NewModel(true)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = m2.(model) // select /dev/nvme0
	m3, _ := m.Update(m.refresh()())
	m = m3.(model)
	if m.info == nil {
		t.Fatal("view with nil info")
	}
	out := m.View()
	for _, want := range []string{"SMART disk monitor", "/dev/nvme0", "Endurance_Percentage", "PASSED"} {
		if !strings.Contains(out, want) {
			t.Errorf("View() missing %q", want)
		}
	}
}

func TestViewLoading(t *testing.T) {
	m := NewModel(true)
	m.loading = true
	m.info = nil
	if out := m.View(); !strings.Contains(out, "Loading…") {
		t.Errorf("View() loading = %q", out)
	}
}

func TestViewError(t *testing.T) {
	m := NewModel(true)
	m.err = "boom"
	m.info = nil
	if out := m.View(); !strings.Contains(out, "Error: boom") {
		t.Errorf("View() error = %q", out)
	}
}
