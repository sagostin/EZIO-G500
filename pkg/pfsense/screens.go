// Package pfsense provides status screens for the LCD display.
package pfsense

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/sagostin/ezio-g500/pkg/display"
	"github.com/sagostin/ezio-g500/pkg/eziog500"
	"github.com/sagostin/ezio-g500/pkg/font"
)

// StatusScreen represents a single status display screen.
type StatusScreen interface {
	Render(d *display.Display, metrics *Metrics) error
	Name() string
}

// StatusDaemon manages rotating status screens.
type StatusDaemon struct {
	display        *display.Display
	metrics        *SystemMetrics
	screens        []StatusScreen
	currentScreen  int
	updateInterval time.Duration
	rotateInterval time.Duration
	history        *MetricsHistory
	frameCount     int
	lastIfaceBytes map[string]ifaceBytes
	lastSampleTime time.Time
	ifaceRates     map[string]ifaceRate
}

type ifaceBytes struct{ tx, rx uint64 }
type ifaceRate struct{ txRate, rxRate float64 }

// MetricsHistory stores historical data.
type MetricsHistory struct {
	CPUHistory     []float64
	TxRateHistory  []float64
	RxRateHistory  []float64
	maxSamples     int
	lastTxBytes    uint64
	lastRxBytes    uint64
	lastSampleTime time.Time
}

func NewMetricsHistory(maxSamples int) *MetricsHistory {
	return &MetricsHistory{
		maxSamples:    maxSamples,
		CPUHistory:    make([]float64, 0, maxSamples),
		TxRateHistory: make([]float64, 0, maxSamples),
		RxRateHistory: make([]float64, 0, maxSamples),
	}
}

func (h *MetricsHistory) AddSample(m *Metrics) {
	h.CPUHistory = append(h.CPUHistory, m.CPU)
	if len(h.CPUHistory) > h.maxSamples {
		h.CPUHistory = h.CPUHistory[1:]
	}
	now := time.Now()
	var totalTx, totalRx uint64
	for _, iface := range m.Interfaces {
		totalTx += iface.TxBytes
		totalRx += iface.RxBytes
	}
	if !h.lastSampleTime.IsZero() {
		elapsed := now.Sub(h.lastSampleTime).Seconds()
		if elapsed > 0 {
			h.TxRateHistory = append(h.TxRateHistory, float64(totalTx-h.lastTxBytes)/elapsed)
			h.RxRateHistory = append(h.RxRateHistory, float64(totalRx-h.lastRxBytes)/elapsed)
			if len(h.TxRateHistory) > h.maxSamples {
				h.TxRateHistory = h.TxRateHistory[1:]
			}
			if len(h.RxRateHistory) > h.maxSamples {
				h.RxRateHistory = h.RxRateHistory[1:]
			}
		}
	}
	h.lastTxBytes, h.lastRxBytes, h.lastSampleTime = totalTx, totalRx, now
}

func NewStatusDaemon(d *display.Display, updateInterval, rotateInterval time.Duration) *StatusDaemon {
	daemon := &StatusDaemon{
		display:        d,
		metrics:        NewSystemMetrics(),
		updateInterval: updateInterval,
		rotateInterval: rotateInterval,
		history:        NewMetricsHistory(60),
		lastIfaceBytes: make(map[string]ifaceBytes),
		ifaceRates:     make(map[string]ifaceRate),
	}

	// Multiple screens with better organization
	daemon.screens = []StatusScreen{
		&LogoScreen{},
		&CPUScreen{},
		&MemoryScreen{},
		&InterfaceScreen{},
		&WANTrafficScreen{daemon: daemon},
		&TunnelTrafficScreen{daemon: daemon},
		&LANTrafficScreen{daemon: daemon},
	}
	return daemon
}

func (sd *StatusDaemon) AddScreen(s StatusScreen) { sd.screens = append(sd.screens, s) }

func (sd *StatusDaemon) Run() error {
	animTicker := time.NewTicker(100 * time.Millisecond)
	rotateTicker := time.NewTicker(sd.rotateInterval)
	defer animTicker.Stop()
	defer rotateTicker.Stop()
	sd.update()
	for {
		select {
		case <-animTicker.C:
			sd.frameCount++
			sd.update()
		case <-rotateTicker.C:
			sd.currentScreen = (sd.currentScreen + 1) % len(sd.screens)
		}
	}
}

func (sd *StatusDaemon) update() error {
	metrics, err := sd.metrics.GetMetrics()
	if err != nil {
		return err
	}
	sd.history.AddSample(metrics)

	// Calculate per-interface rates
	now := time.Now()
	if !sd.lastSampleTime.IsZero() {
		elapsed := now.Sub(sd.lastSampleTime).Seconds()
		if elapsed > 0 {
			for _, iface := range metrics.Interfaces {
				if last, ok := sd.lastIfaceBytes[iface.Name]; ok {
					sd.ifaceRates[iface.Name] = ifaceRate{
						txRate: float64(iface.TxBytes-last.tx) / elapsed,
						rxRate: float64(iface.RxBytes-last.rx) / elapsed,
					}
				}
				sd.lastIfaceBytes[iface.Name] = ifaceBytes{tx: iface.TxBytes, rx: iface.RxBytes}
			}
		}
	}
	sd.lastSampleTime = now

	// Update LEDs based on thresholds
	// LED1 (top) = Info: green when on logo screen, off otherwise
	// LED2 (middle) = Health: green=good, orange=warning, red=critical
	// LED3 (bottom) = Home indicator: green on logo screen
	sd.updateLEDs(metrics)

	if sd.currentScreen < len(sd.screens) {
		// Pass frame to all screens
		switch s := sd.screens[sd.currentScreen].(type) {
		case *LogoScreen:
			s.frame = sd.frameCount
		case *InterfaceScreen:
			s.frame = sd.frameCount
		case *WANTrafficScreen:
			s.frame = sd.frameCount
		case *TunnelTrafficScreen:
			s.frame = sd.frameCount
		case *LANTrafficScreen:
			s.frame = sd.frameCount
		}
		return sd.screens[sd.currentScreen].Render(sd.display, metrics)
	}
	return nil
}

func (sd *StatusDaemon) GetIfaceRate(name string) (tx, rx float64) {
	if r, ok := sd.ifaceRates[name]; ok {
		return r.txRate, r.rxRate
	}
	return 0, 0
}

// updateLEDs sets LED colors based on system metrics thresholds
func (sd *StatusDaemon) updateLEDs(m *Metrics) {
	dev := sd.display.Device()
	if dev == nil {
		return
	}

	memPct := float64(m.MemUsed) / float64(m.MemTotal) * 100

	// LED1 (top) - Info indicator: shows current screen type
	// Green = logo/overview, Orange = traffic, Off = other
	isLogo := sd.currentScreen == 0
	isTraffic := sd.currentScreen >= 4 && sd.currentScreen <= 6
	if isLogo {
		dev.SetLED(eziog500.LED1, eziog500.LEDGreen)
	} else if isTraffic {
		dev.SetLED(eziog500.LED1, eziog500.LEDOrange)
	} else {
		dev.SetLED(eziog500.LED1, eziog500.LEDOff)
	}

	// LED2 (middle) - Health indicator
	// Green = all good (CPU<70%, MEM<80%)
	// Orange = warning (CPU 70-90% or MEM 80-90%)
	// Red = critical (CPU>90% or MEM>90%)
	if m.CPU > 90 || memPct > 90 {
		dev.SetLED(eziog500.LED2, eziog500.LEDRed)
	} else if m.CPU > 70 || memPct > 80 {
		dev.SetLED(eziog500.LED2, eziog500.LEDOrange)
	} else {
		dev.SetLED(eziog500.LED2, eziog500.LEDGreen)
	}

	// LED3 (bottom) - Home indicator: green on logo screen
	if isLogo {
		dev.SetLED(eziog500.LED3, eziog500.LEDGreen)
	} else {
		dev.SetLED(eziog500.LED3, eziog500.LEDOff)
	}
}

// ========== HELPERS ==========

func scrollText(text string, maxLen, frame int) string {
	if len(text) <= maxLen {
		return text
	}

	// Add spacing for seamless loop
	padded := text + "    "
	textLen := len(padded)

	// Scroll speed: every 3 frames (300ms per character) - readable but smooth
	// With a pause at the beginning
	pauseFrames := 10 // Pause for 1 second at start
	cycleLen := textLen + pauseFrames

	adjustedFrame := frame % cycleLen

	// Pause at the beginning before scrolling
	if adjustedFrame < pauseFrames {
		return text[:maxLen]
	}

	// Scroll position
	pos := (adjustedFrame - pauseFrames) / 3
	if pos >= textLen {
		pos = pos % textLen
	}

	// Extract the visible portion, wrapping around
	result := make([]byte, maxLen)
	for i := 0; i < maxLen; i++ {
		idx := (pos + i) % textLen
		result[i] = padded[idx]
	}
	return string(result)
}

func drawBar(fb *eziog500.FrameBuffer, x, y, w, h int, pct float64) {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	fb.DrawRect(x, y, w, h, true)
	fill := int(float64(w-2) * pct / 100)
	if fill > 0 {
		fb.FillRect(x+1, y+1, fill, h-2, true)
	}
}

// draw3DPF draws a rotating 3D "pf" logo
func draw3DPF(fb *eziog500.FrameBuffer, centerX, centerY int, frame int) {
	angle := float64(frame) * 0.08
	cos := math.Cos(angle)
	sin := math.Sin(angle)

	// Scale factor for 3D effect
	scale := 1.0 + 0.3*math.Sin(float64(frame)*0.05)

	// Draw "p" - a circle with stem, with 3D rotation effect
	px := centerX - 12
	py := centerY

	// Apply rotation to x position for 3D effect
	px3d := int(float64(px-centerX)*cos) + centerX

	// The "p" bowl
	radius := int(8 * scale)
	fb.DrawCircle(px3d, py-4, radius, true)
	fb.DrawCircle(px3d, py-4, radius-3, false) // hollow

	// The "p" stem (rotates with character)
	stemX := px3d - int(float64(radius)*cos)
	fb.DrawLine(stemX, py-radius, stemX, py+12, true)
	fb.DrawLine(stemX+1, py-radius, stemX+1, py+12, true)

	// Draw "f" - vertical with horizontal bars
	fx := centerX + 12
	fx3d := int(float64(fx-centerX)*cos) + centerX

	// The "f" stem
	fb.DrawLine(fx3d, py-12, fx3d, py+8, true)
	fb.DrawLine(fx3d+1, py-12, fx3d+1, py+8, true)

	// The "f" top hook
	hookRadius := int(4 * scale)
	for i := 0; i < hookRadius; i++ {
		fb.SetPixel(fx3d+i+1, py-12+int(float64(hookRadius)*sin*0.3), true)
	}

	// The "f" crossbar (rotates slightly)
	barY := py - 2
	barLen := int(8 * scale)
	fb.DrawLine(fx3d-barLen/2, barY, fx3d+barLen/2, barY, true)
	fb.DrawLine(fx3d-barLen/2, barY+1, fx3d+barLen/2, barY+1, true)
}

// ========== SCREENS ==========

// LogoScreen shows animated 3D pfSense logo.
type LogoScreen struct{ frame int }

func (s *LogoScreen) Name() string { return "Logo" }

func (s *LogoScreen) Render(disp *display.Display, m *Metrics) error {
	fb := disp.FrameBuffer()
	fb.Clear()
	f := font.BuiltinFont

	// Draw rotating 3D pf logo on left
	draw3DPF(fb, 28, 32, s.frame)

	// Info on right
	x := 58
	font.RenderText(fb, f, x, 2, "pfSense")
	font.RenderText(fb, f, x, 12, scrollText(m.Hostname, 11, s.frame))

	// Live uptime
	days := int(m.Uptime.Hours() / 24)
	hours := int(m.Uptime.Hours()) % 24
	mins := int(m.Uptime.Minutes()) % 60
	secs := int(m.Uptime.Seconds()) % 60
	font.RenderText(fb, f, x, 24, fmt.Sprintf("%dd%02d:%02d:%02d", days, hours, mins, secs))

	font.RenderText(fb, f, x, 38, fmt.Sprintf("CPU: %.0f%%", m.CPU))
	memPct := float64(m.MemUsed) / float64(m.MemTotal) * 100
	font.RenderText(fb, f, x, 48, fmt.Sprintf("MEM: %.0f%%", memPct))

	return disp.Update()
}

// CPUScreen shows detailed CPU info.
type CPUScreen struct{}

func (s *CPUScreen) Name() string { return "CPU" }

func (s *CPUScreen) Render(d *display.Display, m *Metrics) error {
	fb := d.FrameBuffer()
	fb.Clear()
	f := font.BuiltinFont

	font.RenderTextInverted(fb, f, 0, 0, " CPU ")
	font.RenderText(fb, f, 0, 14, fmt.Sprintf("Usage: %.1f%%", m.CPU))
	drawBar(fb, 0, 26, 125, 10, m.CPU)

	font.RenderText(fb, f, 0, 42, fmt.Sprintf("Load: %.2f %.2f %.2f", m.LoadAvg[0], m.LoadAvg[1], m.LoadAvg[2]))

	days := int(m.Uptime.Hours() / 24)
	hours := int(m.Uptime.Hours()) % 24
	font.RenderText(fb, f, 0, 54, fmt.Sprintf("Uptime: %dd %dh", days, hours))

	return d.Update()
}

// MemoryScreen shows detailed memory info.
type MemoryScreen struct{}

func (s *MemoryScreen) Name() string { return "Memory" }

func (s *MemoryScreen) Render(d *display.Display, m *Metrics) error {
	fb := d.FrameBuffer()
	fb.Clear()
	f := font.BuiltinFont

	memPct := float64(m.MemUsed) / float64(m.MemTotal) * 100
	font.RenderTextInverted(fb, f, 0, 0, " MEMORY ")
	font.RenderText(fb, f, 0, 14, fmt.Sprintf("Usage: %.1f%%", memPct))
	drawBar(fb, 0, 26, 125, 10, memPct)

	usedMB := m.MemUsed / 1024 / 1024
	totalMB := m.MemTotal / 1024 / 1024
	freeMB := totalMB - usedMB
	font.RenderText(fb, f, 0, 42, fmt.Sprintf("Used: %d MB", usedMB))
	font.RenderText(fb, f, 0, 54, fmt.Sprintf("Free: %d MB", freeMB))

	return d.Update()
}

// InterfaceScreen shows active interfaces with IPs.
type InterfaceScreen struct {
	frame     int
	scrollPos int
}

func (s *InterfaceScreen) Name() string { return "Interfaces" }

func (s *InterfaceScreen) Render(d *display.Display, m *Metrics) error {
	fb := d.FrameBuffer()
	fb.Clear()
	f := font.BuiltinFont

	font.RenderTextInverted(fb, f, 0, 0, " INTERFACES ")

	var active []InterfaceMetrics
	for _, iface := range m.Interfaces {
		if iface.IP != "" && iface.Status == "active" {
			active = append(active, iface)
		}
	}

	maxVis := 5
	total := len(active)
	if total > maxVis {
		s.scrollPos = (s.frame / 15) % total
	}

	y := 11
	for i := 0; i < maxVis && i < total; i++ {
		idx := (s.scrollPos + i) % total
		iface := active[idx]
		name := iface.Description
		if name == "" {
			name = iface.Name
		}
		font.RenderText(fb, f, 0, y, scrollText(name, 8, s.frame))
		font.RenderText(fb, f, 55, y, iface.IP)
		y += 10
	}

	if total > maxVis {
		font.RenderText(fb, f, 110, 55, fmt.Sprintf("+%d", total-maxVis))
	}
	if total == 0 {
		font.RenderText(fb, f, 10, 30, "No active ifaces")
	}
	return d.Update()
}

// WANTrafficScreen shows WAN interface traffic.
type WANTrafficScreen struct {
	frame  int
	daemon *StatusDaemon
}

func (s *WANTrafficScreen) Name() string { return "WAN Traffic" }

func (s *WANTrafficScreen) Render(d *display.Display, m *Metrics) error {
	fb := d.FrameBuffer()
	fb.Clear()
	f := font.BuiltinFont

	font.RenderTextInverted(fb, f, 0, 0, " WAN TRAFFIC ")

	y := 12
	count := 0
	for _, iface := range m.Interfaces {
		if iface.Description == "WAN" || strings.HasPrefix(iface.Description, "WAN") {
			if count >= 4 {
				break
			}
			tx, rx := s.daemon.GetIfaceRate(iface.Name)
			name := scrollText(iface.Description, 10, s.frame)
			font.RenderText(fb, f, 0, y, name)
			font.RenderText(fb, f, 0, y+10, fmt.Sprintf("  TX:%s RX:%s", FormatRate(tx), FormatRate(rx)))
			y += 24
			count++
		}
	}
	if count == 0 {
		font.RenderText(fb, f, 10, 30, "No WAN interfaces")
	}
	return d.Update()
}

// TunnelTrafficScreen shows VPN/tunnel traffic.
type TunnelTrafficScreen struct {
	frame     int
	scrollPos int
	daemon    *StatusDaemon
}

func (s *TunnelTrafficScreen) Name() string { return "Tunnel Traffic" }

func (s *TunnelTrafficScreen) Render(d *display.Display, m *Metrics) error {
	fb := d.FrameBuffer()
	fb.Clear()
	f := font.BuiltinFont

	font.RenderTextInverted(fb, f, 0, 0, " TUNNEL TRAFFIC ")

	var tunnels []InterfaceMetrics
	for _, iface := range m.Interfaces {
		if strings.HasPrefix(iface.Name, "tun_wg") ||
			strings.HasPrefix(iface.Description, "GW_") ||
			strings.HasPrefix(iface.Description, "WG_") ||
			strings.HasPrefix(iface.Description, "MULLVAD") {
			tunnels = append(tunnels, iface)
		}
	}

	maxVis := 5
	total := len(tunnels)
	if total > maxVis {
		s.scrollPos = (s.frame / 15) % total
	}

	y := 11
	for i := 0; i < maxVis && i < total; i++ {
		idx := (s.scrollPos + i) % total
		iface := tunnels[idx]
		name := iface.Description
		if name == "" {
			name = iface.Name
		}
		tx, rx := s.daemon.GetIfaceRate(iface.Name)
		font.RenderText(fb, f, 0, y, scrollText(name, 8, s.frame))
		font.RenderText(fb, f, 52, y, fmt.Sprintf("T%s R%s", FormatRate(tx), FormatRate(rx)))
		y += 10
	}

	if total > maxVis {
		font.RenderText(fb, f, 110, 55, fmt.Sprintf("+%d", total-maxVis))
	}
	if total == 0 {
		font.RenderText(fb, f, 15, 30, "No tunnels")
	}
	return d.Update()
}

// LANTrafficScreen shows LAN/other interface traffic.
type LANTrafficScreen struct {
	frame     int
	scrollPos int
	daemon    *StatusDaemon
}

func (s *LANTrafficScreen) Name() string { return "LAN Traffic" }

func (s *LANTrafficScreen) Render(d *display.Display, m *Metrics) error {
	fb := d.FrameBuffer()
	fb.Clear()
	f := font.BuiltinFont

	font.RenderTextInverted(fb, f, 0, 0, " LAN TRAFFIC ")

	var lans []InterfaceMetrics
	for _, iface := range m.Interfaces {
		if iface.Description == "" {
			continue
		}
		// Exclude WAN and tunnels
		if iface.Description == "WAN" || strings.HasPrefix(iface.Description, "WAN") {
			continue
		}
		if strings.HasPrefix(iface.Name, "tun_wg") ||
			strings.HasPrefix(iface.Description, "GW_") ||
			strings.HasPrefix(iface.Description, "WG_") ||
			strings.HasPrefix(iface.Description, "MULLVAD") {
			continue
		}
		lans = append(lans, iface)
	}

	maxVis := 5
	total := len(lans)
	if total > maxVis {
		s.scrollPos = (s.frame / 25) % total
	}

	y := 11
	for i := 0; i < maxVis && i < total; i++ {
		idx := (s.scrollPos + i) % total
		iface := lans[idx]
		tx, rx := s.daemon.GetIfaceRate(iface.Name)
		font.RenderText(fb, f, 0, y, scrollText(iface.Description, 8, s.frame))
		font.RenderText(fb, f, 52, y, fmt.Sprintf("T%s R%s", FormatRate(tx), FormatRate(rx)))
		y += 10
	}

	if total > maxVis {
		font.RenderText(fb, f, 110, 55, fmt.Sprintf("+%d", total-maxVis))
	}
	if total == 0 {
		font.RenderText(fb, f, 10, 30, "No LAN interfaces")
	}
	return d.Update()
}
