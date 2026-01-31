#!/bin/bash
set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

VERSION=${VERSION:-"dev"}
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS="-s -w -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}"

OUTPUT_DIR="bin"
mkdir -p "$OUTPUT_DIR"

echo -e "${YELLOW}Building WatchDog Agent v${VERSION}${NC}"
echo ""

# Platforms to build
PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
)

for PLATFORM in "${PLATFORMS[@]}"; do
    GOOS="${PLATFORM%/*}"
    GOARCH="${PLATFORM#*/}"
    
    OUTPUT_NAME="agent-${GOOS}-${GOARCH}"
    if [ "$GOOS" = "windows" ]; then
        OUTPUT_NAME="${OUTPUT_NAME}.exe"
    fi
    
    echo -e "  Building ${GREEN}${OUTPUT_NAME}${NC}..."
    
    CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" go build \
        -ldflags="$LDFLAGS" \
        -o "${OUTPUT_DIR}/${OUTPUT_NAME}" \
        ./cmd/agent
done

echo ""
echo -e "${GREEN}Build complete!${NC} Binaries are in ${OUTPUT_DIR}/"
ls -lh "$OUTPUT_DIR"/agent-*
