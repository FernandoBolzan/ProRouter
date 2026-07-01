#!/usr/bin/env bash
set -euo pipefail

PROROUTER_INSTALL_DIR="${PROROUTER_INSTALL_DIR:-/usr/local/bin}"

echo "ProRouter Installer"
echo "==================="

install_from_source() {
  if command -v go &>/dev/null; then
    echo "Falling back to 'go install' (requires Go)..."
    go install github.com/FernandoBolzan/ProRouter/gateway-go/cmd/prorouter@latest
    echo ""
    echo "ProRouter installed via 'go install'!"
    echo "Make sure \$GOPATH/bin (or \$HOME/go/bin) is in your PATH."
    echo "Run 'prorouter init' to get started."
    exit 0
  fi
  echo ""
  echo "Could not install ProRouter automatically."
  echo ""
  echo "Option 1: Install Go from https://go.dev/dl/, then run:"
  echo "  go install github.com/FernandoBolzan/ProRouter/gateway-go/cmd/prorouter@latest"
  echo ""
  echo "Option 2: Download a prebuilt binary from:"
  echo "  https://github.com/FernandoBolzan/ProRouter/releases"
  echo ""
  exit 1
}

REPO="FernandoBolzan/ProRouter"
VERSION="${PROROUTER_VERSION:-latest}"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$OS" in
  linux)   OS="linux" ;;
  darwin)  OS="darwin" ;;
  *)       echo "Unsupported OS: $OS"; install_from_source ;;
esac

case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)          echo "Unsupported arch: $ARCH"; install_from_source ;;
esac

if [ "$VERSION" = "latest" ]; then
  VERSION=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" 2>/dev/null | grep '"tag_name"' | cut -d'"' -f4 | sed 's/^v//') || true
  if [ -z "$VERSION" ]; then
    echo "No GitHub release found. Trying 'go install'..."
    install_from_source
  fi
fi

FILENAME="prorouter_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/$REPO/releases/download/v${VERSION}/${FILENAME}"

echo "Downloading ProRouter v${VERSION} (${OS}/${ARCH})..."
if ! curl -fsSL "$URL" -o "/tmp/$FILENAME" 2>/dev/null; then
  echo "Binary download failed. Trying 'go install'..."
  install_from_source
fi

echo "Extracting..."
tar -xzf "/tmp/$FILENAME" -C "/tmp/"

echo "Installing to $PROROUTER_INSTALL_DIR..."
if [ -w "$PROROUTER_INSTALL_DIR" ]; then
  mv "/tmp/prorouter" "$PROROUTER_INSTALL_DIR/prorouter"
else
  sudo mv "/tmp/prorouter" "$PROROUTER_INSTALL_DIR/prorouter"
fi
chmod +x "$PROROUTER_INSTALL_DIR/prorouter"
rm -f "/tmp/$FILENAME"

echo ""
echo "ProRouter v${VERSION} installed successfully!"
echo "Run 'prorouter init' to get started."
