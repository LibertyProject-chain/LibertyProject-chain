#!/bin/bash

set -e
GETH_URL="https://github.com/LibertyProject-chain/LibertyProject-chain/releases/download/v0.6.7/geth"
GENESIS_URL="https://github.com/LibertyProject-chain/LibertyProject-chain/releases/download/v0.6.7/genesis.json"
BOOTNODE="enode://dc6a8df6a883eee5721aee83d3e89be69000b69b532954641a9e6c6fa1d3bedade9b2b481514a2d75738903c0fddb58ae839d16dbcc7b93d98b1b461ff9e4463@46.4.201.254:40404"
RPC_PORT="9945"
P2P_PORT="40404"
NETWORK_ID="16384"

if [ "$EUID" -ne 0 ]; then
  echo "❌ Please run this script as root (e.g., sudo ./deploy_liberty_node.sh)."
  exit 1
fi

ORIG_USER="${SUDO_USER:-$USER}"

if [ "$ORIG_USER" = "root" ]; then
  NODE_DIR="/root/go-liberty"
else
  NODE_DIR="/home/$ORIG_USER/go-liberty"
fi

BIN_DIR="$NODE_DIR/bin"
DATA_DIR="$NODE_DIR/data"
NODE_KEY="$DATA_DIR/geth/nodekey"
SERVICE_FILE="/etc/systemd/system/liberty-node.service"

read -p "Enter your coinbase address: " COINBASE
read -p "Enter the number of mining threads: " MINER_THREADS

echo "✅ Updating system and installing packages..."
apt update
apt install -y curl ufw jq openssl file

echo "✅ Creating directories for $ORIG_USER..."
mkdir -p "$BIN_DIR"
mkdir -p "$DATA_DIR"
chown -R "$ORIG_USER":"$ORIG_USER" "$NODE_DIR"

TEMP_DIR=$(mktemp -d)
echo "✅ Temporary directory: $TEMP_DIR"

echo "✅ Downloading geth and genesis.json..."
curl -L "$GETH_URL" -o "$TEMP_DIR/geth"
curl -L "$GENESIS_URL" -o "$TEMP_DIR/genesis.json"

echo "✅ Checking if geth is a valid ELF binary..."
if file "$TEMP_DIR/geth" | grep -q ELF; then
  echo "✅ Geth binary verified."
else
  echo "❌ The downloaded geth file is not an ELF binary. Check the URL!"
  exit 1
fi

echo "✅ Moving geth and genesis.json to $NODE_DIR..."
mv "$TEMP_DIR/geth" "$BIN_DIR/geth"
chmod +x "$BIN_DIR/geth"

mv "$TEMP_DIR/genesis.json" "$DATA_DIR/genesis.json"


if [ ! -f "$NODE_KEY" ]; then
  echo "✅ Generating nodekey..."
  mkdir -p "$(dirname "$NODE_KEY")"
  openssl rand -hex 32 > "$NODE_KEY"
  chown "$ORIG_USER":"$ORIG_USER" "$NODE_KEY"
else
  echo "⚠️ Nodekey already exists, skipping generation."
fi

echo "✅ Ensuring $DATA_DIR is owned by $ORIG_USER..."
chown -R "$ORIG_USER":"$ORIG_USER" "$DATA_DIR"

if [ ! -d "$DATA_DIR/geth/chaindata" ]; then
  echo "✅ Initializing genesis block..."
  sudo -u "$ORIG_USER" "$BIN_DIR/geth" --datadir "$DATA_DIR" init "$DATA_DIR/genesis.json"
else
  echo "⚠️ Blockchain data already exists, skipping init."
fi

echo "✅ Creating systemd service: $SERVICE_FILE"
cat <<EOF > "$SERVICE_FILE"
[Unit]
Description=Liberty Project Node Service
After=network.target

[Service]
User=$ORIG_USER
WorkingDirectory=$DATA_DIR
ExecStart=$BIN_DIR/geth --datadir $DATA_DIR \\
    --networkid $NETWORK_ID \\
    --nodekey $NODE_KEY \\
    --mine \\
    --miner.threads=$MINER_THREADS \\
    --miner.etherbase=$COINBASE \\
    --port $P2P_PORT \\
    --http \\
    --http.api web3,eth,txpool,net \\
    --http.addr 0.0.0.0 \\
    --http.port $RPC_PORT \\
    --http.corsdomain "*" \\
    --gcmode archive \\
    --syncmode full \\
    --bootnodes $BOOTNODE
Restart=always
RestartSec=10
LimitNOFILE=4096

[Install]
WantedBy=multi-user.target
EOF

echo "✅ Configuring UFW firewall rules..."
ufw allow "$RPC_PORT"/tcp
ufw allow "$P2P_PORT"/tcp
ufw allow "$P2P_PORT"/udp
ufw reload

echo "✅ Starting node service..."
systemctl daemon-reload
systemctl enable liberty-node
systemctl restart liberty-node

echo "✅ Node Enode:"
sudo -u "$ORIG_USER" "$BIN_DIR/geth" attach "$DATA_DIR/geth.ipc" --exec 'admin.nodeInfo.enode' || {
  echo "⚠️ Could not retrieve enode info. Possibly the node is still starting."
}

echo "✅ Viewing live logs..."
journalctl -u liberty-node -f -o cat
