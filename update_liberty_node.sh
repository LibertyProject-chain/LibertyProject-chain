#!/bin/bash

set -e

# URL of the new geth version
NEW_GETH_URL="https://github.com/LibertyProject-chain/LibertyProject-chain/releases/download/v0.6.8/geth"

# Check if the script is running with root privileges
if [ "$EUID" -ne 0 ]; then
  echo "❌ Please run this script as root (e.g., sudo ./update_liberty_node.sh)."
  exit 1
fi

# Determine the original user who invoked sudo (or the current user if run as root)
ORIG_USER="${SUDO_USER:-$USER}"

# Define directories, similar to the deployment script
if [ "$ORIG_USER" = "root" ]; then
  NODE_DIR="/root/go-liberty"
else
  NODE_DIR="/home/$ORIG_USER/go-liberty"
fi

BIN_DIR="$NODE_DIR/bin"
OLD_GETH="$BIN_DIR/geth"

# Systemd service name and file
SERVICE_NAME="liberty-node.service"
SERVICE_FILE="/etc/systemd/system/$SERVICE_NAME"

echo "✅ Stopping the liberty-node service..."
systemctl stop "$SERVICE_NAME"

TEMP_DIR=$(mktemp -d)
echo "✅ Temporary directory for download: $TEMP_DIR"

echo "✅ Downloading the new geth version..."
curl -L "$NEW_GETH_URL" -o "$TEMP_DIR/geth"

echo "✅ Checking if the downloaded file is an ELF binary..."
if file "$TEMP_DIR/geth" | grep -q ELF; then
  echo "✅ The file is an ELF binary."
else
  echo "❌ The geth file is not a valid ELF binary. Check the URL or file content."
  exit 1
fi

echo "✅ Replacing the old geth binary..."
mv "$TEMP_DIR/geth" "$OLD_GETH"
chmod +x "$OLD_GETH"
chown "$ORIG_USER":"$ORIG_USER" "$OLD_GETH"

rm -rf "$TEMP_DIR"

echo "✅ Starting the liberty-node service..."
systemctl start "$SERVICE_NAME"

# Optional: check the service status
sleep 3
systemctl status "$SERVICE_NAME" --no-pager

echo "✅ Update completed! To view live logs, run:"
echo "   journalctl -u $SERVICE_NAME -f -o cat"
