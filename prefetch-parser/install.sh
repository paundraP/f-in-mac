#!/bin/sh
set -e

# install.sh — install pfparse to /usr/local/bin
#
# Usage:
#   ./install.sh

BINARY="pfparse"

# Detect OS and arch
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "unsupported architecture: $ARCH"; exit 1 ;;
esac

case "$OS" in
    darwin|linux) ;;
    *) echo "unsupported OS: $OS"; exit 1 ;;
esac

DEST="${DESTDIR:-/usr/local/bin}"

# If built binary exists locally, install that
if [ -f "bin/${BINARY}-${OS}-${ARCH}" ]; then
    echo "Installing local build: bin/${BINARY}-${OS}-${ARCH} → ${DEST}/${BINARY}"
    install -v "bin/${BINARY}-${OS}-${ARCH}" "${DEST}/${BINARY}"
elif command -v go >/dev/null 2>&1; then
    echo "Building and installing from source..."
    go install -ldflags="-s -w" ./cmd/pfparse/
    echo "Installed to $(go env GOPATH)/bin/${BINARY}"
else
    echo "No local build found for ${OS}/${ARCH} and Go is not installed."
    echo "Download a pre-built release from:"
    echo "  https://github.com/paundraP/f-in-mac/releases"
    exit 1
fi

echo "Done. Run: pfparse -h"
