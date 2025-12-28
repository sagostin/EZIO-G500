// eziolcd is a command-line tool for the EZIO-G500 LCD display.
//
// Usage:
//
//	eziolcd [options] <command> [arguments]
//
// Commands:
//
//	text <message>       Display text on the LCD
//	clear                Clear the display
//	backlight <0-255>    Set backlight level
//	led <1-3> <color>    Set LED color (off, red, green, orange)
//	status               Show system status (pfSense mode)
//	daemon               Run as a daemon with auto-refresh
//	menu                 Interactive menu mode
//	demo                 Run a demo showing various features
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/sagostin/ezio-g500/pkg/display"
	"github.com/sagostin/ezio-g500/pkg/eziog500"
	"github.com/sagostin/ezio-g500/pkg/menu"
	"github.com/sagostin/ezio-g500/pkg/pfsense"
	"github.com/sagostin/ezio-g500/pkg/render3d"
)

var (
	portPath    = flag.String("port", "/dev/ttyS1", "Serial port path")
	refreshRate = flag.Duration("refresh", 5*time.Second, "Refresh rate for daemon mode")
	verbose     = flag.Bool("v", false, "Verbose output")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <command> [arguments]\n\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "Commands:")
		fmt.Fprintln(os.Stderr, "  text <message>       Display text on the LCD")
		fmt.Fprintln(os.Stderr, "  clear                Clear the display")
		fmt.Fprintln(os.Stderr, "  backlight <0-255>    Set backlight level")
		fmt.Fprintln(os.Stderr, "  led <1-3> <color>    Set LED color (off, red, green, orange)")
		fmt.Fprintln(os.Stderr, "  status               Show system status")
		fmt.Fprintln(os.Stderr, "  daemon               Run as a daemon with auto-refresh")
		fmt.Fprintln(os.Stderr, "  menu                 Interactive menu mode (use arrow keys)")
		fmt.Fprintln(os.Stderr, "  demo                 Run a demo showing various features")
		fmt.Fprintln(os.Stderr, "\nOptions:")
		flag.PrintDefaults()
	}

	flag.Parse()

	// Enable verbose mode for debugging
	if *verbose {
		eziog500.SetVerbose(true)
	}

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	cmd := flag.Arg(0)

	switch cmd {
	case "text":
		if flag.NArg() < 2 {
			fmt.Fprintln(os.Stderr, "Usage: eziolcd text <message>")
			os.Exit(1)
		}
		message := strings.Join(flag.Args()[1:], " ")
		if err := cmdText(message); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "clear":
		if err := cmdClear(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "backlight":
		if flag.NArg() < 2 {
			fmt.Fprintln(os.Stderr, "Usage: eziolcd backlight <0-255>")
			os.Exit(1)
		}
		level, err := strconv.Atoi(flag.Arg(1))
		if err != nil || level < 0 || level > 255 {
			fmt.Fprintln(os.Stderr, "Backlight level must be 0-255")
			os.Exit(1)
		}
		if err := cmdBacklight(byte(level)); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "led":
		if flag.NArg() < 3 {
			fmt.Fprintln(os.Stderr, "Usage: eziolcd led <1-3> <off|red|green|orange>")
			os.Exit(1)
		}
		ledNum, err := strconv.Atoi(flag.Arg(1))
		if err != nil || ledNum < 1 || ledNum > 3 {
			fmt.Fprintln(os.Stderr, "LED number must be 1-3")
			os.Exit(1)
		}
		color := strings.ToLower(flag.Arg(2))
		if err := cmdLED(ledNum, color); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "status":
		if err := cmdStatus(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "daemon":
		if err := cmdDaemon(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "buttons":
		if err := cmdButtonTest(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "menu":
		if err := cmdMenu(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "demo":
		if err := cmdDemo(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		flag.Usage()
		os.Exit(1)
	}
}

func cmdText(message string) error {
	// Native text mode - just send raw ASCII directly
	device, err := eziog500.Open(*portPath)
	if err != nil {
		return err
	}
	defer device.Close()

	// Clear screen first: Init + Home + Clear
	if err := device.Init(); err != nil {
		return err
	}
	if err := device.Home(); err != nil {
		return err
	}
	if err := device.Clear(); err != nil {
		return err
	}

	// Send the raw text - the display has built-in font rendering
	return device.Write([]byte(message))
}

func cmdClear() error {
	device, err := eziog500.Open(*portPath)
	if err != nil {
		return err
	}
	defer device.Close()

	// Send Init, Home, and Clear (matches test-serial.sh)
	if err := device.Init(); err != nil {
		return err
	}
	if err := device.Home(); err != nil {
		return err
	}
	return device.Clear()
}

func cmdBacklight(level byte) error {
	device, err := eziog500.Open(*portPath)
	if err != nil {
		return err
	}
	defer device.Close()

	if err := device.Init(); err != nil {
		return err
	}

	return device.SetBacklight(level)
}

func cmdLED(ledNum int, color string) error {
	device, err := eziog500.Open(*portPath)
	if err != nil {
		return err
	}
	defer device.Close()

	if err := device.Init(); err != nil {
		return err
	}

	var led eziog500.LED
	switch ledNum {
	case 1:
		led = eziog500.LED1
	case 2:
		led = eziog500.LED2
	case 3:
		led = eziog500.LED3
	}

	var ledColor eziog500.LEDColor
	switch color {
	case "off":
		ledColor = eziog500.LEDOff
	case "red":
		ledColor = eziog500.LEDRed
	case "green":
		ledColor = eziog500.LEDGreen
	case "orange":
		ledColor = eziog500.LEDOrange
	default:
		return fmt.Errorf("unknown color: %s (use off, red, green, orange)", color)
	}

	return device.SetLED(led, ledColor)
}

func cmdStatus() error {
	disp, err := display.New(*portPath)
	if err != nil {
		return err
	}
	defer disp.Close()

	metrics := pfsense.NewSystemMetrics()
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
		LoadAvg:  fmt.Sprintf("%.2f %.2f %.2f", m.LoadAvg[0], m.LoadAvg[1], m.LoadAvg[2]),
	}

	// Get first interface IP
	if len(m.Interfaces) > 0 {
		for _, iface := range m.Interfaces {
			if iface.IP != "" && iface.Status == "up" {
				status.IPAddress = iface.IP
				break
			}
		}
	}

	template := status.ToTemplate()
	return template.Render(disp)
}

func cmdDaemon() error {
	disp, err := display.New(*portPath)
	if err != nil {
		return err
	}
	defer disp.Close()

	if *verbose {
		fmt.Printf("Starting status daemon on %s\n", *portPath)
		fmt.Printf("Update interval: %s, Screen rotation: 10s\n", *refreshRate)
	}

	// Create status daemon with rotating screens
	// Updates every refreshRate, rotates screens every 10 seconds
	daemon := pfsense.NewStatusDaemon(disp, *refreshRate, 10*time.Second)

	// Run the daemon (blocks forever)
	return daemon.Run()
}

func updateStatus(disp *display.Display, metrics *pfsense.SystemMetrics) error {
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
		LoadAvg:  fmt.Sprintf("%.2f %.2f %.2f", m.LoadAvg[0], m.LoadAvg[1], m.LoadAvg[2]),
	}

	if len(m.Interfaces) > 0 {
		for _, iface := range m.Interfaces {
			if iface.IP != "" && iface.Status == "up" {
				status.IPAddress = iface.IP
				break
			}
		}
	}

	template := status.ToTemplate()
	return template.Render(disp)
}

func cmdDemo() error {
	disp, err := display.New(*portPath)
	if err != nil {
		return err
	}
	defer disp.Close()

	reader := bufio.NewReader(os.Stdin)

	// Demo 1: Graphics text display
	fmt.Println("\n=== Demo 1: Graphics Text ===")
	fmt.Println("This uses our custom bitmap font in graphics mode")
	disp.Clear()
	disp.DrawRect(0, 0, 128, 64)
	disp.Print(5, 5, "EZIO-G500")
	disp.Print(5, 25, "Go Library")
	disp.Print(5, 45, "Demo Mode")
	disp.Update()
	fmt.Println("Press Enter for next demo...")
	reader.ReadString('\n')

	// Demo 2: Drawing primitives
	fmt.Println("\n=== Demo 2: Drawing Primitives ===")
	fmt.Println("Rectangle, diagonal lines, and text")
	disp.Clear()
	disp.DrawRect(0, 0, 128, 64)
	disp.DrawLine(0, 0, 127, 63)
	disp.DrawLine(127, 0, 0, 63)
	disp.Print(40, 28, "GRAPHICS")
	disp.Update()
	fmt.Println("Press Enter for next demo...")
	reader.ReadString('\n')

	// Demo 3: Progress bar
	fmt.Println("\n=== Demo 3: Progress Bar ===")
	fmt.Println("Animated loading bar")
	for pct := 0.0; pct <= 100.0; pct += 5.0 {
		disp.Clear()
		disp.DrawRect(0, 0, 128, 64)
		disp.Print(35, 8, "LOADING...")
		bar := &display.ProgressBar{X: 10, Y: 25, Width: 108, Height: 12}
		bar.Render(disp, pct)
		disp.Print(52, 45, fmt.Sprintf("%.0f%%", pct))
		disp.Update()
		time.Sleep(100 * time.Millisecond)
	}
	fmt.Println("Press Enter for next demo...")
	reader.ReadString('\n')

	// Demo 4: 3D Rotating Cube
	fmt.Println("\n=== Demo 4: 3D Rotating Cube ===")
	fmt.Println("Wireframe cube rotating in 3D space")
	cube := render3d.NewCube(1.5)
	cam := render3d.DefaultCamera()
	for frame := 0; frame < 60; frame++ {
		disp.Clear()
		// Create a fresh cube and rotate it
		frameCube := cube.Copy()
		angle := float64(frame) * 0.1
		frameCube.Rotate(angle*0.7, angle, angle*0.3)
		frameCube.Draw(disp.FrameBuffer(), cam, true)
		disp.Update()
		time.Sleep(50 * time.Millisecond)
	}
	fmt.Println("Press Enter for next demo...")
	reader.ReadString('\n')

	// Demo 4: LED cycling
	fmt.Println("\n=== Demo 5: LED Cycling ===")
	fmt.Println("Cycling through LED colors")
	device := disp.Device()
	for _, led := range []eziog500.LED{eziog500.LED1, eziog500.LED2, eziog500.LED3} {
		fmt.Printf("LED %d: Red...", led)
		device.SetLED(led, eziog500.LEDRed)
		time.Sleep(500 * time.Millisecond)
		fmt.Print(" Green...")
		device.SetLED(led, eziog500.LEDGreen)
		time.Sleep(500 * time.Millisecond)
		fmt.Print(" Orange...")
		device.SetLED(led, eziog500.LEDOrange)
		time.Sleep(500 * time.Millisecond)
		fmt.Println(" Off")
		device.SetLED(led, eziog500.LEDOff)
		time.Sleep(200 * time.Millisecond)
	}

	// Final
	fmt.Println("\n=== Demo Complete ===")
	disp.Clear()
	disp.DrawRect(0, 0, 128, 64)
	disp.Print(15, 28, "DEMO COMPLETE")
	disp.Update()

	return nil
}

func cmdMenu() error {
	disp, err := display.New(*portPath)
	if err != nil {
		return err
	}
	defer disp.Close()

	// Set up signal handling for graceful exit
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		disp.Clear()
		disp.DrawRect(0, 0, 128, 64) // Add border to make it visible
		disp.Print(40, 28, "GOODBYE")
		disp.Update()
		disp.Close()
		os.Exit(0)
	}()

	// Create button reader
	buttonReader := eziog500.NewButtonReader(disp.Device(), 100*time.Millisecond)

	// Build pfSense menu
	menuBuilder := menu.NewPfSenseMenuBuilder(disp)
	rootMenu := menuBuilder.Build()

	// Create menu controller
	controller := menu.NewMenuController(disp, buttonReader, rootMenu)

	if *verbose {
		fmt.Printf("Starting interactive menu on %s\n", *portPath)
		fmt.Println("Use arrow keys to navigate, Enter to select, Escape to go back")
	}

	// Run the menu
	return controller.Run()
}

// cmdButtonTest tests button input from the device
func cmdButtonTest() error {
	fmt.Println("Starting button input test...")
	fmt.Println("Press buttons on the device. Press Ctrl+C to exit.")

	device, err := eziog500.Open(*portPath)
	if err != nil {
		return err
	}
	defer device.Close()

	// Start a persistent session for bidirectional I/O
	session, err := device.StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.Close()

	// Create session button reader
	buttonReader := eziog500.NewSessionButtonReader(session)
	buttons, stop := buttonReader.ButtonChannel()
	defer stop()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("Waiting for button presses...")

	for {
		select {
		case btn := <-buttons:
			fmt.Printf("Button pressed: %s (0x%02X)\n", btn.String(), byte(btn))
		case <-sigChan:
			fmt.Println("\nExiting...")
			return nil
		}
	}
}
