#!/bin/bash
set -e

# WatchDog VPS Setup Script
# Run this once on a fresh Ubuntu/Debian VPS

echo "=== WatchDog VPS Setup ==="

# Install Docker
if ! command -v docker &> /dev/null; then
    echo "Installing Docker..."
    curl -fsSL https://get.docker.com | sh
    sudo usermod -aG docker "$USER"
    echo "Docker installed. You may need to log out and back in for group changes."
fi

# Enable Docker on boot
sudo systemctl enable docker
sudo systemctl start docker

# Create app directory
sudo mkdir -p /opt/watchdog
sudo chown "$USER":"$USER" /opt/watchdog

# Clone repo
if [ ! -d /opt/watchdog/Watchdog ]; then
    git clone https://github.com/sylvester-francis/Watchdog.git /opt/watchdog/Watchdog
else
    cd /opt/watchdog/Watchdog && git pull
fi

cd /opt/watchdog/Watchdog/deployments

# Create .env if it doesn't exist
if [ ! -f .env ]; then
    DB_PASS=$(openssl rand -hex 16)
    ENC_KEY=$(openssl rand -hex 16)
    SESS_KEY=$(openssl rand -hex 16)

    cat > .env <<ENVEOF
DB_USER=watchdog
DB_PASSWORD=${DB_PASS}
DB_NAME=watchdog
ENCRYPTION_KEY=${ENC_KEY}
SESSION_SECRET=${SESS_KEY}
ENVEOF

    echo ""
    echo "=== Generated .env with random secrets ==="
    echo "Saved to /opt/watchdog/Watchdog/deployments/.env"
    echo ""
fi

# Open firewall ports
if command -v ufw &> /dev/null; then
    sudo ufw allow 80/tcp
    sudo ufw allow 443/tcp
    sudo ufw allow 22/tcp
    sudo ufw --force enable
    echo "Firewall configured (80, 443, 22)"
fi

echo ""
echo "=== Setup complete ==="
echo ""
echo "Next steps:"
echo "  1. Point usewatchdog.dev DNS (A record) to this server's IP"
echo "  2. Deploy:"
echo "     cd /opt/watchdog/Watchdog/deployments"
echo "     docker compose -f docker-compose.prod.yml up -d --build"
echo "  3. Check logs:"
echo "     docker compose -f docker-compose.prod.yml logs -f"
echo ""
