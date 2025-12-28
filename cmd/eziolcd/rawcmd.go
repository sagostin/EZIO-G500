package main

import (
	"fmt"
	"os"

	"github.com/sagostin/ezio-g500/pkg/eziog500"
)

// cmdRaw sends raw ASCII text directly to the display (like typing in cu)
// This bypasses all ESC commands and graphics mode
func cmdRaw(text string) error {
	// Enable verbose if flag is set
	if *verbose {
		eziog500.SetVerbose(true)
		fmt.Fprintf(os.Stderr, "[RAW] Opening port: %s\n", *portPath)
	}

	// Open without any stty configuration to see if that's the issue
	device, err := eziog500.OpenWithoutStty(*portPath)
	if err != nil {
		return fmt.Errorf("failed to open device: %w", err)
	}
	defer device.Close()

	if *verbose {
		fmt.Fprintf(os.Stderr, "[RAW] Sending raw text: %q\n", text)
	}

	// Just write raw bytes directly - exactly like typing in cu
	if err := device.Write([]byte(text)); err != nil {
		return fmt.Errorf("failed to write: %w", err)
	}

	if *verbose {
		fmt.Fprintf(os.Stderr, "[RAW] Done\n")
	}

	return nil
}
