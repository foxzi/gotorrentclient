#!/bin/bash

# Build script for gotorrentclient
# Creates binaries for Linux AMD64 and ARM64

set -e  # Exit on error

# Set version from git tag or default
VERSION=$(git describe --tags 2>/dev/null || echo "v0.1.0")
BUILD_DIR="./build"
RELEASE_DIR="./release"

# Clean previous builds
rm -rf "$BUILD_DIR" "$RELEASE_DIR"
mkdir -p "$BUILD_DIR" "$RELEASE_DIR"

# Build function
build_for_platform() {
    local GOOS=$1
    local GOARCH=$2
    local OUTPUT_NAME="gotorrentclient-${VERSION}-${GOOS}-${GOARCH}"
    local BINARY_NAME="gotorrentclient"
    
    if [[ "$GOOS" == "windows" ]]; then
        BINARY_NAME="gotorrentclient.exe"
    fi
    
    echo "Building for $GOOS/$GOARCH..."
    
    # Set environment variables for cross-compilation
    export GOOS=$GOOS
    export GOARCH=$GOARCH
    export CGO_ENABLED=0  # Disable CGO for static binaries
    
    # Build the binary
    go build -ldflags="-s -w -X main.version=${VERSION}" -o "$BUILD_DIR/$BINARY_NAME" .
    
    # Create release package
    mkdir -p "$BUILD_DIR/package"
    cp "$BUILD_DIR/$BINARY_NAME" "$BUILD_DIR/package/"
    cp README.md "$BUILD_DIR/package/" 2>/dev/null || echo "No README.md found"
    cp LICENSE "$BUILD_DIR/package/" 2>/dev/null || echo "No LICENSE found"
    
    # Create archive
    if [[ "$GOOS" == "windows" ]]; then
        (cd "$BUILD_DIR" && zip -r "../$RELEASE_DIR/${OUTPUT_NAME}.zip" "package")
    else
        (cd "$BUILD_DIR" && tar -czf "../$RELEASE_DIR/${OUTPUT_NAME}.tar.gz" "package")
    fi
    
    # Cleanup
    rm -rf "$BUILD_DIR/package" "$BUILD_DIR/$BINARY_NAME"
    
    echo "✅ Built $OUTPUT_NAME"
}

# Build for each target platform
build_for_platform "linux" "amd64"
build_for_platform "linux" "arm64"

echo "Build complete. Binaries are available in the $RELEASE_DIR directory."