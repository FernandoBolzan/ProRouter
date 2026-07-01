#!/usr/bin/env bash
set -euo pipefail

echo "ProRouter Installer"
echo "==================="

REPO="prorouter/prorouter"
INSTALL_DIR="${PROROUTER_INSTALL_DIR:-/usr/local/bin}"
VERSION="${PROROUTER_VERSION:-latest}"

# Detect platform
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$OS" in
  linux)   OS="linux" ;;
  darwin)  OS="darwin" ;;
  *)       echo "Unsupported OS: $OS"; exit 1 ;;
esac

case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)          echo "Unsupported arch: $ARCH"; exit 1 ;;
esac

# Get latest version
if [ "$VERSION" = "latest" ]; then
  VERSION=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | cut -d'"' -f4)
  VERSION="${VERSION#v}"
fi

FILENAME="prorouter_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/$REPO/releases/download/v${VERSION}/${FILENAME}"

echo "Downloading ProRouter v${VERSION} (${OS}/${ARCH})..."
curl -fsSL "$URL" -o "/tmp/$FILENAME"

echo "Extracting..."
tar -xzf "/tmp/$FILENAME" -C "/tmp/"

echo "Installing to $INSTALL_DIR..."
if [ -w "$INSTALL_DIR" ]; then
  mv "/tmp/prorouter" "$INSTALL_DIR/prorouter"
else
  sudo mv "/tmp/prorouter" "$INSTALL_DIR/prorouter"
fi
chmod +x "$INSTALL_DIR/prorouter"
rm -f "/tmp/$FILENAME"

echo ""
echo "ProRouter v${VERSION} installed successfully!"
echo "Run 'prorouter init' to get started."
