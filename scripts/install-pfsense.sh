#!/bin/sh
# pfSense/FreeBSD installation script for eziolcd
# Downloads, installs, and configures the LCD status daemon service

set -e

INSTALL_DIR="/usr/local/bin"
BINARY_NAME="eziolcd"
SERVICE_NAME="eziolcd"
RC_SCRIPT="/usr/local/etc/rc.d/${SERVICE_NAME}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "╔════════════════════════════════════════════╗"
echo "║     EZIO-G500 LCD Daemon Installer         ║"
echo "║     for pfSense / FreeBSD                  ║"
echo "╚════════════════════════════════════════════╝"
echo ""

# Check if running as root
if [ "$(id -u)" -ne 0 ]; then
    echo "${RED}Error: This script must be run as root${NC}"
    exit 1
fi

# Detect architecture
ARCH=$(uname -m)
case $ARCH in
    amd64|x86_64)
        ARCH="amd64"
        ;;
    arm64|aarch64)
        ARCH="arm64"
        ;;
    *)
        echo "${RED}Error: Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

# Detect serial port
SERIAL_PORT="/dev/cuau1"
if [ -c "/dev/cuau1" ]; then
    SERIAL_PORT="/dev/cuau1"
elif [ -c "/dev/ttyS1" ]; then
    SERIAL_PORT="/dev/ttyS1"
elif [ -c "/dev/cuau0" ]; then
    SERIAL_PORT="/dev/cuau0"
fi

echo "Detected: FreeBSD/${ARCH}"
echo "Serial port: ${SERIAL_PORT}"
echo ""

# Check if binary exists or download it
if [ -f "${INSTALL_DIR}/${BINARY_NAME}" ]; then
    echo "Binary already installed at ${INSTALL_DIR}/${BINARY_NAME}"
elif [ -f "./${BINARY_NAME}-freebsd-${ARCH}" ]; then
    echo "Installing binary from current directory..."
    cp "./${BINARY_NAME}-freebsd-${ARCH}" "${INSTALL_DIR}/${BINARY_NAME}"
    chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
elif [ -f "./${BINARY_NAME}" ]; then
    echo "Installing binary from current directory..."
    cp "./${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
    chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
else
    echo "Downloading eziolcd binary..."
    if command -v fetch >/dev/null 2>&1; then
        fetch -o "${INSTALL_DIR}/${BINARY_NAME}" \
            "https://github.com/sagostin/ezio-g500/releases/latest/download/eziolcd-freebsd-${ARCH}" || {
            echo "${RED}Download failed. Please download manually.${NC}"
            exit 1
        }
    elif command -v curl >/dev/null 2>&1; then
        curl -L -o "${INSTALL_DIR}/${BINARY_NAME}" \
            "https://github.com/sagostin/ezio-g500/releases/latest/download/eziolcd-freebsd-${ARCH}" || {
            echo "${RED}Download failed. Please download manually.${NC}"
            exit 1
        }
    else
        echo "${RED}Error: Neither fetch nor curl available${NC}"
        exit 1
    fi
    chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
fi

echo "${GREEN}✓ Binary installed: ${INSTALL_DIR}/${BINARY_NAME}${NC}"

# Stop existing service if running
if [ -f "/var/run/${SERVICE_NAME}.pid" ]; then
    echo "Stopping existing service..."
    service ${SERVICE_NAME} stop 2>/dev/null || true
fi

# Create rc.d script for service management
echo "Creating service script..."
cat > "${RC_SCRIPT}" << 'RCEOF'
#!/bin/sh

# PROVIDE: eziolcd
# REQUIRE: DAEMON NETWORKING
# KEYWORD: shutdown

. /etc/rc.subr

name="eziolcd"
rcvar="${name}_enable"
desc="EZIO-G500 LCD status daemon"

# Set defaults
load_rc_config $name
: ${eziolcd_enable:="NO"}
: ${eziolcd_port:="/dev/cuau1"}
: ${eziolcd_user:="root"}

pidfile="/var/run/${name}.pid"
command="/usr/local/bin/eziolcd"
command_args="-port ${eziolcd_port} daemon"

start_cmd="${name}_start"
stop_cmd="${name}_stop"
status_cmd="${name}_status"
restart_cmd="${name}_restart"

eziolcd_start()
{
    if [ -f "${pidfile}" ] && kill -0 $(cat "${pidfile}") 2>/dev/null; then
        echo "${name} is already running"
        return 1
    fi
    echo "Starting ${name}..."
    /usr/sbin/daemon -f -p "${pidfile}" -u "${eziolcd_user}" ${command} ${command_args}
    sleep 1
    if [ -f "${pidfile}" ] && kill -0 $(cat "${pidfile}") 2>/dev/null; then
        echo "${name} started (pid: $(cat ${pidfile}))"
    else
        echo "Failed to start ${name}"
        return 1
    fi
}

eziolcd_stop()
{
    if [ -f "${pidfile}" ]; then
        echo "Stopping ${name}..."
        kill $(cat "${pidfile}") 2>/dev/null || true
        rm -f "${pidfile}"
        # Also kill any orphaned processes
        pkill -f "eziolcd.*daemon" 2>/dev/null || true
        echo "${name} stopped"
    else
        echo "${name} is not running"
    fi
}

eziolcd_status()
{
    if [ -f "${pidfile}" ] && kill -0 $(cat "${pidfile}") 2>/dev/null; then
        echo "${name} is running (pid: $(cat ${pidfile}))"
    else
        echo "${name} is not running"
        return 1
    fi
}

eziolcd_restart()
{
    eziolcd_stop
    sleep 1
    eziolcd_start
}

run_rc_command "$1"
RCEOF

chmod +x "${RC_SCRIPT}"
echo "${GREEN}✓ Service script created: ${RC_SCRIPT}${NC}"

# Enable service in rc.conf
if ! grep -q "eziolcd_enable" /etc/rc.conf 2>/dev/null; then
    echo "" >> /etc/rc.conf
    echo "# EZIO-G500 LCD Status Daemon" >> /etc/rc.conf
    echo "eziolcd_enable=\"YES\"" >> /etc/rc.conf
    echo "eziolcd_port=\"${SERIAL_PORT}\"" >> /etc/rc.conf
    echo "${GREEN}✓ Service enabled in /etc/rc.conf${NC}"
else
    echo "${YELLOW}! Service already configured in /etc/rc.conf${NC}"
fi

# Start the service
echo ""
echo "Starting eziolcd service..."
service ${SERVICE_NAME} start

echo ""
echo "╔════════════════════════════════════════════╗"
echo "║          Installation Complete!            ║"
echo "╚════════════════════════════════════════════╝"
echo ""
echo "Service commands:"
echo "  service eziolcd start    - Start the daemon"
echo "  service eziolcd stop     - Stop the daemon"
echo "  service eziolcd restart  - Restart the daemon"
echo "  service eziolcd status   - Check if running"
echo ""
echo "Manual commands:"
echo "  eziolcd -port ${SERIAL_PORT} status  - Show status once"
echo "  eziolcd -port ${SERIAL_PORT} demo    - Run demo"
echo "  eziolcd --help                       - Show all options"
echo ""
echo "Configuration:"
echo "  Edit /etc/rc.conf to change settings:"
echo "    eziolcd_port=\"${SERIAL_PORT}\""
echo ""
