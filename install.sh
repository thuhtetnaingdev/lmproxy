#!/bin/sh
set -e

REPO="thuhtetnaingdev/lmproxy"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.lmproxy}"
BIN_DIR="${BIN_DIR:-/usr/local/bin}"
SERVICE="llmproxy"

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
  linux)   ;;
  darwin)  echo "macOS: systemd not available. Binary will be installed without service." ;;
  *)       echo "Unsupported OS: $OS"; exit 1 ;;
esac

echo "→ Installing llmproxy for $OS/$ARCH..."

# Fetch latest release.
LATEST=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": "\(.*\)".*/\1/')
if [ -z "$LATEST" ]; then
  echo "Could not determine latest release. Set VERSION env to pin a specific version."
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
echo "$LATEST" > "$INSTALL_DIR/VERSION"
rm "/tmp/$TARBALL"

# Install binaries.
sudo mkdir -p "$BIN_DIR"
sudo cp "$INSTALL_DIR/llmproxy" "$BIN_DIR/llmproxy"
sudo cp "$INSTALL_DIR/llmproxy-server" "$BIN_DIR/llmproxy-server"
sudo chmod +x "$BIN_DIR/llmproxy" "$BIN_DIR/llmproxy-server"

echo "→ llmproxy → $BIN_DIR/llmproxy"
echo "→ llmproxy-server → $BIN_DIR/llmproxy-server"

# systemd service (Linux only).
if [ "$OS" = "linux" ]; then
  mkdir -p "$HOME/.config/systemd/user"

  cat > "$HOME/.config/systemd/user/$SERVICE.service" <<EOF
[Unit]
Description=LLMProxy — DeepSeek usage tracker
After=network.target

[Service]
Type=simple
WorkingDirectory=$INSTALL_DIR
Environment=STATIC_DIR=$INSTALL_DIR/frontend/dist
ExecStart=$BIN_DIR/llmproxy-server
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
EOF

  systemctl --user daemon-reload
  systemctl --user enable "$SERVICE"
  systemctl --user start "$SERVICE"

  echo "→ systemd service installed and started."
fi

echo ""
echo "✓ llmproxy $LATEST installed."
echo ""
echo "  Commands:"
echo "    llmproxy status     # check service status"
echo "    llmproxy logs       # tail logs"
echo "    llmproxy stop       # stop the proxy"
echo "    llmproxy start      # start the proxy"
echo "    llmproxy update     # update to latest release"
echo "    llmproxy uninstall  # remove everything"
echo ""
echo "  Dashboard: http://localhost:8080"
echo "  Login:     admin / admin"
echo ""
echo "  Set the DeepSeek API key in Settings after login."
