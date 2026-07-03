#!/bin/bash
# Build .deb and .rpm packages for gotorrentclient.
#
# Builds static Linux binaries for amd64 and arm64, then runs nfpm
# (in a Docker container, per project policy) to produce the packages.
#
# Output: ./release/*.deb and ./release/*.rpm

set -e

# Resolve repository root (this script lives in packaging/).
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT_DIR"

BUILD_DIR="./build"
RELEASE_DIR="./release"
NFPM_IMAGE="goreleaser/nfpm:latest"

# Version: git tag without the leading 'v'; hyphens replaced with '.'
# so it is valid for both deb and rpm.
RAW_VERSION=$(git describe --tags 2>/dev/null || echo "v0.1.0")
VERSION=$(echo "${RAW_VERSION#v}" | tr '-' '.')

mkdir -p "$BUILD_DIR" "$RELEASE_DIR"

build_binary() {
    local arch=$1
    echo "Building binary for linux/$arch..."
    # nfpm reads the binary from a fixed literal path (build/gotorrentclient),
    # so build the arch-specific binary directly there.
    GOOS=linux GOARCH="$arch" CGO_ENABLED=0 \
        go build -ldflags="-s -w -X main.version=${RAW_VERSION}" \
        -o "$BUILD_DIR/gotorrentclient" .
}

package_arch() {
    local arch=$1
    echo "Packaging deb and rpm for $arch (version $VERSION)..."
    for packager in deb rpm; do
        docker run --rm \
            -v "$ROOT_DIR:/work" -w /work \
            -e VERSION="$VERSION" \
            -e ARCH="$arch" \
            "$NFPM_IMAGE" package \
            --config packaging/nfpm.yaml \
            --target "$RELEASE_DIR/" \
            --packager "$packager"
    done
}

for arch in amd64 arm64; do
    build_binary "$arch"
    package_arch "$arch"
done

echo ""
echo "Done. Packages in $RELEASE_DIR:"
ls -1 "$RELEASE_DIR"/*.deb "$RELEASE_DIR"/*.rpm 2>/dev/null
