// Package ui provides reusable UI widgets for the EZIO-G500 display.
package ui

import (
	"github.com/sagostin/ezio-g500/pkg/eziog500"
	"github.com/sagostin/ezio-g500/pkg/font"
)

// Widget is the base interface for UI components.
type Widget interface {
	Render(fb *eziog500.FrameBuffer, x, y int)
	Width() int
	Height() int
}

// Button represents a clickable button widget.
type Button struct {
	Label    string
	Selected bool
	Disabled bool
	width    int
	height   int
}

// NewButton creates a new button with the given label.
func NewButton(label string) *Button {
	f := font.BuiltinFont
	width := font.MeasureText(f, label) + 8 // padding
	return &Button{
		Label:  label,
		width:  width,
		height: f.Height() + 4,
	}
}

// Render draws the button on the framebuffer.
func (b *Button) Render(fb *eziog500.FrameBuffer, x, y int) {
	f := font.BuiltinFont

	// Draw button border/background
	if b.Selected {
		// Filled button (selected)
		fb.FillRect(x, y, b.width, b.height, true)
		// Draw text inverted
		textX := x + 4
		textY := y + 2
		for _, r := range b.Label {
			glyph := f.GetGlyph(r)
			if glyph == nil {
				continue
			}
			for col, by := range glyph {
				for bit := 0; bit < 8; bit++ {
					if (by & (1 << bit)) != 0 {
						fb.SetPixel(textX+col, textY+bit, false)
					}
				}
			}
			textX += len(glyph)
		}
	} else {
		// Outline button
		fb.DrawRect(x, y, b.width, b.height, true)
		font.RenderText(fb, f, x+4, y+2, b.Label)
	}

	// Disabled style (strikethrough)
	if b.Disabled {
		midY := y + b.height/2
		fb.DrawHLine(x, x+b.width-1, midY, true)
	}
}

func (b *Button) Width() int  { return b.width }
func (b *Button) Height() int { return b.height }

// ProgressIndicator shows a horizontal progress bar.
type ProgressIndicator struct {
	Value   float64 // 0.0 to 100.0
	Width   int
	Height  int
	Rounded bool
}

// NewProgressIndicator creates a new progress indicator.
func NewProgressIndicator(width, height int) *ProgressIndicator {
	return &ProgressIndicator{
		Width:  width,
		Height: height,
	}
}

// Render draws the progress indicator.
func (p *ProgressIndicator) Render(fb *eziog500.FrameBuffer, x, y int) {
	// Draw border
	if p.Rounded && p.Height > 4 {
		fb.DrawRoundedRect(x, y, p.Width, p.Height, 2, true)
	} else {
		fb.DrawRect(x, y, p.Width, p.Height, true)
	}

	// Draw fill
	fillWidth := int(float64(p.Width-4) * p.Value / 100.0)
	if fillWidth > 0 {
		fb.FillRect(x+2, y+2, fillWidth, p.Height-4, true)
	}
}

// Label is a simple text label widget.
type Label struct {
	Text     string
	Inverted bool
}

// NewLabel creates a new label.
func NewLabel(text string) *Label {
	return &Label{Text: text}
}

// Render draws the label.
func (l *Label) Render(fb *eziog500.FrameBuffer, x, y int) {
	f := font.BuiltinFont
	if l.Inverted {
		font.RenderTextInverted(fb, f, x, y, l.Text)
	} else {
		font.RenderText(fb, f, x, y, l.Text)
	}
}

func (l *Label) Width() int {
	return font.MeasureText(font.BuiltinFont, l.Text)
}

func (l *Label) Height() int {
	return font.BuiltinFont.Height()
}

// Checkbox represents a toggle checkbox.
type Checkbox struct {
	Label   string
	Checked bool
	size    int
}

// NewCheckbox creates a new checkbox.
func NewCheckbox(label string) *Checkbox {
	return &Checkbox{
		Label: label,
		size:  8,
	}
}

// Toggle toggles the checkbox state.
func (c *Checkbox) Toggle() {
	c.Checked = !c.Checked
}

// Render draws the checkbox.
func (c *Checkbox) Render(fb *eziog500.FrameBuffer, x, y int) {
	f := font.BuiltinFont

	// Draw checkbox box
	fb.DrawRect(x, y, c.size, c.size, true)

	// Draw check mark if checked
	if c.Checked {
		fb.DrawLine(x+2, y+4, x+3, y+6, true)
		fb.DrawLine(x+3, y+6, x+6, y+2, true)
	}

	// Draw label
	font.RenderText(fb, f, x+c.size+4, y, c.Label)
}

func (c *Checkbox) Width() int {
	return c.size + 4 + font.MeasureText(font.BuiltinFont, c.Label)
}

func (c *Checkbox) Height() int { return c.size }

// Divider is a horizontal line separator.
type Divider struct {
	W int // width of divider
}

// NewDivider creates a new horizontal divider.
func NewDivider(width int) *Divider {
	return &Divider{W: width}
}

// Render draws the divider.
func (d *Divider) Render(fb *eziog500.FrameBuffer, x, y int) {
	fb.DrawHLine(x, x+d.W-1, y+1, true)
}

func (d *Divider) Width() int  { return d.W }
func (d *Divider) Height() int { return 3 }

// Icon represents a simple 8x8 icon.
type Icon struct {
	Data [8]byte // 8 bytes, each representing a column
}

// Render draws the icon.
func (i *Icon) Render(fb *eziog500.FrameBuffer, x, y int) {
	for col := 0; col < 8; col++ {
		for bit := 0; bit < 8; bit++ {
			if (i.Data[col] & (1 << bit)) != 0 {
				fb.SetPixel(x+col, y+bit, true)
			}
		}
	}
}

func (i *Icon) Width() int  { return 8 }
func (i *Icon) Height() int { return 8 }

// Predefined icons
var (
	IconArrowUp    = Icon{Data: [8]byte{0x00, 0x04, 0x02, 0xFF, 0x02, 0x04, 0x00, 0x00}}
	IconArrowDown  = Icon{Data: [8]byte{0x00, 0x20, 0x40, 0xFF, 0x40, 0x20, 0x00, 0x00}}
	IconArrowLeft  = Icon{Data: [8]byte{0x00, 0x08, 0x1C, 0x3E, 0x7F, 0x1C, 0x08, 0x00}}
	IconArrowRight = Icon{Data: [8]byte{0x00, 0x08, 0x1C, 0x7F, 0x3E, 0x1C, 0x08, 0x00}}
	IconCheck      = Icon{Data: [8]byte{0x00, 0x60, 0x30, 0x18, 0x0C, 0x06, 0x03, 0x00}}
	IconX          = Icon{Data: [8]byte{0x00, 0x63, 0x36, 0x1C, 0x1C, 0x36, 0x63, 0x00}}
)
