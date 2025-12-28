package eziog500

import (
	"time"
)

// Button represents a button press from the EZIO-G500.
type Button byte

const (
	ButtonNone  Button = 0x00
	ButtonHelp  Button = 0x41 // HELP button
	ButtonLeft  Button = 0x42 // LEFT
	ButtonEsc   Button = 0x43 // ESC
	ButtonUp    Button = 0x44 // UP
	ButtonEnter Button = 0x45 // ENTER (sometimes 0x01)
	ButtonDown  Button = 0x46 // DOWN
	ButtonRight Button = 0x47 // RIGHT
)

// String returns the button name.
func (b Button) String() string {
	switch b {
	case ButtonUp:
		return "Up"
	case ButtonDown:
		return "Down"
	case ButtonLeft:
		return "Left"
	case ButtonRight:
		return "Right"
	case ButtonEnter:
		return "Enter"
	case ButtonEsc:
		return "Escape"
	case ButtonHelp:
		return "Help"
	default:
		return "None"
	}
}

// ButtonReader provides non-blocking button input reading.
type ButtonReader struct {
	device  *Device
	timeout time.Duration
}

// NewButtonReader creates a button reader for the device.
func NewButtonReader(device *Device, timeout time.Duration) *ButtonReader {
	return &ButtonReader{
		device:  device,
		timeout: timeout,
	}
}

// ReadButton reads a single button press with timeout.
// Returns ButtonNone if no button was pressed within the timeout.
func (br *ButtonReader) ReadButton() Button {
	buf := make([]byte, 1)

	// Set read timeout on the port if supported
	// For now, we do a simple read attempt
	n, err := br.device.Read(buf)
	if err != nil || n == 0 {
		return ButtonNone
	}

	return Button(buf[0])
}

// ReadButtonBlocking reads a button press, blocking until a button is pressed.
func (br *ButtonReader) ReadButtonBlocking() Button {
	buf := make([]byte, 1)

	for {
		n, err := br.device.Read(buf)
		if err != nil {
			time.Sleep(10 * time.Millisecond)
			continue
		}
		if n > 0 {
			return Button(buf[0])
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// ButtonChannel returns a channel that emits button presses.
// The channel should be closed by calling Stop() on the returned stopper.
func (br *ButtonReader) ButtonChannel() (<-chan Button, func()) {
	ch := make(chan Button, 10)
	stop := make(chan struct{})

	go func() {
		defer close(ch)
		buf := make([]byte, 1)

		for {
			select {
			case <-stop:
				return
			default:
				n, err := br.device.Read(buf)
				if err == nil && n > 0 {
					btn := Button(buf[0])
					if btn != ButtonNone {
						select {
						case ch <- btn:
						default:
							// Channel full, drop the button
						}
					}
				}
				time.Sleep(10 * time.Millisecond)
			}
		}
	}()

	stopper := func() {
		close(stop)
	}

	return ch, stopper
}

// SessionButtonReader reads button input from a PersistentSession.
type SessionButtonReader struct {
	session *PersistentSession
}

// NewSessionButtonReader creates a button reader using a persistent session.
func NewSessionButtonReader(session *PersistentSession) *SessionButtonReader {
	return &SessionButtonReader{session: session}
}

// ReadButton reads a single button press.
// Returns immediately with ButtonNone if no data available.
func (br *SessionButtonReader) ReadButton() Button {
	buf := make([]byte, 16) // Read multiple in case of buffered input
	n, err := br.session.Read(buf)
	if err != nil || n == 0 {
		return ButtonNone
	}

	// Return the last valid button in the buffer
	for i := n - 1; i >= 0; i-- {
		b := Button(buf[i])
		if b != ButtonNone {
			debugf("Button read: 0x%02X (%s)", buf[i], b.String())
			return b
		}
	}
	return ButtonNone
}

// ButtonChannel returns a channel that emits button presses.
func (br *SessionButtonReader) ButtonChannel() (<-chan Button, func()) {
	ch := make(chan Button, 10)
	stop := make(chan struct{})

	go func() {
		defer close(ch)
		buf := make([]byte, 16) // Read multiple bytes at once

		for {
			select {
			case <-stop:
				return
			default:
				n, err := br.session.Read(buf)
				if err == nil && n > 0 {
					// Log all raw bytes received
					debugf("Raw bytes received (%d): % 02X", n, buf[:n])

					// Process each byte
					for i := 0; i < n; i++ {
						btn := Button(buf[i])
						if btn != ButtonNone {
							debugf("Button: 0x%02X (%s)", buf[i], btn.String())
							select {
							case ch <- btn:
							default:
								// Channel full, drop
							}
						}
					}
				}
				time.Sleep(10 * time.Millisecond)
			}
		}
	}()

	stopper := func() {
		close(stop)
	}

	return ch, stopper
}
