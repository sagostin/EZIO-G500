// Example: pfSense status display daemon
//
// This example runs as a daemon, displaying system status with auto-refresh.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sagostin/ezio-g500/pkg/display"
	"github.com/sagostin/ezio-g500/pkg/eziog500"
	"github.com/sagostin/ezio-g500/pkg/pfsense"
)

var (
	port     = flag.String("port", "/dev/ttyS1", "Serial port")
	interval = flag.Duration("interval", 5*time.Second, "Refresh interval")
)

func main() {
	flag.Parse()

	fmt.Printf("Starting pfSense LCD daemon on %s (refresh: %s)\n", *port, *interval)

	disp, err := display.New(*port)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open display: %v\n", err)
		os.Exit(1)
	}
	defer disp.Close()

	// Set LED to green on startup
	disp.SetLED(eziog500.LED1, eziog500.LEDGreen)
	disp.SetBacklight(200)

	metrics := pfsense.NewSystemMetrics()

	// Handle shutdown gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create a multi-screen display
	multiScreen := display.NewMultiScreen(*interval)

	// Screen 1: System status
	multiScreen.AddScreen(func(d *display.Display) error {
		m, err := metrics.GetMetrics()
		if err != nil {
			return err
		}

		status := &display.SystemStatus{
			Hostname: m.Hostname,
			Uptime:   m.Uptime,
			CPU:      m.CPU,
			MemUsed:  m.MemUsed,
			MemTotal: m.MemTotal,
			LoadAvg:  fmt.Sprintf("%.2f %.2f", m.LoadAvg[0], m.LoadAvg[1]),
		}

		for _, iface := range m.Interfaces {
			if iface.IP != "" && iface.Status == "up" {
				status.IPAddress = iface.IP
				break
			}
		}

		template := status.ToTemplate()
		return template.Render(d)
	})

	// Screen 2: Network interfaces
	multiScreen.AddScreen(func(d *display.Display) error {
		m, err := metrics.GetMetrics()
		if err != nil {
			return err
		}

		var infos []display.InterfaceInfo
		for _, iface := range m.Interfaces {
			if iface.Status == "up" && iface.IP != "" {
				infos = append(infos, display.InterfaceInfo{
					Name:   iface.Name,
					Status: iface.Status,
					IP:     iface.IP,
					RxRate: pfsense.FormatBytes(iface.RxBytes),
					TxRate: pfsense.FormatBytes(iface.TxBytes),
				})
			}
		}

		netStatus := &display.NetworkStatus{Interfaces: infos}
		template := netStatus.ToTemplate()
		return template.Render(d)
	})

	// Initial render
	if err := multiScreen.RenderCurrent(disp); err != nil {
		fmt.Fprintf(os.Stderr, "Render error: %v\n", err)
	}

	// Refresh loop
	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	screenTicker := time.NewTicker(*interval * 3) // Rotate screens every 3 intervals
	defer screenTicker.Stop()

	for {
		select {
		case <-sigChan:
			fmt.Println("\nShutting down...")
			disp.Clear()
			disp.PrintLineCentered(3, "SHUTDOWN")
			disp.Update()
			disp.SetLED(eziog500.LED1, eziog500.LEDOff)
			return

		case <-ticker.C:
			if err := multiScreen.RenderCurrent(disp); err != nil {
				fmt.Fprintf(os.Stderr, "Render error: %v\n", err)
				disp.SetLED(eziog500.LED2, eziog500.LEDRed)
			} else {
				disp.SetLED(eziog500.LED2, eziog500.LEDOff)
			}

		case <-screenTicker.C:
			multiScreen.Next()
		}
	}
}
