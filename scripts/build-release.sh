#!/bin/bash

set -e

VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "v2.0.0")
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

BUILD_DIR="build"
DIST_DIR="dist"

mkdir -p "$BUILD_DIR" "$DIST_DIR"

LD_FLAGS="-s -w -X main.version=$VERSION -X main.buildDate=$BUILD_DATE -X main.commit=$COMMIT"

build_target() {
    local os=$1
    local arch=$2
    local ext=""
    if [ "$os" = "windows" ]; then
        ext=".exe"
    fi
    
    local target_dir="$BUILD_DIR/$os-$arch"
    mkdir -p "$target_dir"
    
    echo "Building $os/$arch..."
    GOOS=$os GOARCH=$arch go build -ldflags="$LD_FLAGS" -o "$target_dir/clipsync$ext" ./cmd/clipsync
    
    cp certs/server.crt "$target_dir/" 2>/dev/null || true
    cp certs/server.key "$target_dir/" 2>/dev/null || true
    
    local tarball="$DIST_DIR/clipsync-$VERSION-$os-$arch.tar.gz"
    cd "$BUILD_DIR" && tar -czvf "../$tarball" "$os-$arch" && cd ..
    
    echo "Built: $tarball"
}

echo "=== ClipSync Release Builder ==="
echo "Version: $VERSION"
echo "Commit: $COMMIT"
echo "Build Date: $BUILD_DATE"
echo ""

build_target "linux" "amd64"
build_target "windows" "amd64"

echo ""
echo "=== Build Complete ==="
ls -la "$DIST_DIR/"