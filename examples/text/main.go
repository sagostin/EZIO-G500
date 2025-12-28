// Example: Basic text display
//
// This example shows how to display text on the EZIO-G500.
package main

import (
	"fmt"
	"os"

	"github.com/sagostin/ezio-g500/pkg/display"
)

func main() {
	// Default port for Checkpoint/pfSense
	port := "/dev/ttyS1"
	if len(os.Args) > 1 {
		port = os.Args[1]
	}

	// Create display
	disp, err := display.New(port)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open display: %v\n", err)
		os.Exit(1)
	}
	defer disp.Close()

	// Clear the display
	disp.Clear()

	// Display text on different lines
	disp.PrintLineCentered(0, "EZIO-G500")
	disp.PrintLine(2, "Line 2 left")
	disp.PrintLineRight(3, "Line 3 right")
	disp.PrintLineCentered(5, "Centered")

	// Send to display
	if err := disp.Update(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to update display: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Text displayed successfully!")
}
