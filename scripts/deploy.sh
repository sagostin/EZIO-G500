#!/bin/bash
# Deploy and test EZIO-G500 on remote pfSense/FreeBSD system
#
# Usage: ./deploy.sh [command]
#
# Commands:
#   build     - Build the FreeBSD binary
#   deploy    - Build and deploy to remote system
#   test      - Run a quick test on the remote system
#   shell     - Open SSH shell to remote system
#   all       - Build, deploy, and test (default)
#
# Note: pfSense shows a console menu on SSH login. This script handles that
# by using SFTP for file transfer and piping commands through SSH.

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
BINARY_NAME="eziolcd-freebsd-amd64"

# Load environment variables
if [ -f "$SCRIPT_DIR/.env" ]; then
    source "$SCRIPT_DIR/.env"
else
    echo "Error: $SCRIPT_DIR/.env not found!"
    echo "Copy .env.example to .env and configure it."
    exit 1
fi

# Validate required variables
: ${EZIO_HOST:?'EZIO_HOST not set in .env'}
: ${EZIO_USER:?'EZIO_USER not set in .env'}
: ${EZIO_PASS:?'EZIO_PASS not set in .env'}
: ${EZIO_PORT:='/dev/cuau1'}
: ${EZIO_INSTALL_PATH:='/usr/local/bin/eziolcd'}

SSH_TARGET="${EZIO_USER}@${EZIO_HOST}"

# Check for sshpass
HAS_SSHPASS=false
if command -v sshpass &> /dev/null; then
    HAS_SSHPASS=true
fi

# Helper function to upload files via SFTP (works better with pfSense)
run_sftp() {
    local local_file="$1"
    local remote_path="$2"
    
    # Create SFTP batch file
    local batch_file=$(mktemp)
    echo "put \"$local_file\" \"$remote_path\"" > "$batch_file"
    echo "chmod 755 \"$remote_path\"" >> "$batch_file"
    echo "bye" >> "$batch_file"
    
    if [ "$HAS_SSHPASS" = true ]; then
        # Use SSHPASS environment variable for sftp
        SSHPASS="$EZIO_PASS" sshpass -e sftp -oBatchMode=no -o StrictHostKeyChecking=no -b "$batch_file" "$SSH_TARGET"
    else
        sftp -o StrictHostKeyChecking=no -b "$batch_file" "$SSH_TARGET"
    fi
    
    local result=$?
    rm -f "$batch_file"
    return $result
}

# Helper function to run SSH commands on pfSense
# Pipes the command with "8" to select shell first
run_pfsense_cmd() {
    local cmd="$1"
    
    if [ "$HAS_SSHPASS" = true ]; then
        # Try direct command first, then with menu selection
        (echo "8"; sleep 0.3; echo "$cmd"; sleep 0.5; echo "exit") | \
            sshpass -p "$EZIO_PASS" ssh -o StrictHostKeyChecking=no -t "$SSH_TARGET" 2>/dev/null || true
    else
        (echo "8"; sleep 0.3; echo "$cmd"; sleep 0.5; echo "exit") | \
            ssh -o StrictHostKeyChecking=no -t "$SSH_TARGET" 2>/dev/null || true
    fi
}

# Run command directly without menu handling (for systems without menu)
run_ssh_direct() {
    local cmd="$1"
    
    if [ "$HAS_SSHPASS" = true ]; then
        sshpass -p "$EZIO_PASS" ssh -o StrictHostKeyChecking=no "$SSH_TARGET" "$cmd"
    else
        ssh "$SSH_TARGET" "$cmd"
    fi
}

cmd_build() {
    echo "=== Building FreeBSD binary ==="
    cd "$PROJECT_DIR"
    
    # Build
    echo "Compiling for FreeBSD amd64..."
    GOOS=freebsd GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o "$BINARY_NAME" ./cmd/eziolcd
    
    echo "Built: $BINARY_NAME ($(ls -lh "$BINARY_NAME" | awk '{print $5}'))"
}

cmd_deploy() {
    cmd_build
    
    echo ""
    echo "=== Deploying to $EZIO_HOST ==="
    
    # Upload binary via SFTP (more reliable with pfSense)
    echo "Uploading binary via SFTP..."
    run_sftp "$PROJECT_DIR/$BINARY_NAME" "${EZIO_INSTALL_PATH}"
    
    echo "Deployment complete! Binary installed to ${EZIO_INSTALL_PATH}"
}

cmd_test() {
    echo "=== Testing on $EZIO_HOST ==="
    
    echo "Configuring serial port and running test..."
    run_pfsense_cmd "stty -f ${EZIO_PORT} 115200 cs8 -parenb -cstopb clocal raw -echo -icanon; ${EZIO_INSTALL_PATH} -port ${EZIO_PORT} text 'HELLO PFSENSE'"
    
    echo ""
    echo "Test complete! Check the LCD display."
}

cmd_demo() {
    echo "=== Running demo on $EZIO_HOST ==="
    run_pfsense_cmd "${EZIO_INSTALL_PATH} -port ${EZIO_PORT} demo"
}

cmd_status() {
    echo "=== Showing status on $EZIO_HOST ==="
    run_pfsense_cmd "${EZIO_INSTALL_PATH} -port ${EZIO_PORT} status"
}

cmd_clear() {
    echo "=== Clearing display on $EZIO_HOST ==="
    run_pfsense_cmd "${EZIO_INSTALL_PATH} -port ${EZIO_PORT} clear"
}

cmd_shell() {
    echo "=== Opening SSH shell to $EZIO_HOST ==="
    echo "Note: Select option 8 for shell if you see the pfSense menu"
    if [ "$HAS_SSHPASS" = true ]; then
        sshpass -p "$EZIO_PASS" ssh -o StrictHostKeyChecking=no "$SSH_TARGET"
    else
        ssh "$SSH_TARGET"
    fi
}

cmd_rawtest() {
    echo "=== Raw serial test on $EZIO_HOST ==="
    echo "Configuring port and sending raw text..."
    run_pfsense_cmd "stty -f ${EZIO_PORT} 115200 cs8 -parenb -cstopb clocal raw -echo -icanon && printf '\033@\013\014HELLO LCD' > ${EZIO_PORT}"
    echo "Sent: ESC @ (init), 0x0B (home), 0x0C (clear), 'HELLO LCD'"
}

cmd_backlight() {
    local level=${2:-255}
    echo "=== Setting backlight to $level on $EZIO_HOST ==="
    run_pfsense_cmd "${EZIO_INSTALL_PATH} -port ${EZIO_PORT} backlight $level"
}

cmd_install() {
    echo "=== Full Installation on $EZIO_HOST ==="
    
    # First, build and deploy the binary
    cmd_deploy
    
    echo ""
    echo "=== Installing service ==="
    
    # Upload the install script
    echo "Uploading install script..."
    run_sftp "$SCRIPT_DIR/install-pfsense.sh" "/tmp/install-eziolcd.sh"
    
    # Run the install script via SSH
    echo "Running install script on remote system..."
    if [ "$HAS_SSHPASS" = true ]; then
        SSHPASS="$EZIO_PASS" sshpass -e ssh -o StrictHostKeyChecking=no "$SSH_TARGET" \
            "chmod +x /tmp/install-eziolcd.sh && /tmp/install-eziolcd.sh && rm /tmp/install-eziolcd.sh"
    else
        ssh -o StrictHostKeyChecking=no "$SSH_TARGET" \
            "chmod +x /tmp/install-eziolcd.sh && /tmp/install-eziolcd.sh && rm /tmp/install-eziolcd.sh"
    fi
    
    echo ""
    echo "=== Installation Complete! ==="
    echo "The eziolcd service is now running on $EZIO_HOST"
    echo "Use 'service eziolcd status' on the device to check"
}

cmd_restart() {
    echo "=== Restarting service on $EZIO_HOST ==="
    if [ "$HAS_SSHPASS" = true ]; then
        SSHPASS="$EZIO_PASS" sshpass -e ssh -o StrictHostKeyChecking=no "$SSH_TARGET" \
            "service eziolcd restart"
    else
        ssh -o StrictHostKeyChecking=no "$SSH_TARGET" "service eziolcd restart"
    fi
    echo "Service restarted"
}

cmd_stop() {
    echo "=== Stopping service on $EZIO_HOST ==="
    if [ "$HAS_SSHPASS" = true ]; then
        SSHPASS="$EZIO_PASS" sshpass -e ssh -o StrictHostKeyChecking=no "$SSH_TARGET" \
            "service eziolcd stop; killall eziolcd 2>/dev/null || true"
    else
        ssh -o StrictHostKeyChecking=no "$SSH_TARGET" "service eziolcd stop; killall eziolcd 2>/dev/null || true"
    fi
    echo "Service stopped"
}

# Print usage
usage() {
    echo "EZIO-G500 Deploy Script"
    echo ""
    echo "Usage: $0 [command]"
    echo ""
    echo "Commands:"
    echo "  build      - Build the FreeBSD binary only"
    echo "  deploy     - Build and deploy binary to remote system"
    echo "  install    - Full install: deploy + create service + start"
    echo "  restart    - Restart the eziolcd service"
    echo "  stop       - Stop the eziolcd service"
    echo "  test       - Run a quick text test"
    echo "  demo       - Run the demo on remote system"
    echo "  status     - Show system status on display"
    echo "  clear      - Clear the display"
    echo "  backlight  - Set backlight (0-255)"
    echo "  rawtest    - Raw serial test (no binary needed)"
    echo "  shell      - Open SSH shell to remote system"
    echo "  all        - Build, deploy, and test (default)"
    echo ""
    echo "Configuration: scripts/.env"
    echo "  EZIO_HOST=$EZIO_HOST"
    echo "  EZIO_USER=$EZIO_USER"
    echo "  EZIO_PORT=$EZIO_PORT"
    if [ "$HAS_SSHPASS" = true ]; then
        echo "  sshpass: installed"
    else
        echo "  sshpass: NOT installed (will prompt for password)"
    fi
}

# Main
case "${1:-all}" in
    build)
        cmd_build
        ;;
    deploy)
        cmd_deploy
        ;;
    install)
        cmd_install
        ;;
    restart)
        cmd_restart
        ;;
    stop)
        cmd_stop
        ;;
    test)
        cmd_test
        ;;
    demo)
        cmd_demo
        ;;
    status)
        cmd_status
        ;;
    clear)
        cmd_clear
        ;;
    backlight)
        cmd_backlight "$@"
        ;;
    rawtest)
        cmd_rawtest
        ;;
    shell)
        cmd_shell
        ;;
    all)
        cmd_deploy
        echo ""
        cmd_test
        ;;
    help|--help|-h)
        usage
        ;;
    *)
        echo "Unknown command: $1"
        usage
        exit 1
        ;;
esac
