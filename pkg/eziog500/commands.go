package eziog500

// Command bytes and sequences for the EZIO-G500 display.
const (
	// Escape byte - prefix for most commands
	ESC = 0x1B

	// Single-byte commands
	cmdClear = 0x0C // Clear screen
	cmdHome  = 0x0B // Home cursor

	// ESC + second byte commands
	cmdInit      = 0x40 // '@' - Display initialization
	cmdBacklight = 0x42 // 'B' - Backlight control
	cmdUpload    = 0x47 // 'G' - Upload graphics
	cmdLED       = 0x4C // 'L' - LED control
	cmdShowPage  = 0x50 // 'P' - Show graphics page
	cmdSavePage  = 0x53 // 'S' - Save screen to page
	cmdInverted  = 0x72 // 'r' - Inverted character mode

	// Cursor movement (ESC [ X)
	cmdCursorPrefix = 0x5B // '['
	cmdCursorUp     = 0x41 // 'A'
	cmdCursorDown   = 0x42 // 'B'
	cmdCursorRight  = 0x43 // 'C'
	cmdCursorLeft   = 0x44 // 'D'
	cmdCursorHome   = 0x48 // 'H'
)

// Direction represents cursor movement direction.
type Direction int

const (
	Up Direction = iota
	Down
	Right
	Left
)

// Init initializes the display (ESC @).
// This should be called before other operations.
func (d *Device) Init() error {
	return d.Write([]byte{ESC, cmdInit})
}

// Clear clears the display screen (0x0C).
func (d *Device) Clear() error {
	return d.Write([]byte{cmdClear})
}

// Home moves the cursor to the home position (0x0B).
func (d *Device) Home() error {
	return d.Write([]byte{cmdHome})
}

// SetBacklight sets the backlight level (ESC B n).
// Level is 0-255, where 0 is off and 255 is maximum brightness.
func (d *Device) SetBacklight(level byte) error {
	return d.Write([]byte{ESC, cmdBacklight, level})
}

// UploadImage uploads a 1024-byte graphics image to the display (ESC G + data).
// The data must be exactly 1024 bytes in the correct format.
// Use FrameBuffer.ToDeviceFormat() to get properly formatted data.
// Note: This flushes immediately so animations work frame-by-frame.
func (d *Device) UploadImage(data [1024]byte) error {
	// Send command prefix
	if err := d.Write([]byte{ESC, cmdUpload}); err != nil {
		return err
	}
	// Send image data
	if err := d.Write(data[:]); err != nil {
		return err
	}
	// Flush immediately so this frame is displayed
	return d.Flush()
}

// ShowPage displays a previously uploaded graphics page (ESC P n).
func (d *Device) ShowPage(page byte) error {
	return d.Write([]byte{ESC, cmdShowPage, page})
}

// SavePage saves the current screen to a page (ESC S n).
func (d *Device) SavePage(page byte) error {
	return d.Write([]byte{ESC, cmdSavePage, page})
}

// SetInverted enables or disables inverted character mode (ESC r n).
func (d *Device) SetInverted(on bool) error {
	var val byte = 0
	if on {
		val = 1
	}
	return d.Write([]byte{ESC, cmdInverted, val})
}

// MoveCursor moves the cursor in the specified direction (ESC [ A/B/C/D).
func (d *Device) MoveCursor(direction Direction) error {
	var dirByte byte
	switch direction {
	case Up:
		dirByte = cmdCursorUp
	case Down:
		dirByte = cmdCursorDown
	case Right:
		dirByte = cmdCursorRight
	case Left:
		dirByte = cmdCursorLeft
	default:
		dirByte = cmdCursorUp
	}
	return d.Write([]byte{ESC, cmdCursorPrefix, dirByte})
}

// CursorHome moves the cursor to the home position using cursor command (ESC [ H).
func (d *Device) CursorHome() error {
	return d.Write([]byte{ESC, cmdCursorPrefix, cmdCursorHome})
}

// WriteText sends raw ASCII text to the display's native text mode.
// This is the simplest way to display text - just send characters directly.
// The display has a built-in character set and will render the text.
// Use Init(), Home(), and Clear() to prepare the display first.
func (d *Device) WriteText(text string) error {
	return d.Write([]byte(text))
}

// WriteTextLine writes text and moves to the next line.
// Note: The newline character behavior depends on the display's configuration.
func (d *Device) WriteTextLine(text string) error {
	if err := d.Write([]byte(text)); err != nil {
		return err
	}
	return d.Write([]byte{'\n'})
}
