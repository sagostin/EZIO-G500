# EZIO-G500 Go Library

A Go library for the EZIO-G500 graphics LCD display found in Checkpoint/pfSense appliances.

## Features

- **Status Daemon** - 7 rotating screens with live system metrics
- **3D Animated Logo** - Rotating "pf" letters with smooth animation
- **LED Health Indicators** - CPU/memory threshold alerts (green/orange/red)
- **Live Bandwidth Monitoring** - Per-interface KB/s rates
- **128x64 Graphics** - Full framebuffer with drawing primitives
- **Smooth Scrolling** - For long text and interface lists
- **pfSense Integration** - CPU, memory, interfaces, WireGuard tunnels

## Quick Start (pfSense)

```bash
# SSH to your pfSense device
ssh root@your-pfsense-ip

# Download and install the service
fetch -o install.sh https://raw.githubusercontent.com/sagostin/ezio-g500/main/scripts/install-pfsense.sh
chmod +x install.sh
./install.sh
```

The installer will:
- Download the binary
- Create an rc.d service
- Enable it in rc.conf
- Start the daemon automatically

## Status Screens

The daemon cycles through 7 screens every 5 seconds:

| Screen | Content |
|--------|---------|
| **Logo** | 3D rotating pf, hostname, uptime, CPU/MEM |
| **CPU** | Usage bar, load average, uptime |
| **Memory** | Usage bar, used/free MB |
| **Interfaces** | Active interfaces with IPs |
| **WAN Traffic** | Live WAN bandwidth (KB/s) |
| **Tunnel Traffic** | VPN/WireGuard bandwidth |
| **LAN Traffic** | Other interfaces bandwidth |

## LED Indicators

| LED | Meaning |
|-----|---------|
| LED1 (top) | 🟢 Logo screen, 🟠 Traffic screens |
| LED2 (middle) | 🟢 Healthy, 🟠 Warning (70-90%), 🔴 Critical (>90%) |
| LED3 (bottom) | 🟢 Home (logo screen) |

## Manual Usage

```bash
# Run status daemon
eziolcd -port /dev/cuau1 daemon

# Show single status
eziolcd -port /dev/cuau1 status

# Display text
eziolcd -port /dev/cuau1 text "Hello World"

# Run demo
eziolcd -port /dev/cuau1 demo

# Control LEDs
eziolcd -port /dev/cuau1 led 1 green
```

## Building from Source

```bash
git clone https://github.com/sagostin/ezio-g500.git
cd ezio-g500

# Build for pfSense
GOOS=freebsd GOARCH=amd64 go build -o eziolcd-freebsd-amd64 ./cmd/eziolcd

# Deploy to device
cd scripts && ./deploy.sh install
```

## Serial Ports

| Platform | Port |
|----------|------|
| pfSense/FreeBSD | `/dev/cuau1` |
| Linux | `/dev/ttyS1` |
| USB Serial | `/dev/ttyUSB0` |

## Package Structure

```
pkg/
├── eziog500/     # Core driver: device, framebuffer, LEDs
├── display/      # High-level display API
├── font/         # 8px and 6px pixel fonts
├── pfsense/      # Metrics and status screens
├── menu/         # Interactive menu system
├── render3d/     # 3D wireframe rendering
└── ui/           # UI widgets
```

## Credits

Protocol reverse engineering by:
- [Saint-Frater](https://git.nox-rhea.org/globals/reverse-engineering/ezio-g500)
- [tchatzi](https://github.com/tchatzi/EZIO-G500)

## License

MIT License
