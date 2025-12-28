// Package font provides font rendering for the EZIO-G500 display.
package font

import (
	"github.com/sagostin/ezio-g500/pkg/eziog500"
)

// Font represents a bitmap font for rendering text on the display.
type Font interface {
	// GetGlyph returns the pixel data for a character.
	// Each byte in the slice represents a column (8 vertical pixels).
	// Returns nil if the character is not supported.
	GetGlyph(r rune) []byte

	// Height returns the font height in pixels.
	Height() int

	// GetWidth returns the width of a character in pixels.
	// Returns 0 if the character is not supported.
	GetWidth(r rune) int
}

// RenderText renders text to the framebuffer at the specified position.
// Returns the x position after the last character.
func RenderText(fb *eziog500.FrameBuffer, f Font, x, y int, text string) int {
	curX := x
	for _, r := range text {
		glyph := f.GetGlyph(r)
		if glyph == nil {
			// Skip unknown characters
			continue
		}

		// Render each column of the glyph
		for col, b := range glyph {
			for bit := 0; bit < 8; bit++ {
				if (b & (1 << bit)) != 0 {
					fb.SetPixel(curX+col, y+bit, true)
				}
			}
		}
		curX += len(glyph)
	}
	return curX
}

// RenderTextInverted renders inverted text (white on black background).
func RenderTextInverted(fb *eziog500.FrameBuffer, f Font, x, y int, text string) int {
	// First, calculate width
	width := MeasureText(f, text)

	// Draw background
	fb.FillRect(x, y, width, f.Height(), true)

	// Render text with inverted logic
	curX := x
	for _, r := range text {
		glyph := f.GetGlyph(r)
		if glyph == nil {
			continue
		}

		for col, b := range glyph {
			for bit := 0; bit < 8; bit++ {
				if (b & (1 << bit)) != 0 {
					// Set to off (black) where glyph is on
					fb.SetPixel(curX+col, y+bit, false)
				}
			}
		}
		curX += len(glyph)
	}
	return curX
}

// MeasureText returns the width in pixels of the rendered text.
func MeasureText(f Font, text string) int {
	width := 0
	for _, r := range text {
		glyph := f.GetGlyph(r)
		if glyph != nil {
			width += len(glyph)
		}
	}
	return width
}

// MeasureTextRunes returns the width for a slice of runes.
func MeasureTextRunes(f Font, runes []rune) int {
	width := 0
	for _, r := range runes {
		w := f.GetWidth(r)
		if w > 0 {
			width += w
		}
	}
	return width
}
