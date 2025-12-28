// Package eziog500 provides low-level communication with the EZIO-G500 LCD display.
//
// The EZIO-G500 is a 128x64 pixel graphics LCD found in Checkpoint appliances.
// It communicates via serial at 115200 baud, 8N1, using an ESC-based command protocol.
package eziog500

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

const (
	// DefaultBaudRate is the baud rate for EZIO-G500 communication
	DefaultBaudRate = 115200

	// DefaultCommandDelay is the delay after each command (from Checkpoint driver analysis)
	// According to docs: "each command bloc was followed by an usleep(1000);"
	DefaultCommandDelay = 1 * time.Millisecond
)

// Verbose enables debug output when set to true
var Verbose = false

// Device represents a connection to an EZIO-G500 LCD display.
type Device struct {
	portPath     string
	port         *os.File // Direct file handle to serial port
	mu           sync.Mutex
	commandDelay time.Duration
	buffer       bytes.Buffer // Buffer to collect data to send
}

// SetVerbose enables or disables verbose debug output globally
func SetVerbose(v bool) {
	Verbose = v
}

// debugf prints debug output if verbose mode is enabled
func debugf(format string, args ...interface{}) {
	if Verbose {
		fmt.Fprintf(os.Stderr, "[EZIO] "+format+"\n", args...)
	}
}

// Open opens a connection to the EZIO-G500 display on the specified serial port.
// The port is typically /dev/cuau1 on FreeBSD or /dev/ttyS1 on Linux.
func Open(portPath string) (*Device, error) {
	debugf("Opening port: %s at %d baud", portPath, DefaultBaudRate)

	// Configure serial port using stty (one-time setup)
	cmd := exec.Command("stty", "-f", portPath, fmt.Sprintf("%d", DefaultBaudRate), "cs8", "-cstopb", "-parenb", "raw", "-echo")
	if err := cmd.Run(); err != nil {
		debugf("stty failed: %v (continuing anyway)", err)
	}

	// Open the serial port directly
	port, err := os.OpenFile(portPath, os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open serial port: %w", err)
	}

	d := &Device{
		portPath:     portPath,
		port:         port,
		commandDelay: DefaultCommandDelay,
	}

	return d, nil
}

// OpenWithoutStty opens without stty configuration (for testing)
func OpenWithoutStty(portPath string) (*Device, error) {
	port, err := os.OpenFile(portPath, os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open serial port: %w", err)
	}

	return &Device{
		portPath:     portPath,
		port:         port,
		commandDelay: DefaultCommandDelay,
	}, nil
}

// Close closes the connection to the display.
func (d *Device) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	debugf("Closing port")

	// Flush any remaining data
	if d.buffer.Len() > 0 {
		if err := d.flushDirect(); err != nil {
			debugf("Error flushing on close: %v", err)
		}
	}

	if d.port != nil {
		err := d.port.Close()
		d.port = nil
		return err
	}
	return nil
}

// flushDirect sends buffered data directly to the serial port (no cu!)
func (d *Device) flushDirect() error {
	if d.buffer.Len() == 0 {
		return nil
	}

	if d.port == nil {
		return fmt.Errorf("serial port not open")
	}

	data := d.buffer.Bytes()
	debugf("Flushing %d bytes directly", len(data))

	// Debug: show what we're writing
	if Verbose {
		if len(data) <= 20 {
			debugf("Data: % 02X", data)
		} else {
			debugf("Data: % 02X... (truncated)", data[:20])
		}
	}

	// Write directly to the serial port
	_, err := d.port.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write to serial port: %w", err)
	}

	// Sync to ensure data is sent
	d.port.Sync()

	// Clear the buffer
	d.buffer.Reset()

	debugf("Flush complete")
	return nil
}

// SetCommandDelay sets the delay between commands.
// The default is 1ms, based on the Checkpoint driver analysis.
func (d *Device) SetCommandDelay(delay time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.commandDelay = delay
	debugf("Command delay set to: %v", delay)
}

// Write buffers bytes to send to the display.
// Data is actually sent when Close() is called or Flush() is explicitly called.
func (d *Device) Write(data []byte) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Debug: show what we're buffering
	if Verbose {
		if len(data) <= 20 {
			debugf("Buffer %d bytes: % 02X", len(data), data)
		} else {
			debugf("Buffer %d bytes: % 02X... (truncated)", len(data), data[:20])
		}
	}

	d.buffer.Write(data)

	// Apply command delay
	if d.commandDelay > 0 {
		time.Sleep(d.commandDelay)
	}

	return nil
}

// Flush sends all buffered data immediately
func (d *Device) Flush() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.flushDirect()
}

// Read reads bytes from the display (for button input).
func (d *Device) Read(buf []byte) (int, error) {
	if d.port == nil {
		return 0, io.EOF
	}
	return d.port.Read(buf)
}

// PortPath returns the serial port path.
func (d *Device) PortPath() string {
	return d.portPath
}

// PersistentSession represents a long-running session for bidirectional I/O.
// Now uses direct file I/O instead of spawning cu processes.
type PersistentSession struct {
	port *os.File
	mu   sync.Mutex
}

// StartSession starts a persistent session for bidirectional communication.
func (d *Device) StartSession() (*PersistentSession, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	debugf("Starting persistent session on %s", d.portPath)

	// Use the existing port if available, otherwise open a new one
	if d.port != nil {
		return &PersistentSession{port: d.port}, nil
	}

	// Configure serial port
	cmd := exec.Command("stty", "-f", d.portPath, fmt.Sprintf("%d", DefaultBaudRate), "cs8", "-cstopb", "-parenb", "raw", "-echo")
	if err := cmd.Run(); err != nil {
		debugf("stty failed: %v", err)
	}

	port, err := os.OpenFile(d.portPath, os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open serial port: %w", err)
	}

	debugf("Persistent session started")
	return &PersistentSession{port: port}, nil
}

// Write sends data to the display via the persistent session.
func (ps *PersistentSession) Write(data []byte) (int, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return ps.port.Write(data)
}

// Read reads data from the display (button presses).
func (ps *PersistentSession) Read(buf []byte) (int, error) {
	return ps.port.Read(buf)
}

// Close ends the persistent session.
func (ps *PersistentSession) Close() error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	debugf("Closing persistent session")
	// Don't close the port here - it may be shared with the Device
	// The Device.Close() will handle port cleanup
	return nil
}
