#!/usr/bin/env bash
# Build script for ezio-g500
# Builds the CLI tool for multiple platforms

set -e

VERSION=${VERSION:-"dev"}
BUILD_DIR="dist"
BINARY_NAME="eziolcd"

# Ensure we're in the project root
cd "$(dirname "$0")/.."

echo "Building eziolcd v${VERSION}"
echo "================================"

# Clean previous builds
rm -rf "${BUILD_DIR}"
mkdir -p "${BUILD_DIR}"

# Build targets
TARGETS=(
    "freebsd/amd64"
    "freebsd/arm64"
    "linux/amd64"
    "linux/arm64"
    "linux/arm"
    "darwin/amd64"
    "darwin/arm64"
)

for target in "${TARGETS[@]}"; do
    IFS='/' read -r GOOS GOARCH <<< "$target"
    output="${BUILD_DIR}/${BINARY_NAME}-${GOOS}-${GOARCH}"
    
    echo "Building for ${GOOS}/${GOARCH}..."
    
    GOOS=$GOOS GOARCH=$GOARCH CGO_ENABLED=0 go build \
        -ldflags="-s -w -X main.version=${VERSION}" \
        -o "$output" \
        ./cmd/eziolcd
    
    echo "  -> ${output}"
done

# Create checksums
echo ""
echo "Creating checksums..."
cd "${BUILD_DIR}"
sha256sum ${BINARY_NAME}-* > checksums.txt
cat checksums.txt

echo ""
echo "Build complete! Binaries in ${BUILD_DIR}/"
