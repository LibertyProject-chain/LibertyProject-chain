# Base image
FROM ubuntu:22.04

# Set environment variables
ENV DATA_DIR="/liberty/data"
ENV BIN_DIR="/liberty/bin"
ENV NETWORK_ID="16384"
ENV P2P_PORT="40404"
ENV RPC_PORT="9945"
ENV MINER_THREADS="2"
ENV MINER_ADDRESS="0xb88EE4b816b3162B9768707aBc5ca19c3B8A5Ccc"
ENV BOOTNODES="enode://dc6a8df6a883eee5721aee83d3e89be69000b69b532954641a9e6c6fa1d3bedade9b2b481514a2d75738903c0fddb58ae839d16dbcc7b93d98b1b461ff9e4463@46.4.201.254:40404"

# Install required packages
RUN apt-get update && \
    apt-get install -y curl openssl ca-certificates && \
    rm -rf /var/lib/apt/lists/*

# Create working directories
RUN mkdir -p "$BIN_DIR" "$DATA_DIR"

# Download Geth and Genesis.json directly
RUN curl -L -o "$BIN_DIR/geth" "https://github.com/LibertyProject-chain/LibertyProject-chain/releases/download/v0.6.7/geth" && \
    curl -L -o "$DATA_DIR/genesis.json" "https://github.com/LibertyProject-chain/LibertyProject-chain/releases/download/v0.6.7/genesis.json" && \
    chmod +x "$BIN_DIR/geth"

# Initialize the genesis block
RUN "$BIN_DIR/geth" --datadir "$DATA_DIR" init "$DATA_DIR/genesis.json"

# Expose required ports
EXPOSE 40404/tcp
EXPOSE 40404/udp
EXPOSE 9945/tcp

# Start node on container launch
CMD ["sh", "-c", "/liberty/bin/geth \
    --datadir /liberty/data \
    --networkid $NETWORK_ID \
    --port $P2P_PORT \
    --discovery.port $P2P_PORT \
    --http --http.addr 0.0.0.0 --http.port $RPC_PORT --http.api web3,eth,txpool,net --http.corsdomain '*' \
    --gcmode archive \
    --syncmode full \
    --bootnodes $BOOTNODES \
    --mine \
    --miner.threads=$MINER_THREADS \
    --miner.etherbase=$MINER_ADDRESS"]
