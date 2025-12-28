package font

import (
	"testing"

	"github.com/sagostin/ezio-g500/pkg/eziog500"
)

func TestBuiltinFont_GetGlyph(t *testing.T) {
	font := BuiltinFont

	// Test that all standard characters have glyphs
	chars := "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 .:,-+/"

	for _, c := range chars {
		glyph := font.GetGlyph(c)
		if glyph == nil {
			t.Errorf("No glyph for character '%c'", c)
		}
		if len(glyph) == 0 {
			t.Errorf("Empty glyph for character '%c'", c)
		}
	}
}

func TestBuiltinFont_LowercaseToUppercase(t *testing.T) {
	font := BuiltinFont

	// Lowercase and uppercase should produce the same glyph
	upper := font.GetGlyph('A')
	lower := font.GetGlyph('a')

	if len(upper) != len(lower) {
		t.Error("Uppercase and lowercase 'A' should produce same glyph")
	}

	for i := range upper {
		if upper[i] != lower[i] {
			t.Error("Uppercase and lowercase 'A' glyphs should be identical")
			break
		}
	}
}

func TestBuiltinFont_Height(t *testing.T) {
	if BuiltinFont.Height() != 8 {
		t.Errorf("Expected height 8, got %d", BuiltinFont.Height())
	}

	if SmallFont.Height() != 6 {
		t.Errorf("Expected small font height 6, got %d", SmallFont.Height())
	}
}

func TestBuiltinFont_GetWidth(t *testing.T) {
	font := BuiltinFont

	// Check some known character widths
	if font.GetWidth('A') == 0 {
		t.Error("'A' should have non-zero width")
	}

	// Space should have a width
	if font.GetWidth(' ') == 0 {
		t.Error("Space should have non-zero width")
	}
}

func TestRenderText(t *testing.T) {
	fb := eziog500.NewFrameBuffer()
	font := BuiltinFont

	// Render some text
	endX := RenderText(fb, font, 0, 0, "ABC")

	// Should return position after text
	if endX == 0 {
		t.Error("RenderText should return non-zero position after rendering")
	}

	// Check that some pixels were set (text should have visible pixels)
	hasPixels := false
	for y := 0; y < 8; y++ {
		for x := 0; x < endX; x++ {
			if fb.GetPixel(x, y) {
				hasPixels = true
				break
			}
		}
		if hasPixels {
			break
		}
	}

	if !hasPixels {
		t.Error("Rendered text should have some pixels set")
	}
}

func TestRenderText_AtPosition(t *testing.T) {
	fb := eziog500.NewFrameBuffer()
	font := BuiltinFont

	// Render at offset position
	RenderText(fb, font, 50, 20, "X")

	// Origin should be empty
	if fb.GetPixel(0, 0) {
		t.Error("Origin should be empty when rendering at offset")
	}

	// Near the render position should have pixels
	hasPixels := false
	for y := 20; y < 28; y++ {
		for x := 50; x < 60; x++ {
			if fb.GetPixel(x, y) {
				hasPixels = true
				break
			}
		}
	}

	if !hasPixels {
		t.Error("Text rendered at (50,20) should have pixels near that position")
	}
}

func TestMeasureText(t *testing.T) {
	font := BuiltinFont

	// Empty text should have zero width
	if MeasureText(font, "") != 0 {
		t.Error("Empty text should have zero width")
	}

	// Single character
	singleWidth := MeasureText(font, "A")
	if singleWidth == 0 {
		t.Error("Single character should have non-zero width")
	}

	// Multiple characters should be wider
	multiWidth := MeasureText(font, "ABC")
	if multiWidth <= singleWidth {
		t.Error("Multiple characters should be wider than single character")
	}
}

func TestSmallFont_BasicOperation(t *testing.T) {
	font := SmallFont

	// Should have glyphs for basic characters
	if font.GetGlyph('A') == nil {
		t.Error("SmallFont should have glyph for 'A'")
	}

	if font.GetGlyph('0') == nil {
		t.Error("SmallFont should have glyph for '0'")
	}

	// Should be shorter than builtin font
	if font.Height() >= BuiltinFont.Height() {
		t.Error("SmallFont should be shorter than BuiltinFont")
	}
}
