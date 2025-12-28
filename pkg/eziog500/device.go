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

	d := &Device{
		portPath:     portPath,
		commandDelay: DefaultCommandDelay,
	}

	return d, nil
}

// OpenWithoutStty is the same as Open (kept for API compatibility)
func OpenWithoutStty(portPath string) (*Device, error) {
	return Open(portPath)
}

// Close closes the connection to the display.
// This flushes any buffered data using cu.
func (d *Device) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	debugf("Closing port (flushing buffer)")

	// Flush any remaining data
	if d.buffer.Len() > 0 {
		if err := d.flushWithCu(); err != nil {
			return err
		}
	}

	return nil
}

// flushWithCu sends buffered data using cu
func (d *Device) flushWithCu() error {
	if d.buffer.Len() == 0 {
		return nil
	}

	data := d.buffer.Bytes()
	debugf("Flushing %d bytes via cu", len(data))

	// Debug: show what we're writing
	if Verbose {
		if len(data) <= 20 {
			debugf("Data: % 02X", data)
		} else {
			debugf("Data: % 02X... (truncated)", data[:20])
		}
	}

	// Use cu to send data - this is what actually works!
	cmd := exec.Command("cu", "-l", d.portPath, "-s", fmt.Sprintf("%d", DefaultBaudRate))

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	// Start cu
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start cu: %w", err)
	}

	// Write data to cu's stdin
	_, err = stdin.Write(data)
	if err != nil {
		cmd.Process.Kill()
		return fmt.Errorf("failed to write to cu: %w", err)
	}

	// Small delay to let cu process the data
	time.Sleep(100 * time.Millisecond)

	// Close stdin to signal EOF
	stdin.Close()

	// Send ~. to exit cu cleanly (via separate write if needed)
	// Actually, closing stdin should cause cu to exit

	// Wait for cu to finish with a timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			debugf("cu exited with: %v (this is often OK)", err)
		}
	case <-time.After(2 * time.Second):
		debugf("cu timeout, killing process")
		cmd.Process.Kill()
	}

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

// Flush sends all buffered data immediately using cu
func (d *Device) Flush() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.flushWithCu()
}

// Read reads bytes from the display (for button input).
// Note: This is not yet implemented with cu-based approach
func (d *Device) Read(buf []byte) (int, error) {
	return 0, io.EOF
}

// PortPath returns the serial port path.
func (d *Device) PortPath() string {
	return d.portPath
}

// PersistentSession represents a long-running cu session for bidirectional I/O.
type PersistentSession struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	mu     sync.Mutex
}

// StartSession starts a persistent cu session for bidirectional communication.
// This is needed for button monitoring while also sending display commands.
func (d *Device) StartSession() (*PersistentSession, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	debugf("Starting persistent session on %s", d.portPath)

	cmd := exec.Command("cu", "-l", d.portPath, "-s", fmt.Sprintf("%d", DefaultBaudRate))

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		return nil, fmt.Errorf("failed to start cu: %w", err)
	}

	session := &PersistentSession{
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
	}

	// cu outputs "Connected" on startup - drain this before returning
	// Read until we get the newline after "Connected\n"
	drainBuf := make([]byte, 64)
	for {
		n, err := stdout.Read(drainBuf)
		if err != nil || n == 0 {
			break
		}
		debugf("Draining cu startup: %q", string(drainBuf[:n]))
		// Check if we've received the newline (end of Connected message)
		if bytes.Contains(drainBuf[:n], []byte{'\n'}) {
			break
		}
	}

	// Small additional delay to let things settle
	time.Sleep(50 * time.Millisecond)

	debugf("Persistent session started")
	return session, nil
}

// Write sends data to the display via the persistent session.
func (ps *PersistentSession) Write(data []byte) (int, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return ps.stdin.Write(data)
}

// Read reads data from the display (button presses).
func (ps *PersistentSession) Read(buf []byte) (int, error) {
	return ps.stdout.Read(buf)
}

// Close ends the persistent session.
func (ps *PersistentSession) Close() error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	debugf("Closing persistent session")

	// Close stdin to signal EOF
	ps.stdin.Close()

	// Give cu a moment to exit
	done := make(chan error, 1)
	go func() {
		done <- ps.cmd.Wait()
	}()

	select {
	case <-done:
		// Ok
	case <-time.After(2 * time.Second):
		ps.cmd.Process.Kill()
	}

	return nil
}
