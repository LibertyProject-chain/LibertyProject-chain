
# Liberty-Project-chain

Based on the `go-ethereum` v1.11.6 source code with custom modifications to the mining algorithm for research and testing purposes.

## Custom Mining Algorithm

Our code has been modified to optimize the mining process and test dynamic block reward distribution. The mining algorithm uses a combined `Blake3/Keccak256` hashing function, and the block reward automatically decreases with time

## Mining Process Overview

```
+-------------------------------------------------------------+
|                        Mining Process                       |
+-------------------------------------------------------------+
| 1. Retrieve Block Header                                    |
|    - Extract header data from the block                     |
+-------------------------------------------------------------+
| 2. Calculate Target                                         |
|    - target = 2^256 / header.Difficulty                     |
+-------------------------------------------------------------+
| 3. Initialize Variables                                     |
|    - nonce = seed, powBuffer = big.Int, iterCount = 3000    |
+-------------------------------------------------------------+
| 4. Start Nonce Search                                       |
|    - Loop until a valid nonce is found or signal received   |
+-------------------------------------------------------------+
| 5. Generate SealHash                                        |
|    - Exclude Nonce/MixDigest, Keccak256 remaining fields    |
+-------------------------------------------------------------+
| 6. Concatenate SealHash and Nonce                           |
+-------------------------------------------------------------+
| 7. Perform Multiple Hashing                                 |
|    - hashResult = Blake3(sealHash + nonce)                  |
|    - Loop iterCount: hashResult = Blake3(hashResult)        |
+-------------------------------------------------------------+
| 8. Convert and Compare Result                               |
|    - Convert hashResult to big.Int (powBuffer)              |
|    - Compare powBuffer with target                          |
+-------------------------------------------------------------+
| 9. Check Result                                             |
|    - If powBuffer <= target: Found valid nonce             |
|    - Otherwise: Increment nonce and repeat                  |
+-------------------------------------------------------------+
| 10. Complete Block                                          |
|    - Set header.Nonce and header.MixDigest                  |
|    - Log and submit the block                               |
+-------------------------------------------------------------+
```
More info in:
- [Discord Liberty Project](https://discord.gg/vVmzgB5t)
- [Explorer Lyberty Project](https://explorer.libertyproject.space)
- [Website Lyberty Project](https://libertyproject.space)


## How to Deploy the Node and Start Mining

### Prerequisites
Operating System: Linux (Ubuntu recommended)
User Permissions: Root or sudo privileges
Ethereum Wallet: MetaMask or any EVM-compatible wallet

### 1. Wallet Configuration
We recommend using MetaMask or any other EVM-compatible wallet.

### Network Settings
- **Network Name**: Liberty Project
- **RPC URL**: https://rpc.libertyproject.space
- **Chain ID**: 16384
- **Currency Symbol**: LBRT
- **Block Explorer URL**: https://explorer.libertyproject.space
  
Simply input these settings in your wallet to connect seamlessly with the Liberty network.



## Liberty Node Deployment
This repository provides three ways to deploy a Liberty node:
### Quick Setup via Script (recommended for most users)
### Quick Setup with Docker (for container-based deployments)
### Manual Configuration (for advanced users)

After deploying your node, instructions are included for stopping and removing the service.

### 1. Quick Setup via Script
#### 1.1. Install Dependencies
Make sure your system is up to date and that you have required packages:

```bash
sudo apt update && sudo apt install -y curl ufw jq openssl file
```

#### 1.2. Download the Deployment Script
Download deploy_liberty_node.sh from this repository:

```bash
curl -L https://raw.githubusercontent.com/LibertyProject-chain/LibertyProject-chain/main/deploy_liberty_node.sh -o deploy_liberty_node.sh
```
#### 1.3. Make It Executable
```bash
chmod +x deploy_liberty_node.sh
```

#### 1.4. Run the Script
```bash
sudo ./deploy_liberty_node.sh
```

You will be prompted for:
Coinbase address (where mining rewards go)
Number of mining threads
The script will automatically:
Download the latest Geth binary for Liberty.
Create systemd service files.
Start the node under your local user (if using sudo).
Display logs in real-time.
When completed, your node is running as a systemd service named liberty-node.

### 2. Quick Setup with Docker
Below is a simple Docker-based approach:

#### 2.1. Install Docker
```bash
sudo apt update && sudo apt install -y docker.io
```

#### 2.2. Download the Dockerfile
```bash
curl -L https://raw.githubusercontent.com/LibertyProject-chain/LibertyProject-chain/main/Dockerfile -o Dockerfile
```

#### 2.3. Build the Docker Image
```bash
sudo docker build -t liberty-node .
```

#### 2.4. Run the Docker Container
Use a single command line to run:

```bash
docker run -d --name liberty-node \
    -p 40404:40404/tcp \
    -p 40404:40404/udp \
    -p 9945:9945/tcp \
    -e MINER_ADDRESS=0xYourCoinbaseAddress \
    -e MINER_THREADS=4 \
    liberty-node
```
set your address for rewards and the number of cpu threads that will be allocated for mining

Display logs in real-time
```bash
docker logs -f liberty-node
```

### 3. Manual Configuration (Advanced Users)
For advanced or customized setups, you can configure everything manually:

#### 3.1. Create a User
```bash
sudo useradd -m -s /bin/bash liberty
```

#### 3.2. Create Directories for Data & Software
```bash
sudo mkdir -p /home/liberty/liberty
sudo chown liberty:liberty /home/liberty/liberty
```

#### 3.3. Download Node Software
```bash
sudo curl -L https://github.com/LibertyProject-chain/LibertyProject-chain/releases/download/v0.6.7/geth -o /usr/local/bin/geth
sudo chmod +x /usr/local/bin/geth
```

#### 3.4. Download and Initialize the Genesis File
```bash
sudo curl -L https://github.com/LibertyProject-chain/LibertyProject-chain/releases/download/v0.6.7/genesis.json -o /home/liberty/liberty/genesis.json
sudo chown liberty:liberty /home/liberty/liberty/genesis.json
sudo -u liberty geth --datadir /home/liberty/liberty init /home/liberty/liberty/genesis.json
```

#### 3.5. Create a Systemd Service
Create /etc/systemd/system/liberty-node.service:

```bash
[Unit]
Description=Liberty Node
After=network.target

[Service]
User=liberty
ExecStart=/usr/local/bin/geth --datadir /home/liberty/liberty \
--networkid 16384 \
--port 40404 \
--mine \
--miner.threads=16 \
--miner.etherbase=<your_address> \
--gcmode=archive \
--http.api web3,eth,txpool,net \
--http --http.addr "0.0.0.0" \
--http.port 9945 --http.corsdomain '*' \
--syncmode full
Restart=on-failure
RestartSec=10
LimitNOFILE=4096

[Install]
WantedBy=multi-user.target
```
(Adjust --miner.threads and --miner.etherbase as needed.)

#### 3.6. Enable and Start the Service
```bash
sudo systemctl daemon-reload
sudo systemctl enable liberty-node
sudo systemctl start liberty-node
```

#### 3.7. Check Status and Logs
```bash
sudo systemctl status liberty-node
sudo journalctl -u liberty-node -f --no-hostname -o cat
```

### Stopping and Removing the Service
Regardless of how you installed the node (script, Docker, or manual):

#### Stop the Node (systemd)
```bash
sudo systemctl stop liberty-node
```

#### Disable the Node from Autostart
```bash
sudo systemctl disable liberty-node
```

#### (Optional) Remove the Service File
```bash
sudo rm /etc/systemd/system/liberty-node.service
sudo systemctl daemon-reload
```

#### (Optional) Remove Node Data and Binaries
If you want to remove everything:

Script / Manual installations typically store data in /home/<user>/go-liberty/ or /root/go-liberty/. Remove that folder:
```bash
sudo rm -rf /home/<user>/go-liberty
```

#### Docker container can be removed via:
```bash
sudo docker stop liberty-node-container && sudo docker rm liberty-node-container
sudo docker rmi liberty-node
```

#### Getting Help
If you encounter issues or have additional questions, please open an issue in this repository or contact the maintainers. For real-time logs:

```bash
sudo journalctl -u liberty-node -f --no-hostname -o cat
```

Thank you for using Liberty Node! Feel free to contribute improvements or open issues as needed.


## Command Line Options

| Option              | Description                                                                                       |
|---------------------|---------------------------------------------------------------------------------------------------|
| `--datadir`        | Specifies the data directory for the blockchain data.                                             |
| `--networkid`      | Defines the unique ID for the private network.                                                    |
| `--port`           | Specifies the network listening port for peer-to-peer communication.                              |
| `--http.api`       | Defines the APIs exposed over the HTTP RPC interface.                                             |
| `--mine`           | Enables mining on this node.                                                                      |
| `--miner.threads`  | Sets the number of CPU threads used for mining.                                                   |
| `--miner.etherbase`| Specifies the miner’s wallet address to receive mining rewards.                                   |
| `--http`           | Enables the HTTP-RPC server.                                                                      |
| `--http.addr`      | Sets the HTTP-RPC server address.                                                                 |
| `--http.port`      | Defines the HTTP-RPC server port.                                                                 |
| `--http.corsdomain`| Specifies allowed domains for CORS requests.                                                      |
| `--gcmode=archive` | Enables archival mode for retaining the full chain state.                                         |
| `--bootnodes`      | Adds boot nodes for initial connections to the network.                                           |

These options can be used to customize the node's operation and networking setup for specific requirements.


## License

This project is licensed under the MIT License.
