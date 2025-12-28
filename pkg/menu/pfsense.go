package menu

import (
	"fmt"
	"time"

	"github.com/sagostin/ezio-g500/pkg/display"
	"github.com/sagostin/ezio-g500/pkg/eziog500"
	"github.com/sagostin/ezio-g500/pkg/font"
	"github.com/sagostin/ezio-g500/pkg/pfsense"
)

// PfSenseMenuBuilder creates a pre-built menu for pfSense systems.
type PfSenseMenuBuilder struct {
	display *display.Display
	metrics *pfsense.SystemMetrics
}

// NewPfSenseMenuBuilder creates a new pfSense menu builder.
func NewPfSenseMenuBuilder(d *display.Display) *PfSenseMenuBuilder {
	return &PfSenseMenuBuilder{
		display: d,
		metrics: pfsense.NewSystemMetrics(),
	}
}

// Build creates the complete pfSense menu structure.
func (b *PfSenseMenuBuilder) Build() *Menu {
	// Main Menu
	mainMenu := NewMenu("PFSENSE LCD", []MenuItem{})

	// Status submenu
	statusMenu := b.buildStatusMenu()
	mainMenu.AddSubMenu("System Status", statusMenu)

	// Network submenu
	networkMenu := b.buildNetworkMenu()
	mainMenu.AddSubMenu("Network", networkMenu)

	// Display submenu
	displayMenu := b.buildDisplayMenu()
	mainMenu.AddSubMenu("Display", displayMenu)

	// Actions
	mainMenu.AddItem(MenuItem{
		Label: "Refresh",
		Action: func() error {
			return b.showStatus()
		},
	})

	mainMenu.AddItem(MenuItem{
		Label: "Exit Menu",
		Action: func() error {
			return b.showStatus()
		},
	})

	return mainMenu
}

func (b *PfSenseMenuBuilder) buildStatusMenu() *Menu {
	menu := NewMenu("SYSTEM STATUS", []MenuItem{})

	menu.AddItem(MenuItem{
		Label: "CPU",
		Value: func() string {
			m, _ := b.metrics.GetMetrics()
			return fmt.Sprintf("%.1f%%", m.CPU)
		},
	})

	menu.AddItem(MenuItem{
		Label: "Memory",
		Value: func() string {
			m, _ := b.metrics.GetMetrics()
			if m.MemTotal > 0 {
				pct := float64(m.MemUsed) / float64(m.MemTotal) * 100
				return fmt.Sprintf("%.1f%%", pct)
			}
			return "N/A"
		},
	})

	menu.AddItem(MenuItem{
		Label: "Load",
		Value: func() string {
			m, _ := b.metrics.GetMetrics()
			return fmt.Sprintf("%.2f", m.LoadAvg[0])
		},
	})

	menu.AddItem(MenuItem{
		Label: "Uptime",
		Value: func() string {
			m, _ := b.metrics.GetMetrics()
			return formatUptime(m.Uptime)
		},
	})

	menu.AddItem(MenuItem{
		Label: "View Full Status",
		Action: func() error {
			return b.showStatus()
		},
	})

	return menu
}

func (b *PfSenseMenuBuilder) buildNetworkMenu() *Menu {
	menu := NewMenu("NETWORK", []MenuItem{})

	// Add interface items dynamically
	m, _ := b.metrics.GetMetrics()
	for _, iface := range m.Interfaces {
		ifaceCopy := iface // Capture for closure
		menu.AddItem(MenuItem{
			Label: ifaceCopy.Name,
			Value: func() string {
				if ifaceCopy.IP != "" {
					return ifaceCopy.IP
				}
				return ifaceCopy.Status
			},
		})
	}

	menu.AddItem(MenuItem{
		Label: "View All Interfaces",
		Action: func() error {
			return b.showNetworkStatus()
		},
	})

	return menu
}

func (b *PfSenseMenuBuilder) buildDisplayMenu() *Menu {
	menu := NewMenu("DISPLAY", []MenuItem{})

	// Backlight control
	backlightLevels := []struct {
		name  string
		level byte
	}{
		{"Backlight: Off", 0},
		{"Backlight: Low", 64},
		{"Backlight: Medium", 128},
		{"Backlight: High", 200},
		{"Backlight: Max", 255},
	}

	for _, bl := range backlightLevels {
		level := bl.level
		menu.AddItem(MenuItem{
			Label: bl.name,
			Action: func() error {
				return b.display.SetBacklight(level)
			},
		})
	}

	// LED controls
	ledMenu := NewMenu("LED CONTROL", []MenuItem{})
	for ledNum := 1; ledNum <= 3; ledNum++ {
		led := eziog500.LED(ledNum - 1)
		ledMenu.AddItem(MenuItem{
			Label: fmt.Sprintf("LED %d: Off", ledNum),
			Action: func() error {
				return b.display.SetLED(led, eziog500.LEDOff)
			},
		})
		ledMenu.AddItem(MenuItem{
			Label: fmt.Sprintf("LED %d: Red", ledNum),
			Action: func() error {
				return b.display.SetLED(led, eziog500.LEDRed)
			},
		})
		ledMenu.AddItem(MenuItem{
			Label: fmt.Sprintf("LED %d: Green", ledNum),
			Action: func() error {
				return b.display.SetLED(led, eziog500.LEDGreen)
			},
		})
	}
	menu.AddSubMenu("LED Control", ledMenu)

	return menu
}

func (b *PfSenseMenuBuilder) showStatus() error {
	m, err := b.metrics.GetMetrics()
	if err != nil {
		return err
	}

	status := &display.SystemStatus{
		Hostname: m.Hostname,
		Uptime:   m.Uptime,
		CPU:      m.CPU,
		MemUsed:  m.MemUsed,
		MemTotal: m.MemTotal,
		LoadAvg:  fmt.Sprintf("%.2f %.2f %.2f", m.LoadAvg[0], m.LoadAvg[1], m.LoadAvg[2]),
	}

	for _, iface := range m.Interfaces {
		if iface.IP != "" && iface.Status == "up" {
			status.IPAddress = iface.IP
			break
		}
	}

	template := status.ToTemplate()
	return template.Render(b.display)
}

func (b *PfSenseMenuBuilder) showNetworkStatus() error {
	m, err := b.metrics.GetMetrics()
	if err != nil {
		return err
	}

	var infos []display.InterfaceInfo
	for _, iface := range m.Interfaces {
		infos = append(infos, display.InterfaceInfo{
			Name:   iface.Name,
			Status: iface.Status,
			IP:     iface.IP,
			RxRate: pfsense.FormatBytes(iface.RxBytes),
			TxRate: pfsense.FormatBytes(iface.TxBytes),
		})
	}

	netStatus := &display.NetworkStatus{Interfaces: infos}
	template := netStatus.ToTemplate()
	return template.Render(b.display)
}

func formatUptime(d time.Duration) string {
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

// QuickInfoScreen shows a quick info screen with key metrics.
type QuickInfoScreen struct {
	display *display.Display
	metrics *pfsense.SystemMetrics
}

// NewQuickInfoScreen creates a quick info screen.
func NewQuickInfoScreen(d *display.Display) *QuickInfoScreen {
	return &QuickInfoScreen{
		display: d,
		metrics: pfsense.NewSystemMetrics(),
	}
}

// Show displays the quick info screen.
func (q *QuickInfoScreen) Show() error {
	m, err := q.metrics.GetMetrics()
	if err != nil {
		return err
	}

	fb := q.display.FrameBuffer()
	fb.Clear()

	f := font.BuiltinFont
	lh := f.Height()

	// Title
	font.RenderTextInverted(fb, f, 0, 0, m.Hostname)

	// Metrics with progress bars
	y := lh

	// CPU bar
	font.RenderText(fb, f, 0, y, fmt.Sprintf("CPU: %.0f%%", m.CPU))
	drawMiniBar(fb, 70, y, 54, lh-2, m.CPU/100)
	y += lh

	// Memory bar
	memPct := float64(m.MemUsed) / float64(m.MemTotal) * 100
	font.RenderText(fb, f, 0, y, fmt.Sprintf("MEM: %.0f%%", memPct))
	drawMiniBar(fb, 70, y, 54, lh-2, memPct/100)
	y += lh

	// Load
	font.RenderText(fb, f, 0, y, fmt.Sprintf("LOAD: %.2f %.2f %.2f", m.LoadAvg[0], m.LoadAvg[1], m.LoadAvg[2]))
	y += lh

	// Uptime
	font.RenderText(fb, f, 0, y, fmt.Sprintf("UP: %s", formatUptime(m.Uptime)))
	y += lh

	// IP Address
	for _, iface := range m.Interfaces {
		if iface.IP != "" && iface.Status == "up" {
			font.RenderText(fb, f, 0, y, fmt.Sprintf("IP: %s", iface.IP))
			break
		}
	}

	return q.display.Update()
}

func drawMiniBar(fb *eziog500.FrameBuffer, x, y, w, h int, pct float64) {
	if pct < 0 {
		pct = 0
	}
	if pct > 1 {
		pct = 1
	}

	// Border
	fb.DrawRect(x, y, w, h, true)

	// Fill
	fillW := int(float64(w-2) * pct)
	if fillW > 0 {
		fb.FillRect(x+1, y+1, fillW, h-2, true)
	}
}
