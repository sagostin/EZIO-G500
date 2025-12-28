// Package display provides a high-level interface for the EZIO-G500 LCD.
package display

import (
	"github.com/sagostin/ezio-g500/pkg/eziog500"
	"github.com/sagostin/ezio-g500/pkg/font"
)

// Display provides a high-level interface for text and graphics on the LCD.
type Display struct {
	device *eziog500.Device
	fb     *eziog500.FrameBuffer
	font   font.Font
}

// New creates a new Display connected to the specified serial port.
func New(portPath string) (*Display, error) {
	device, err := eziog500.Open(portPath)
	if err != nil {
		return nil, err
	}

	d := &Display{
		device: device,
		fb:     eziog500.NewFrameBuffer(),
		font:   font.BuiltinFont,
	}

	// Don't call Init - it interferes with graphics mode
	// ESC G works without Init
	return d, nil
}

// NewWithDevice creates a Display using an existing device connection.
func NewWithDevice(device *eziog500.Device) *Display {
	return &Display{
		device: device,
		fb:     eziog500.NewFrameBuffer(),
		font:   font.BuiltinFont,
	}
}

// Close closes the display connection.
func (d *Display) Close() error {
	return d.device.Close()
}

// Device returns the underlying device for advanced operations.
func (d *Display) Device() *eziog500.Device {
	return d.device
}

// FrameBuffer returns the underlying framebuffer for advanced drawing.
func (d *Display) FrameBuffer() *eziog500.FrameBuffer {
	return d.fb
}

// SetFont sets the font used for text rendering.
func (d *Display) SetFont(f font.Font) {
	d.font = f
}

// Clear clears the framebuffer and optionally updates the display.
func (d *Display) Clear() error {
	d.fb.Clear()
	return nil
}

// Update sends the current framebuffer contents to the display.
func (d *Display) Update() error {
	data := d.fb.ToDeviceFormat()
	return d.device.UploadImage(data)
}

// ClearAndUpdate clears and immediately updates the display.
func (d *Display) ClearAndUpdate() error {
	d.fb.Clear()
	return d.Update()
}

// Print renders text at the specified pixel position.
func (d *Display) Print(x, y int, text string) {
	font.RenderText(d.fb, d.font, x, y, text)
}

// PrintInverted renders inverted text (white background, black text).
func (d *Display) PrintInverted(x, y int, text string) {
	font.RenderTextInverted(d.fb, d.font, x, y, text)
}

// PrintLine renders text on a specific line number (0-7 for 8px font).
func (d *Display) PrintLine(line int, text string) {
	y := line * d.font.Height()
	font.RenderText(d.fb, d.font, 0, y, text)
}

// PrintLineCentered renders centered text on a specific line.
func (d *Display) PrintLineCentered(line int, text string) {
	y := line * d.font.Height()
	width := font.MeasureText(d.font, text)
	x := (eziog500.Width - width) / 2
	if x < 0 {
		x = 0
	}
	font.RenderText(d.fb, d.font, x, y, text)
}

// PrintLineRight renders right-aligned text on a specific line.
func (d *Display) PrintLineRight(line int, text string) {
	y := line * d.font.Height()
	width := font.MeasureText(d.font, text)
	x := eziog500.Width - width
	if x < 0 {
		x = 0
	}
	font.RenderText(d.fb, d.font, x, y, text)
}

// MaxLines returns the maximum number of text lines for the current font.
func (d *Display) MaxLines() int {
	return eziog500.Height / d.font.Height()
}

// SetBacklight sets the display backlight level (0-255).
func (d *Display) SetBacklight(level byte) error {
	return d.device.SetBacklight(level)
}

// SetLED sets the color of an LED.
func (d *Display) SetLED(led eziog500.LED, color eziog500.LEDColor) error {
	return d.device.SetLED(led, color)
}

// DrawLine draws a line on the framebuffer.
func (d *Display) DrawLine(x1, y1, x2, y2 int) {
	d.fb.DrawLine(x1, y1, x2, y2, true)
}

// DrawRect draws a rectangle outline on the framebuffer.
func (d *Display) DrawRect(x, y, w, h int) {
	d.fb.DrawRect(x, y, w, h, true)
}

// FillRect fills a rectangle on the framebuffer.
func (d *Display) FillRect(x, y, w, h int) {
	d.fb.FillRect(x, y, w, h, true)
}

// SetPixel sets a pixel on the framebuffer.
func (d *Display) SetPixel(x, y int, on bool) {
	d.fb.SetPixel(x, y, on)
}
