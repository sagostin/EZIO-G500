package display

import (
	"fmt"
	"time"

	"github.com/sagostin/ezio-g500/pkg/font"
)

// StatusLine represents a label-value pair for status display.
type StatusLine struct {
	Label string
	Value string
}

// StatusTemplate displays key-value pairs in a structured format.
type StatusTemplate struct {
	Title    string
	Lines    []StatusLine
	MaxWidth int // Maximum characters per line (0 = auto)
}

// Render draws the status template to the display.
func (t *StatusTemplate) Render(d *Display) error {
	d.fb.Clear()

	y := 0
	fontHeight := d.font.Height()

	// Render title if present
	if t.Title != "" {
		// Draw title with inverted style
		font.RenderTextInverted(d.fb, d.font, 0, y, t.Title)
		y += fontHeight
	}

	// Render status lines
	for _, line := range t.Lines {
		if y >= 64 {
			break // Don't overflow
		}

		// Format: "Label: Value"
		text := line.Label
		if line.Value != "" {
			text = line.Label + ": " + line.Value
		}

		font.RenderText(d.fb, d.font, 0, y, text)
		y += fontHeight
	}

	return d.Update()
}

// SystemStatus is a predefined template for system information.
type SystemStatus struct {
	Hostname  string
	Uptime    time.Duration
	CPU       float64
	MemUsed   uint64
	MemTotal  uint64
	LoadAvg   string
	IPAddress string
}

// ToTemplate converts SystemStatus to a StatusTemplate.
func (s *SystemStatus) ToTemplate() *StatusTemplate {
	lines := []StatusLine{}

	if s.Hostname != "" {
		lines = append(lines, StatusLine{Label: "Host", Value: s.Hostname})
	}

	if s.Uptime > 0 {
		lines = append(lines, StatusLine{Label: "Up", Value: formatDuration(s.Uptime)})
	}

	if s.CPU > 0 {
		lines = append(lines, StatusLine{Label: "CPU", Value: fmt.Sprintf("%.1f%%", s.CPU)})
	}

	if s.MemTotal > 0 {
		memPct := float64(s.MemUsed) / float64(s.MemTotal) * 100
		lines = append(lines, StatusLine{Label: "Mem", Value: fmt.Sprintf("%.1f%%", memPct)})
	}

	if s.LoadAvg != "" {
		lines = append(lines, StatusLine{Label: "Load", Value: s.LoadAvg})
	}

	if s.IPAddress != "" {
		lines = append(lines, StatusLine{Label: "IP", Value: s.IPAddress})
	}

	return &StatusTemplate{
		Title: "PFSENSE STATUS",
		Lines: lines,
	}
}

// NetworkStatus displays network interface information.
type NetworkStatus struct {
	Interfaces []InterfaceInfo
}

// InterfaceInfo contains network interface details.
type InterfaceInfo struct {
	Name   string
	Status string
	IP     string
	RxRate string
	TxRate string
}

// ToTemplate converts NetworkStatus to a StatusTemplate.
func (n *NetworkStatus) ToTemplate() *StatusTemplate {
	lines := []StatusLine{}

	for _, iface := range n.Interfaces {
		// First line: interface name and status
		status := iface.Status
		if iface.IP != "" {
			status = iface.IP
		}
		lines = append(lines, StatusLine{Label: iface.Name, Value: status})

		// Second line: traffic if available
		if iface.RxRate != "" || iface.TxRate != "" {
			traffic := fmt.Sprintf("R:%s T:%s", iface.RxRate, iface.TxRate)
			lines = append(lines, StatusLine{Label: " ", Value: traffic})
		}
	}

	return &StatusTemplate{
		Title: "NETWORK",
		Lines: lines,
	}
}

// formatDuration formats a duration in a compact form.
func formatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh", days, hours)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}

// ProgressBar renders a progress bar at the specified position.
type ProgressBar struct {
	X, Y   int
	Width  int
	Height int
}

// Render draws a progress bar with the given percentage (0-100).
func (p *ProgressBar) Render(d *Display, percent float64) {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	// Draw border
	d.fb.DrawRect(p.X, p.Y, p.Width, p.Height, true)

	// Fill based on percentage
	fillWidth := int(float64(p.Width-2) * percent / 100)
	if fillWidth > 0 {
		d.fb.FillRect(p.X+1, p.Y+1, fillWidth, p.Height-2, true)
	}
}

// MultiScreen manages multiple display screens that can be cycled.
type MultiScreen struct {
	screens  []func(*Display) error
	current  int
	interval time.Duration
}

// NewMultiScreen creates a multi-screen manager.
func NewMultiScreen(interval time.Duration) *MultiScreen {
	return &MultiScreen{
		screens:  make([]func(*Display) error, 0),
		interval: interval,
	}
}

// AddScreen adds a screen rendering function.
func (m *MultiScreen) AddScreen(render func(*Display) error) {
	m.screens = append(m.screens, render)
}

// RenderCurrent renders the current screen.
func (m *MultiScreen) RenderCurrent(d *Display) error {
	if len(m.screens) == 0 {
		return nil
	}
	return m.screens[m.current](d)
}

// Next advances to the next screen.
func (m *MultiScreen) Next() {
	if len(m.screens) > 0 {
		m.current = (m.current + 1) % len(m.screens)
	}
}

// Previous goes to the previous screen.
func (m *MultiScreen) Previous() {
	if len(m.screens) > 0 {
		m.current = (m.current - 1 + len(m.screens)) % len(m.screens)
	}
}

// SetScreen sets the current screen index.
func (m *MultiScreen) SetScreen(index int) {
	if index >= 0 && index < len(m.screens) {
		m.current = index
	}
}
