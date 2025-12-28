# EZIO-G500 Scripts

Scripts for building, deploying, and testing the eziolcd utility.

## Setup

Copy `.env.example` to `.env` and configure:

```bash
cp .env.example .env
# Edit .env with your pfSense IP and password
```

## Scripts

| Script | Description |
|--------|-------------|
| `deploy.sh` | Build and deploy to pfSense device |
| `build.sh` | Build locally or for FreeBSD |
| `install-pfsense.sh` | Install service on pfSense (run on device) |
| `find-lcd.sh` | Auto-detect LCD serial port |
| `test-serial.sh` | Test serial port communication |

## Deployment

```bash
# Build and deploy to pfSense
./deploy.sh deploy

# Just build locally
./deploy.sh build

# Build for FreeBSD only
./build.sh freebsd
```

## pfSense Installation

After deploying, SSH to pfSense and run:

```bash
# Quick install (downloads if needed, creates service, auto-starts)
./install-pfsense.sh

# The service will start automatically and run at boot
service eziolcd status
```
