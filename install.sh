#!/bin/sh
set -e

REPO="thuhtetnaingdev/lmproxy"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.lmproxy}"
BIN_DIR="${BIN_DIR:-/usr/local/bin}"

# Detect OS and arch.
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  arm64)   ARCH="arm64" ;;
  *)       echo "Unsupported arch: $ARCH"; exit 1 ;;
esac

case "$OS" in
  linux|darwin) ;;
  *)           echo "Unsupported OS: $OS"; exit 1 ;;
esac

echo "→ Installing llmproxy for $OS/$ARCH..."

# Fetch latest release tag.
LATEST=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": "\(.*\)".*/\1/')
if [ -z "$LATEST" ]; then
  echo "Could not determine latest release. Set VERSION env to pin a specific tag."
  exit 1
fi
echo "→ Latest release: $LATEST"

TARBALL="llmproxy-${OS}-${ARCH}.tar.gz"
URL="https://github.com/$REPO/releases/download/${LATEST}/${TARBALL}"

echo "→ Downloading $URL..."
curl -sL "$URL" -o "/tmp/$TARBALL"

echo "→ Extracting to $INSTALL_DIR..."
rm -rf "$INSTALL_DIR"
mkdir -p "$INSTALL_DIR"
tar xzf "/tmp/$TARBALL" -C "$INSTALL_DIR"
rm "/tmp/$TARBALL"

# Symlink binary.
sudo mkdir -p "$BIN_DIR"
sudo ln -sf "$INSTALL_DIR/llmproxy" "$BIN_DIR/llmproxy"
echo "→ llmproxy → $BIN_DIR/llmproxy"

echo ""
echo "✓ llmproxy $LATEST installed."
echo ""
echo "  Set the DeepSeek API key via the dashboard, or export it:"
echo "    export DEEPSEEK_API_KEY=sk-..."
echo ""
echo "  Start the proxy:"
echo "    export ADMIN_PASSWORD=your-password"
echo "    export JWT_SECRET=your-secret   # optional, auto-generated if unset"
echo "    cd $INSTALL_DIR && STATIC_DIR=frontend/dist llmproxy"
echo ""
echo "  Then open http://localhost:8080"
