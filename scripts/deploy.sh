#!/bin/bash
set -e

# WatchDog Deploy Script
# Run from your local machine to deploy to production

VPS_HOST="${1:-watchdog-vps}"
REMOTE_DIR="/opt/watchdog/Watchdog"

echo "=== Deploying WatchDog to ${VPS_HOST} ==="

ssh "$VPS_HOST" "cd ${REMOTE_DIR} && git pull && cd deployments && docker compose -f docker-compose.prod.yml up -d --build"

echo ""
echo "=== Deployed. Checking health... ==="
sleep 5
ssh "$VPS_HOST" "docker compose -f ${REMOTE_DIR}/deployments/docker-compose.prod.yml ps"

echo ""
echo "Done. Site: https://usewatchdog.dev"
