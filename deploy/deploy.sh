#!/bin/bash
# =============================================================================
# ShiftMaster Production Deployment Script
# =============================================================================
# Run this script ON the production server.
#
# Prerequisites:
#   - PostgreSQL installed and running
#   - Caddy installed (sudo apt install caddy)
#   - Go 1.26+ installed (only for building from source)
#   - Node.js 20+ (only for building frontend from source)
#
# Usage:
#   chmod +x deploy.sh
#   sudo ./deploy.sh
#
# FIRST RUN ON AN EXISTING DATABASE
#   The migration series is now recorded in a schema_migrations table. A database
#   that predates that table has the schema but no record of it, so the migrator
#   would try to replay all 49 files. Adopt the ledger once, before deploying:
#
#       ./shiftmaster-migrate -dir internal/database/migrations baseline
#
#   That records every file as applied without executing any of it. Only do this
#   on a database whose schema already matches the series.
# =============================================================================

set -euo pipefail

APP_DIR="/opt/shiftmaster"
LOG_DIR="/var/log/shiftmaster"
CADDY_LOG_DIR="/var/log/caddy"
PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"

# The account the systemd unit runs as. Every deployed path is owned by it.
SERVICE_USER="techsupport"
SERVICE_GROUP="techsupport"

echo "============================================="
echo "      ShiftMaster Production Deployment      "
echo "============================================="
echo ""

# ── 1. Create directories ──
echo "[INFO] Creating directories..."
sudo mkdir -p "$APP_DIR/frontend" "$APP_DIR/uploads" "$LOG_DIR" "$CADDY_LOG_DIR"
sudo chown -R "$SERVICE_USER:$SERVICE_GROUP" "$APP_DIR" "$LOG_DIR"

# Uploads are served to authenticated users and must not be world-readable on
# disk; the service account needs to write them.
sudo chmod 750 "$APP_DIR/uploads"

# ── 2. Build Go binaries ──
echo "[INFO] Building Go backend..."
cd "$PROJECT_DIR"
CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o build-api ./cmd/api/
CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o build-migrate ./cmd/migrate/

sudo mv build-api "$APP_DIR/shiftmaster-api"
sudo mv build-migrate "$APP_DIR/shiftmaster-migrate"
# Owned by the account that runs them. This previously used a different user,
# and only worked because chmod +x grants world-execute.
sudo chown "$SERVICE_USER:$SERVICE_GROUP" "$APP_DIR/shiftmaster-api" "$APP_DIR/shiftmaster-migrate"
sudo chmod 750 "$APP_DIR/shiftmaster-api" "$APP_DIR/shiftmaster-migrate"

# ── 3. Build Frontend ──
echo "[INFO] Building frontend..."
cd "$PROJECT_DIR/frontend"
rm -rf dist
npm ci --include=dev --legacy-peer-deps
npm run build
sudo rm -rf "$APP_DIR/frontend/dist"
sudo cp -r dist "$APP_DIR/frontend/"
sudo chown -R "$SERVICE_USER:$SERVICE_GROUP" "$APP_DIR/frontend"
cd "$PROJECT_DIR"

# ── 4. Verify configuration ──
echo "[INFO] Verifying configuration..."
if [ ! -f "$APP_DIR/.env" ]; then
    echo "  [FATAL] $APP_DIR/.env not found. Create it from .env.example before deploying." >&2
    exit 1
fi
# Credentials live here; keep them off other accounts.
sudo chown "$SERVICE_USER:$SERVICE_GROUP" "$APP_DIR/.env"
sudo chmod 600 "$APP_DIR/.env"
echo "  [OK] .env present"

# ── 5. Run database migrations ──
# Errors are fatal. The previous version piped every migration through
# `psql ... 2>/dev/null || true`, which discarded failures and then printed
# "Migrations complete" regardless, so a broken schema reached production
# looking like a successful deploy.
echo "[INFO] Running database migrations..."
set -a
# shellcheck disable=SC1091
source "$APP_DIR/.env"
set +a

sudo -E -u "$SERVICE_USER" "$APP_DIR/shiftmaster-migrate" \
    -dir "$PROJECT_DIR/internal/database/migrations" up
echo "  [OK] Migrations applied"

# ── 6. Install systemd service ──
echo "[INFO] Setting up systemd service..."
sudo cp "$PROJECT_DIR/deploy/shiftmaster.service" /etc/systemd/system/shiftmaster.service
sudo systemctl daemon-reload
sudo systemctl enable shiftmaster
sudo systemctl restart shiftmaster
echo "  [OK] shiftmaster.service started"

# ── 7. Configure Caddy ──
echo "[INFO] Configuring Caddy..."
if [ -f "$PROJECT_DIR/Caddyfile" ]; then
    sudo caddy validate --config "$PROJECT_DIR/Caddyfile" --adapter caddyfile
    sudo cp "$PROJECT_DIR/Caddyfile" /etc/caddy/Caddyfile
fi
sudo systemctl reload caddy || sudo systemctl restart caddy
echo "  [OK] Caddy configured"

# ── 8. Verify ──
echo ""
echo "[INFO] Waiting for the API to become healthy..."

API_STATUS="000"
for _ in $(seq 1 15); do
    API_STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:8080/health || echo "000")
    [ "$API_STATUS" = "200" ] && break
    sleep 1
done

echo ""
echo "============================================="
echo "             Deployment Results              "
echo "============================================="
if [ "$API_STATUS" = "200" ]; then
    echo "  [OK]     API Backend:  http://127.0.0.1:8080"
else
    echo "  [FAILED] API Backend:  HTTP $API_STATUS"
fi
echo "---------------------------------------------"
echo "  Logs:                                      "
echo "    API:   journalctl -u shiftmaster -f      "
echo "    Caddy: /var/log/caddy/shiftmaster.log    "
echo "============================================="

# A deploy that leaves the API unhealthy is a failed deploy, and must say so
# through its exit status so any surrounding automation stops.
if [ "$API_STATUS" != "200" ]; then
    echo ""
    echo "[ERROR] The API did not become healthy. Check:" >&2
    echo "   1. Database credentials in $APP_DIR/.env" >&2
    echo "   2. sudo journalctl -u shiftmaster --no-pager -n 50" >&2
    exit 1
fi
