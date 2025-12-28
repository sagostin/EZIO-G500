package eziog500

// LED constants for EZIO-G500 bicolor LEDs.
// Each LED has red and green components that can be combined for orange.

// LED identifies one of the three LEDs on the display.
type LED int

const (
	LED1 LED = iota
	LED2
	LED3
)

// LEDColor represents the color state of an LED.
type LEDColor int

const (
	LEDOff    LEDColor = iota // LED is off
	LEDRed                    // LED is red
	LEDGreen                  // LED is green
	LEDOrange                 // LED is orange (both red and green)
)

// LED control command values from protocol analysis.
// The command format is: ESC L <ledid + status>
// Where ledid is the high nibble and status is bit 0.
const (
	led1Red   byte = 0x10
	led1Green byte = 0x20
	led2Red   byte = 0x30
	led2Green byte = 0x40
	led3Red   byte = 0x50
	led3Green byte = 0x60

	ledStatusOn  byte = 0x01
	ledStatusOff byte = 0x00
)

// SetLED sets the color of the specified LED.
// To turn off an LED, use LEDOff.
// For orange, the function sends both red and green commands.
//
// Protocol: ESC L (ledid | status) - 3 bytes total
// The ledid is the high nibble (0x10-0x60), status is bit 0 (0 or 1)
func (d *Device) SetLED(led LED, color LEDColor) error {
	var redCmd, greenCmd byte

	switch led {
	case LED1:
		redCmd = led1Red
		greenCmd = led1Green
	case LED2:
		redCmd = led2Red
		greenCmd = led2Green
	case LED3:
		redCmd = led3Red
		greenCmd = led3Green
	default:
		redCmd = led1Red
		greenCmd = led1Green
	}

	switch color {
	case LEDOff:
		// Turn off both red and green
		if err := d.Write([]byte{ESC, cmdLED, redCmd | ledStatusOff}); err != nil {
			return err
		}
		if err := d.Write([]byte{ESC, cmdLED, greenCmd | ledStatusOff}); err != nil {
			return err
		}
	case LEDRed:
		// Green off, red on
		if err := d.Write([]byte{ESC, cmdLED, greenCmd | ledStatusOff}); err != nil {
			return err
		}
		if err := d.Write([]byte{ESC, cmdLED, redCmd | ledStatusOn}); err != nil {
			return err
		}
	case LEDGreen:
		// Red off, green on
		if err := d.Write([]byte{ESC, cmdLED, redCmd | ledStatusOff}); err != nil {
			return err
		}
		if err := d.Write([]byte{ESC, cmdLED, greenCmd | ledStatusOn}); err != nil {
			return err
		}
	case LEDOrange:
		// Both red and green on
		if err := d.Write([]byte{ESC, cmdLED, redCmd | ledStatusOn}); err != nil {
			return err
		}
		if err := d.Write([]byte{ESC, cmdLED, greenCmd | ledStatusOn}); err != nil {
			return err
		}
	}

	// Flush immediately so LED changes are visible
	return d.Flush()
}

// SetLEDRaw sends a raw LED control command.
// The value should be (ledId | status) where ledId is 0x10-0x60 and status is 0 or 1.
func (d *Device) SetLEDRaw(value byte) error {
	return d.Write([]byte{ESC, cmdLED, value})
}
