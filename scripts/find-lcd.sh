#!/bin/sh
# EZIO-G500 Serial Port Discovery Script
# Tests all available serial devices to find the LCD display
#
# Usage: ./find-lcd.sh
#
# This script attempts to communicate with the EZIO-G500 display
# on each available serial port to identify the correct one.

set -e

# Colors for output (if terminal supports it)
if [ -t 1 ]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    NC='\033[0m' # No Color
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    NC=''
fi

echo "${BLUE}======================================${NC}"
echo "${BLUE}  EZIO-G500 Serial Port Discovery${NC}"
echo "${BLUE}======================================${NC}"
echo ""

# Detect OS
OS=$(uname -s)
echo "Operating System: ${OS}"
echo ""

# Find all serial devices based on OS
find_serial_devices() {
    case $OS in
        FreeBSD)
            # FreeBSD uses cuau* for dial-out, ttyu* for dial-in
            # Also check ttyS* which some systems use
            ls /dev/cuau* /dev/ttyu* /dev/ttyS* /dev/ttyU* 2>/dev/null || true
            ;;
        Linux)
            # Linux: ttyS*, ttyUSB*, ttyACM*, ttyAMA* (Raspberry Pi)
            ls /dev/ttyS* /dev/ttyUSB* /dev/ttyACM* /dev/ttyAMA* 2>/dev/null || true
            ;;
        Darwin)
            # macOS: cu.* and tty.*
            ls /dev/cu.* /dev/tty.* 2>/dev/null | grep -v Bluetooth || true
            ;;
        *)
            echo "Unknown OS: $OS"
            ls /dev/tty* 2>/dev/null | head -20 || true
            ;;
    esac
}

# Test a single serial port
test_port() {
    port=$1
    
    # Check if device exists and is readable
    if [ ! -e "$port" ]; then
        return 1
    fi
    
    if [ ! -r "$port" ] || [ ! -w "$port" ]; then
        echo "  ${YELLOW}Permission denied${NC}"
        return 1
    fi
    
    # Configure the port (115200 8N1)
    if command -v stty >/dev/null 2>&1; then
        stty -F "$port" 115200 cs8 -cstopb -parenb raw -echo 2>/dev/null || \
        stty -f "$port" 115200 cs8 -cstopb -parenb raw -echo 2>/dev/null || \
        return 1
    fi
    
    # Try to send init command and read response
    # EZIO-G500 Init: ESC @ (0x1B 0x40)
    # Also try clear: 0x0C
    
    # Send init command
    printf '\033@' > "$port" 2>/dev/null
    sleep 0.1
    
    # Send clear command
    printf '\014' > "$port" 2>/dev/null
    sleep 0.1
    
    # Try to send a test pattern and see if we get any response
    # The display doesn't typically respond, but we can check if the
    # port is functional by successfully writing
    
    # Send backlight command (ESC B 255)
    printf '\033B\377' > "$port" 2>/dev/null
    sleep 0.1
    
    # If we got here without errors, the port is likely functional
    return 0
}

# Advanced test - tries to draw something on the display
full_test_port() {
    port=$1
    
    echo "${BLUE}Running full test on $port...${NC}"
    
    # Configure port
    if command -v stty >/dev/null 2>&1; then
        stty -F "$port" 115200 cs8 -cstopb -parenb raw -echo 2>/dev/null || \
        stty -f "$port" 115200 cs8 -cstopb -parenb raw -echo 2>/dev/null || {
            echo "  ${RED}Failed to configure port${NC}"
            return 1
        }
    fi
    
    # Init
    printf '\033@' > "$port"
    sleep 0.1
    
    # Clear
    printf '\014' > "$port"
    sleep 0.1
    
    # Backlight on (ESC B 200)
    printf '\033B\310' > "$port"
    sleep 0.1
    
    # Try to set LED 1 to green (ESC L 0x20 0x01)
    printf '\033L \001' > "$port"
    sleep 0.2
    
    # Flash LED to red (ESC L 0x10 0x01)
    printf '\033L\020\001' > "$port"
    sleep 0.5
    
    # Back to green
    printf '\033L \001' > "$port"
    
    echo "  ${GREEN}Commands sent successfully${NC}"
    echo "  ${YELLOW}Check the LCD display for:${NC}"
    echo "    - Display should have cleared"
    echo "    - Backlight should be on"
    echo "    - LED 1 should have flashed red, then turned green"
    echo ""
    
    return 0
}

# Get list of devices
echo "Scanning for serial devices..."
echo ""

DEVICES=$(find_serial_devices)

if [ -z "$DEVICES" ]; then
    echo "${RED}No serial devices found!${NC}"
    echo ""
    echo "Tips:"
    echo "  - Make sure you're running as root (sudo)"
    echo "  - Check if serial ports are enabled in BIOS"
    echo "  - On pfSense, internal serial ports are typically /dev/cuau0, /dev/cuau1"
    exit 1
fi

echo "Found devices:"
echo "$DEVICES" | while read dev; do
    echo "  $dev"
done
echo ""

# Test each device
echo "${BLUE}Testing each device...${NC}"
echo ""

WORKING_PORTS=""

for port in $DEVICES; do
    printf "Testing ${YELLOW}%s${NC}... " "$port"
    
    if test_port "$port" 2>/dev/null; then
        echo "${GREEN}OK${NC} (writable)"
        WORKING_PORTS="$WORKING_PORTS $port"
    else
        echo "${RED}Failed${NC}"
    fi
done

echo ""

if [ -z "$WORKING_PORTS" ]; then
    echo "${RED}No working serial ports found.${NC}"
    echo ""
    echo "Tips for pfSense/FreeBSD:"
    echo "  - Run as root: sudo ./find-lcd.sh"
    echo "  - Common EZIO-G500 ports: /dev/cuau1, /dev/ttyS1"
    echo "  - Check /var/log/messages for serial port detection"
    exit 1
fi

echo "${GREEN}Working ports:${NC}$WORKING_PORTS"
echo ""

# Ask user if they want to run full test
echo "Would you like to run a full visual test on a port?"
echo "This will flash the LED and turn on the backlight."
echo ""

for port in $WORKING_PORTS; do
    printf "Test ${YELLOW}%s${NC}? [y/N] " "$port"
    read answer
    case $answer in
        [Yy]*)
            full_test_port "$port"
            echo ""
            printf "Did the display respond correctly? [y/N] "
            read confirm
            case $confirm in
                [Yy]*)
                    echo ""
                    echo "${GREEN}======================================${NC}"
                    echo "${GREEN}  SUCCESS! Found LCD on: $port${NC}"
                    echo "${GREEN}======================================${NC}"
                    echo ""
                    echo "To use this port with eziolcd:"
                    echo "  eziolcd -port $port status"
                    echo "  eziolcd -port $port menu"
                    echo "  eziolcd -port $port daemon"
                    echo ""
                    echo "For permanent configuration, add to /etc/rc.conf:"
                    echo "  eziolcd_port=\"$port\""
                    echo ""
                    exit 0
                    ;;
            esac
            ;;
    esac
done

echo ""
echo "No confirmed working port found."
echo "Try testing ports manually with:"
echo "  eziolcd -port /dev/cuauX status"
echo ""
echo "Replace X with the port number (0, 1, 2, etc.)"
