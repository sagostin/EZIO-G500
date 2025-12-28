package eziog500

import (
	"testing"
)

func TestFrameBuffer_SetGetPixel(t *testing.T) {
	fb := NewFrameBuffer()

	// Initially all pixels should be off
	if fb.GetPixel(0, 0) {
		t.Error("Expected pixel (0,0) to be off initially")
	}

	// Set a pixel
	fb.SetPixel(10, 20, true)
	if !fb.GetPixel(10, 20) {
		t.Error("Expected pixel (10,20) to be on after setting")
	}

	// Clear the pixel
	fb.SetPixel(10, 20, false)
	if fb.GetPixel(10, 20) {
		t.Error("Expected pixel (10,20) to be off after clearing")
	}

	// Test bounds checking
	fb.SetPixel(-1, 0, true)  // Should not panic
	fb.SetPixel(0, -1, true)  // Should not panic
	fb.SetPixel(128, 0, true) // Should not panic
	fb.SetPixel(0, 64, true)  // Should not panic

	if fb.GetPixel(-1, 0) {
		t.Error("Out of bounds pixel should return false")
	}
}

func TestFrameBuffer_Clear(t *testing.T) {
	fb := NewFrameBuffer()

	// Set some pixels
	fb.SetPixel(50, 30, true)
	fb.SetPixel(100, 60, true)

	// Clear
	fb.Clear()

	// All should be off
	if fb.GetPixel(50, 30) || fb.GetPixel(100, 60) {
		t.Error("All pixels should be off after Clear()")
	}
}

func TestFrameBuffer_Fill(t *testing.T) {
	fb := NewFrameBuffer()

	fb.Fill()

	// All should be on
	for y := 0; y < Height; y++ {
		for x := 0; x < Width; x++ {
			if !fb.GetPixel(x, y) {
				t.Errorf("Pixel (%d,%d) should be on after Fill()", x, y)
				return
			}
		}
	}
}

func TestFrameBuffer_Invert(t *testing.T) {
	fb := NewFrameBuffer()

	fb.SetPixel(10, 10, true)
	fb.Invert(10, 10)

	if fb.GetPixel(10, 10) {
		t.Error("Pixel should be off after inversion")
	}

	fb.Invert(10, 10)
	if !fb.GetPixel(10, 10) {
		t.Error("Pixel should be on after second inversion")
	}
}

func TestFrameBuffer_ToDeviceFormat(t *testing.T) {
	fb := NewFrameBuffer()

	// Set a pattern to test encoding
	fb.SetPixel(0, 0, true)
	fb.SetPixel(0, 7, true)
	fb.SetPixel(64, 0, true)

	data := fb.ToDeviceFormat()

	// Check total size
	if len(data) != BufferSize {
		t.Errorf("Expected %d bytes, got %d", BufferSize, len(data))
	}

	// Verify the data is not all zeros
	allZero := true
	for _, b := range data {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("Device format should contain non-zero data")
	}
}

func TestFrameBuffer_RoundTrip(t *testing.T) {
	fb := NewFrameBuffer()

	// Set a complex pattern
	for i := 0; i < 100; i++ {
		x := (i * 7) % Width
		y := (i * 11) % Height
		fb.SetPixel(x, y, true)
	}

	// Convert to device format and back
	data := fb.ToDeviceFormat()

	fb2 := NewFrameBuffer()
	fb2.FromDeviceFormat(data)

	// Compare all pixels
	for y := 0; y < Height; y++ {
		for x := 0; x < Width; x++ {
			if fb.GetPixel(x, y) != fb2.GetPixel(x, y) {
				t.Errorf("Pixel mismatch at (%d,%d): original=%v, restored=%v",
					x, y, fb.GetPixel(x, y), fb2.GetPixel(x, y))
			}
		}
	}
}

func TestFrameBuffer_DrawHLine(t *testing.T) {
	fb := NewFrameBuffer()

	fb.DrawHLine(10, 50, 20, true)

	// Check the line
	for x := 10; x <= 50; x++ {
		if !fb.GetPixel(x, 20) {
			t.Errorf("Pixel (%d,20) should be on", x)
		}
	}

	// Check outside the line
	if fb.GetPixel(9, 20) || fb.GetPixel(51, 20) {
		t.Error("Pixels outside the line should be off")
	}
}

func TestFrameBuffer_DrawVLine(t *testing.T) {
	fb := NewFrameBuffer()

	fb.DrawVLine(30, 10, 50, true)

	// Check the line
	for y := 10; y <= 50; y++ {
		if !fb.GetPixel(30, y) {
			t.Errorf("Pixel (30,%d) should be on", y)
		}
	}
}

func TestFrameBuffer_DrawRect(t *testing.T) {
	fb := NewFrameBuffer()

	fb.DrawRect(10, 10, 20, 15, true)

	// Check corners
	if !fb.GetPixel(10, 10) || !fb.GetPixel(29, 10) ||
		!fb.GetPixel(10, 24) || !fb.GetPixel(29, 24) {
		t.Error("Rectangle corners should be on")
	}

	// Check that interior is not filled
	if fb.GetPixel(15, 15) {
		t.Error("Rectangle interior should not be filled by DrawRect")
	}
}

func TestFrameBuffer_FillRect(t *testing.T) {
	fb := NewFrameBuffer()

	fb.FillRect(10, 10, 20, 15, true)

	// Check interior
	for y := 10; y < 25; y++ {
		for x := 10; x < 30; x++ {
			if !fb.GetPixel(x, y) {
				t.Errorf("Pixel (%d,%d) should be on inside filled rect", x, y)
			}
		}
	}

	// Check outside
	if fb.GetPixel(9, 15) || fb.GetPixel(30, 15) {
		t.Error("Pixels outside rectangle should be off")
	}
}

func TestFrameBuffer_DrawLine(t *testing.T) {
	fb := NewFrameBuffer()

	// Horizontal line
	fb.DrawLine(0, 0, 50, 0, true)
	if !fb.GetPixel(0, 0) || !fb.GetPixel(25, 0) || !fb.GetPixel(50, 0) {
		t.Error("Horizontal line should have start, middle, and end pixels set")
	}

	// Diagonal line
	fb.Clear()
	fb.DrawLine(0, 0, 20, 20, true)
	if !fb.GetPixel(0, 0) || !fb.GetPixel(10, 10) || !fb.GetPixel(20, 20) {
		t.Error("Diagonal line should have start, middle, and end pixels set")
	}

	// Steep line
	fb.Clear()
	fb.DrawLine(0, 0, 5, 20, true)
	if !fb.GetPixel(0, 0) || !fb.GetPixel(5, 20) {
		t.Error("Steep line should have start and end pixels set")
	}
}

func TestFrameBuffer_Copy(t *testing.T) {
	fb := NewFrameBuffer()
	fb.SetPixel(50, 30, true)

	copy := fb.Copy()

	// Modify original
	fb.SetPixel(50, 30, false)

	// Copy should still have the pixel set
	if !copy.GetPixel(50, 30) {
		t.Error("Copy should be independent of original")
	}
}
