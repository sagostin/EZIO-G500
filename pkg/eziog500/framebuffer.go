package eziog500

// FrameBuffer represents a 128x64 pixel graphics buffer for the EZIO-G500 display.
//
// The EZIO-G500 uses vertical byte encoding:
//   - Each byte represents 8 vertical pixels (a column of 8 rows)
//   - Bit 0 = row 0 (top), Bit 7 = row 7 (bottom) within each 8-row band
//   - The display is organized in 8 horizontal bands (64 rows / 8 = 8 bands)
//   - Each band has 128 bytes (one per column)
//
// The display also expects data in a specific order:
//   - Left half (columns 0-63) of all bands, then
//   - Right half (columns 64-127) of all bands
type FrameBuffer struct {
	// data stores pixels in a simple linear format for manipulation
	// Organized as [y][x] where y is 0-63 and x is 0-127
	data [64][128]bool
}

// Width of the display in pixels.
const Width = 128

// Height of the display in pixels.
const Height = 64

// BufferSize is the total bytes needed for the display (128x64 / 8).
const BufferSize = 1024

// NewFrameBuffer creates a new empty framebuffer.
func NewFrameBuffer() *FrameBuffer {
	return &FrameBuffer{}
}

// Clear resets all pixels to off (black).
func (fb *FrameBuffer) Clear() {
	for y := 0; y < Height; y++ {
		for x := 0; x < Width; x++ {
			fb.data[y][x] = false
		}
	}
}

// Fill sets all pixels to on (white).
func (fb *FrameBuffer) Fill() {
	for y := 0; y < Height; y++ {
		for x := 0; x < Width; x++ {
			fb.data[y][x] = true
		}
	}
}

// SetPixel sets a pixel at (x, y) to on or off.
// Coordinates are clipped to display bounds.
func (fb *FrameBuffer) SetPixel(x, y int, on bool) {
	if x < 0 || x >= Width || y < 0 || y >= Height {
		return
	}
	fb.data[y][x] = on
}

// GetPixel returns the state of a pixel at (x, y).
// Returns false for out-of-bounds coordinates.
func (fb *FrameBuffer) GetPixel(x, y int) bool {
	if x < 0 || x >= Width || y < 0 || y >= Height {
		return false
	}
	return fb.data[y][x]
}

// Invert toggles the state of a pixel at (x, y).
func (fb *FrameBuffer) Invert(x, y int) {
	if x < 0 || x >= Width || y < 0 || y >= Height {
		return
	}
	fb.data[y][x] = !fb.data[y][x]
}

// InvertAll inverts all pixels in the framebuffer.
func (fb *FrameBuffer) InvertAll() {
	for y := 0; y < Height; y++ {
		for x := 0; x < Width; x++ {
			fb.data[y][x] = !fb.data[y][x]
		}
	}
}

// ToDeviceFormat converts the framebuffer to the EZIO-G500 wire format.
//
// The wire format is:
//  1. Vertical byte encoding: each byte = 8 vertical pixels (bit 0 = top row)
//  2. Display split: left 64 columns first, then right 64 columns
//  3. Within each half: organized by 8-row bands, left-to-right within each band
//  4. XOR with 0xFF: inverts pixels (matching bmp2lcd Perl script)
//
// With XOR: pixel ON in framebuffer = pixel OFF on display (dark on light)
// This gives a black background with white text (standard LCD appearance)
//
// Total: 1024 bytes = (64 columns * 8 bands) * 2 halves
func (fb *FrameBuffer) ToDeviceFormat() [BufferSize]byte {
	var result [BufferSize]byte

	// First, create the raw vertical-encoded buffer
	// 128 columns x 8 bands = 1024 bytes in column-major order
	var raw [BufferSize]byte

	for x := 0; x < Width; x++ {
		for band := 0; band < 8; band++ {
			var b byte
			for bit := 0; bit < 8; bit++ {
				y := band*8 + bit
				if fb.data[y][x] {
					b |= 1 << bit
				}
			}
			// In raw format: index = band * 128 + x
			raw[band*128+x] = b
		}
	}

	// Now rearrange for the display's expected format:
	// - Left half (columns 0-63) of each band, all bands
	// - Right half (columns 64-127) of each band, all bands

	idx := 0

	// Left half: columns 0-63, all 8 bands
	for band := 0; band < 8; band++ {
		for x := 0; x < 64; x++ {
			// Send raw data without XOR to test
			result[idx] = raw[band*128+x]
			idx++
		}
	}

	// Right half: columns 64-127, all 8 bands
	for band := 0; band < 8; band++ {
		for x := 64; x < 128; x++ {
			// Send raw data without XOR to test
			result[idx] = raw[band*128+x]
			idx++
		}
	}

	return result
}

// FromDeviceFormat populates the framebuffer from device format data.
func (fb *FrameBuffer) FromDeviceFormat(data [BufferSize]byte) {
	// First, reconstruct the raw format
	var raw [BufferSize]byte

	idx := 0

	// Left half: columns 0-63, all 8 bands
	for band := 0; band < 8; band++ {
		for x := 0; x < 64; x++ {
			raw[band*128+x] = data[idx]
			idx++
		}
	}

	// Right half: columns 64-127, all 8 bands
	for band := 0; band < 8; band++ {
		for x := 64; x < 128; x++ {
			raw[band*128+x] = data[idx]
			idx++
		}
	}

	// Now decode vertical bytes to pixels
	for x := 0; x < Width; x++ {
		for band := 0; band < 8; band++ {
			b := raw[band*128+x]
			for bit := 0; bit < 8; bit++ {
				y := band*8 + bit
				fb.data[y][x] = (b & (1 << bit)) != 0
			}
		}
	}
}

// Copy returns a copy of the framebuffer.
func (fb *FrameBuffer) Copy() *FrameBuffer {
	newFB := &FrameBuffer{}
	for y := 0; y < Height; y++ {
		for x := 0; x < Width; x++ {
			newFB.data[y][x] = fb.data[y][x]
		}
	}
	return newFB
}

// DrawHLine draws a horizontal line from (x1, y) to (x2, y).
func (fb *FrameBuffer) DrawHLine(x1, x2, y int, on bool) {
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	for x := x1; x <= x2; x++ {
		fb.SetPixel(x, y, on)
	}
}

// DrawVLine draws a vertical line from (x, y1) to (x, y2).
func (fb *FrameBuffer) DrawVLine(x, y1, y2 int, on bool) {
	if y1 > y2 {
		y1, y2 = y2, y1
	}
	for y := y1; y <= y2; y++ {
		fb.SetPixel(x, y, on)
	}
}

// DrawRect draws a rectangle outline.
func (fb *FrameBuffer) DrawRect(x, y, w, h int, on bool) {
	fb.DrawHLine(x, x+w-1, y, on)     // Top
	fb.DrawHLine(x, x+w-1, y+h-1, on) // Bottom
	fb.DrawVLine(x, y, y+h-1, on)     // Left
	fb.DrawVLine(x+w-1, y, y+h-1, on) // Right
}

// FillRect fills a rectangle.
func (fb *FrameBuffer) FillRect(x, y, w, h int, on bool) {
	for dy := 0; dy < h; dy++ {
		fb.DrawHLine(x, x+w-1, y+dy, on)
	}
}

// DrawLine draws a line from (x1, y1) to (x2, y2) using Bresenham's algorithm.
func (fb *FrameBuffer) DrawLine(x1, y1, x2, y2 int, on bool) {
	dx := abs(x2 - x1)
	dy := -abs(y2 - y1)
	sx := 1
	if x1 > x2 {
		sx = -1
	}
	sy := 1
	if y1 > y2 {
		sy = -1
	}
	err := dx + dy

	for {
		fb.SetPixel(x1, y1, on)
		if x1 == x2 && y1 == y2 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x1 += sx
		}
		if e2 <= dx {
			err += dx
			y1 += sy
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// DrawCircle draws a circle outline using the midpoint circle algorithm.
func (fb *FrameBuffer) DrawCircle(cx, cy, r int, on bool) {
	x := r
	y := 0
	err := 0

	for x >= y {
		fb.SetPixel(cx+x, cy+y, on)
		fb.SetPixel(cx+y, cy+x, on)
		fb.SetPixel(cx-y, cy+x, on)
		fb.SetPixel(cx-x, cy+y, on)
		fb.SetPixel(cx-x, cy-y, on)
		fb.SetPixel(cx-y, cy-x, on)
		fb.SetPixel(cx+y, cy-x, on)
		fb.SetPixel(cx+x, cy-y, on)

		y++
		err += 1 + 2*y
		if 2*(err-x)+1 > 0 {
			x--
			err += 1 - 2*x
		}
	}
}

// FillCircle fills a circle.
func (fb *FrameBuffer) FillCircle(cx, cy, r int, on bool) {
	for y := -r; y <= r; y++ {
		for x := -r; x <= r; x++ {
			if x*x+y*y <= r*r {
				fb.SetPixel(cx+x, cy+y, on)
			}
		}
	}
}

// DrawRoundedRect draws a rectangle with rounded corners.
func (fb *FrameBuffer) DrawRoundedRect(x, y, w, h, r int, on bool) {
	// Top and bottom lines (excluding corners)
	fb.DrawHLine(x+r, x+w-1-r, y, on)
	fb.DrawHLine(x+r, x+w-1-r, y+h-1, on)
	// Left and right lines (excluding corners)
	fb.DrawVLine(x, y+r, y+h-1-r, on)
	fb.DrawVLine(x+w-1, y+r, y+h-1-r, on)
	// Corners
	fb.drawCorner(x+r, y+r, r, 2, on)         // Top-left
	fb.drawCorner(x+w-1-r, y+r, r, 1, on)     // Top-right
	fb.drawCorner(x+r, y+h-1-r, r, 3, on)     // Bottom-left
	fb.drawCorner(x+w-1-r, y+h-1-r, r, 0, on) // Bottom-right
}

// drawCorner draws a quarter circle for rounded rectangle corners.
// quadrant: 0=bottom-right, 1=bottom-left, 2=top-left, 3=top-right
func (fb *FrameBuffer) drawCorner(cx, cy, r, quadrant int, on bool) {
	x := r
	y := 0
	err := 0

	for x >= y {
		switch quadrant {
		case 0: // Bottom-right
			fb.SetPixel(cx+x, cy+y, on)
			fb.SetPixel(cx+y, cy+x, on)
		case 1: // Bottom-left
			fb.SetPixel(cx-x, cy+y, on)
			fb.SetPixel(cx-y, cy+x, on)
		case 2: // Top-left
			fb.SetPixel(cx-x, cy-y, on)
			fb.SetPixel(cx-y, cy-x, on)
		case 3: // Top-right
			fb.SetPixel(cx+x, cy-y, on)
			fb.SetPixel(cx+y, cy-x, on)
		}

		y++
		err += 1 + 2*y
		if 2*(err-x)+1 > 0 {
			x--
			err += 1 - 2*x
		}
	}
}

// DrawTriangle draws a triangle outline.
func (fb *FrameBuffer) DrawTriangle(x1, y1, x2, y2, x3, y3 int, on bool) {
	fb.DrawLine(x1, y1, x2, y2, on)
	fb.DrawLine(x2, y2, x3, y3, on)
	fb.DrawLine(x3, y3, x1, y1, on)
}
