// Example: Graphics and drawing primitives
//
// This example shows how to use the framebuffer for drawing.
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/sagostin/ezio-g500/pkg/display"
	"github.com/sagostin/ezio-g500/pkg/eziog500"
)

func main() {
	port := "/dev/ttyS1"
	if len(os.Args) > 1 {
		port = os.Args[1]
	}

	disp, err := display.New(port)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open display: %v\n", err)
		os.Exit(1)
	}
	defer disp.Close()

	fb := disp.FrameBuffer()

	// Demo 1: Border and diagonals
	fmt.Println("Drawing border and diagonals...")
	fb.Clear()
	fb.DrawRect(0, 0, eziog500.Width, eziog500.Height, true)
	fb.DrawLine(0, 0, eziog500.Width-1, eziog500.Height-1, true)
	fb.DrawLine(eziog500.Width-1, 0, 0, eziog500.Height-1, true)
	disp.Update()
	time.Sleep(2 * time.Second)

	// Demo 2: Filled rectangles
	fmt.Println("Drawing filled rectangles...")
	fb.Clear()
	fb.FillRect(10, 10, 30, 20, true)
	fb.FillRect(50, 20, 30, 30, true)
	fb.FillRect(90, 10, 30, 20, true)
	disp.Update()
	time.Sleep(2 * time.Second)

	// Demo 3: Pattern
	fmt.Println("Drawing checkerboard pattern...")
	fb.Clear()
	for y := 0; y < eziog500.Height; y += 8 {
		for x := 0; x < eziog500.Width; x += 8 {
			if (x/8+y/8)%2 == 0 {
				fb.FillRect(x, y, 8, 8, true)
			}
		}
	}
	disp.Update()
	time.Sleep(2 * time.Second)

	// Demo 4: Animation
	fmt.Println("Running animation...")
	for i := 0; i < 50; i++ {
		fb.Clear()
		x := i * 2 % eziog500.Width
		y := 20 + (i % 20)
		if i/20%2 == 1 {
			y = 40 - (i % 20)
		}
		fb.FillRect(x, y, 20, 20, true)
		disp.Update()
		time.Sleep(50 * time.Millisecond)
	}

	// Final
	fb.Clear()
	disp.PrintLineCentered(3, "DEMO COMPLETE")
	disp.Update()

	fmt.Println("Graphics demo complete!")
}
