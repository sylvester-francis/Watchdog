#!/bin/sh
# WatchDog Agent Installer
#
# Usage:
#   curl -sSL https://usewatchdog.dev/install | sh
#   curl -sSL https://usewatchdog.dev/install | sh -s -- --api-key YOUR_KEY

set -e

INSTALL_DIR="/usr/local/bin"
BINARY_NAME="watchdog-agent"
GITHUB_REPO="sylvester-francis/watchdog-agent"
DEFAULT_HUB_URL="https://usewatchdog.dev"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Error: unsupported architecture: $ARCH"; exit 1 ;;
esac

case "$OS" in
    linux|darwin) ;;
    *) echo "Error: unsupported OS: $OS (use Linux or macOS)"; exit 1 ;;
esac

# Parse arguments
API_KEY=""
HUB_URL="$DEFAULT_HUB_URL"

while [ $# -gt 0 ]; do
    case "$1" in
        --api-key) API_KEY="$2"; shift 2 ;;
        --hub|--hub-url) HUB_URL="$2"; shift 2 ;;
        --help|-h)
            echo "Usage: install.sh [--api-key KEY] [--hub-url URL]"
            echo ""
            echo "Options:"
            echo "  --api-key KEY     Agent API key (prompted if not provided)"
            echo "  --hub-url URL     Hub URL (default: $DEFAULT_HUB_URL)"
            exit 0
            ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
done

echo ""
echo "  WatchDog Agent Installer"
echo "  ========================"

# Prompt for API key if not provided
if [ -z "$API_KEY" ]; then
    printf "  API Key: "
    read -r API_KEY < /dev/tty
    if [ -z "$API_KEY" ]; then
        echo "  Error: API key is required."
        echo "  Get your key from the WatchDog dashboard under Agents."
        exit 1
    fi
fi

# Check for curl
if ! command -v curl > /dev/null 2>&1; then
    echo "  Error: curl is required but not installed"
    exit 1
fi

echo "  OS:   $OS"
echo "  Arch: $ARCH"
echo "  Hub:  $HUB_URL"
echo ""

# Download binary from GitHub Releases
DOWNLOAD_URL="https://github.com/${GITHUB_REPO}/releases/latest/download/agent-${OS}-${ARCH}"

echo "  Downloading agent..."
if ! curl -fsSL -o "/tmp/${BINARY_NAME}" "$DOWNLOAD_URL"; then
    echo ""
    echo "  Error: download failed from $DOWNLOAD_URL"
    echo "  Check that a release exists at https://github.com/${GITHUB_REPO}/releases"
    exit 1
fi

chmod +x "/tmp/${BINARY_NAME}"

# Install binary (may need sudo)
if [ -w "$INSTALL_DIR" ]; then
    mv "/tmp/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
else
    echo "  Installing to ${INSTALL_DIR} (requires sudo)..."
    sudo mv "/tmp/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
fi

echo "  Installed to ${INSTALL_DIR}/${BINARY_NAME}"

# Create systemd service on Linux
if [ "$OS" = "linux" ] && command -v systemctl > /dev/null 2>&1; then
    echo "  Creating systemd service..."

    sudo tee /etc/systemd/system/watchdog-agent.service > /dev/null << EOF
[Unit]
Description=WatchDog Monitoring Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=${INSTALL_DIR}/${BINARY_NAME} --hub "${HUB_URL}" --api-key "${API_KEY}"
Restart=always
RestartSec=5
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF

    sudo systemctl daemon-reload
    sudo systemctl enable watchdog-agent
    sudo systemctl start watchdog-agent

    echo "  Agent started as systemd service"
    echo ""
    echo "  Useful commands:"
    echo "    sudo systemctl status watchdog-agent"
    echo "    sudo journalctl -u watchdog-agent -f"
else
    echo ""
    echo "  Run the agent:"
    echo "    ${BINARY_NAME} --hub \"${HUB_URL}\" --api-key \"${API_KEY}\""
fi

echo ""
echo "  Done! Your agent will appear in the dashboard shortly."
echo ""
