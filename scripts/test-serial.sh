#!/bin/sh
# Direct serial test for EZIO-G500
# Usage: ./test-serial.sh /dev/cuau1
#
# This sends raw commands to the display for testing

PORT=${1:-/dev/cuau1}

if [ ! -e "$PORT" ]; then
    echo "Error: Port $PORT does not exist"
    echo "Available ports:"
    ls /dev/cuau* /dev/ttyS* /dev/ttyU* 2>/dev/null || echo "  (none found)"
    exit 1
fi

echo "Testing EZIO-G500 on $PORT"
echo "================================"

# Configure serial port: 115200 8N1
echo "Configuring serial port..."
if command -v stty >/dev/null 2>&1; then
    # Try FreeBSD style first, then Linux
    stty -f "$PORT" 115200 cs8 -cstopb -parenb raw -echo 2>/dev/null || \
    stty -F "$PORT" 115200 cs8 -cstopb -parenb raw -echo 2>/dev/null || {
        echo "Warning: Could not configure port with stty"
    }
fi

sleep 0.1

echo "Sending Display Init (ESC @)..."
printf '\033@' > "$PORT"
sleep 0.1

echo "Sending Clear Screen (0x0C)..."
printf '\014' > "$PORT"
sleep 0.1

echo "Sending Backlight ON (ESC B 0xFF)..."
printf '\033B\377' > "$PORT"
sleep 0.1

echo "Sending LED 1 Green ON (ESC L 0x21)..."
# LED 1 Green = 0x20, Status ON = 0x01, Combined = 0x21
printf '\033L!' > "$PORT"
sleep 0.3

echo "Sending LED 1 Red ON (ESC L 0x11)..."
# LED 1 Red = 0x10, Status ON = 0x01, Combined = 0x11
printf '\033L\021' > "$PORT"
sleep 0.3

echo "Sending LED 1 OFF (ESC L 0x20, ESC L 0x10)..."
printf '\033L ' > "$PORT"  # 0x20 = green off
printf '\033L\020' > "$PORT"  # 0x10 = red off
sleep 0.1

echo ""
echo "Now sending test image (all white)..."

# ESC G followed by 1024 bytes of 0xFF (all pixels on)
{
    printf '\033G'
    # Send 1024 bytes of 0xFF
    dd if=/dev/zero bs=1 count=1024 2>/dev/null | tr '\000' '\377'
} > "$PORT"

sleep 0.2

echo ""
echo "Test complete!"
echo ""
echo "Expected results:"
echo "  - Display should have cleared"
echo "  - Backlight should be ON (bright)"
echo "  - LED 1 should have flashed green, then red, then off"  
echo "  - Display should show all white (all pixels on)"
echo ""
echo "If nothing happened:"
echo "  1. Check that $PORT is the correct serial port"
echo "  2. Try running as root: sudo $0 $PORT"
echo "  3. Check dmesg for serial port errors"
