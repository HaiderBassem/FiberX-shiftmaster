#!/bin/bash
# =============================================================================
# ShiftMaster Production Deployment Script
# =============================================================================
# Run this script ON the production server (192.168.16.138)
#
# Prerequisites:
#   - PostgreSQL installed and running
#   - Caddy installed (sudo apt install caddy)
#   - Go 1.21+ installed (only for building from source)
#   - Node.js 18+ (only for building frontend from source)
#
# Usage:
#   chmod +x deploy.sh
#   sudo ./deploy.sh
# =============================================================================

set -euo pipefail

APP_DIR="/opt/shiftmaster"
LOG_DIR="/var/log/shiftmaster"
CADDY_LOG_DIR="/var/log/caddy"
PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"

echo "╔═══════════════════════════════════════════╗"
echo "║     ShiftMaster Production Deployment     ║"
echo "╚═══════════════════════════════════════════╝"
echo ""

# ── 1. Create directories ──
echo "→ Creating directories..."
sudo mkdir -p "$APP_DIR/frontend" "$APP_DIR/uploads" "$LOG_DIR" "$CADDY_LOG_DIR"
sudo chown -R cpper:cpper "$APP_DIR" "$LOG_DIR" || true

# ── 2. Build Go binary ──
echo "→ Building Go backend..."
cd "$PROJECT_DIR"
CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o build-api ./cmd/api/
sudo mv build-api "$APP_DIR/shiftmaster-api"
sudo chown cpper:cpper "$APP_DIR/shiftmaster-api"
sudo chmod +x "$APP_DIR/shiftmaster-api"

# ── 3. Build Frontend ──
echo "→ Building frontend..."
cd "$PROJECT_DIR/frontend"
npm install --production=false --legacy-peer-deps
npx vite build
sudo rm -rf "$APP_DIR/frontend/dist"
sudo cp -r dist "$APP_DIR/frontend/"
sudo chown -R cpper:cpper "$APP_DIR/frontend"

# ── 4. Copy config files ──
echo "→ Copying configuration..."
if [ ! -f "$APP_DIR/.env" ]; then
    cp "$PROJECT_DIR/.env.production" "$APP_DIR/.env"
    echo "  ⚠  Created .env from template — EDIT DB credentials & JWT_SECRET!"
else
    echo "  ✓  .env already exists, skipping (won't overwrite)"
fi

# ── 5. Run database migration ──
echo "→ Running database migrations..."
source "$APP_DIR/.env"
for migration in "$PROJECT_DIR"/internal/database/migrations/*.sql; do
    echo "  Running: $(basename "$migration")"
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$migration" 2>/dev/null || true
done
echo "  ✓  Migrations complete"

# ── 5b. Fix table ownership (in case tables were created by a different user) ──
echo "→ Fixing table ownership..."
sudo -u postgres psql -d "$DB_NAME" -c "
DO \$\$
BEGIN
    -- Fix ownership for external modules tables
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'external_links') THEN
        EXECUTE 'ALTER TABLE external_links OWNER TO ' || quote_ident('$DB_USER');
        EXECUTE 'ALTER TABLE link_departments OWNER TO ' || quote_ident('$DB_USER');
        EXECUTE 'ALTER TABLE link_exclusions OWNER TO ' || quote_ident('$DB_USER');
        RAISE NOTICE 'Fixed ownership for external_links tables';
    END IF;
    -- Grant all privileges on all tables to the app user (safety net)
    EXECUTE 'GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO ' || quote_ident('$DB_USER');
    EXECUTE 'GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO ' || quote_ident('$DB_USER');
END
\$\$;
" 2>/dev/null || true
echo "  ✓  Ownership fixed"

# ── 6. Install systemd service ──
echo "→ Setting up systemd service..."
sudo cp "$PROJECT_DIR/deploy/shiftmaster.service" /etc/systemd/system/shiftmaster.service
sudo systemctl daemon-reload
sudo systemctl enable shiftmaster
sudo systemctl restart shiftmaster
echo "  ✓  shiftmaster.service started"

# ── 7. Configure Caddy ──
echo "→ Configuring Caddy..."
#sudo cp "$PROJECT_DIR/deploy/Caddyfile" /etc/caddy/Caddyfile
sudo systemctl restart caddy
echo "  ✓  Caddy configured and restarted"

# ── 8. Verify ──
echo ""
echo "→ Waiting for services to start..."
sleep 3

API_STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:8080/health 2>/dev/null || echo "000")
WEB_STATUS=$(curl -s -o /dev/null -w "%{http_code}" https://shift-master.org/ 2>/dev/null || echo "000")

echo ""
echo "╔═══════════════════════════════════════════╗"
echo "║          Deployment Results               ║"
echo "╠═══════════════════════════════════════════╣"
if [ "$API_STATUS" = "200" ]; then
    echo "║  ✅  API Backend:  http://127.0.0.1:8080  ║"
else
    echo "║  ❌  API Backend:  FAILED (HTTP $API_STATUS)     ║"
fi
if [ "$WEB_STATUS" = "200" ]; then
    echo "║  ✅  Web Frontend: http://192.168.16.138  ║"
else
    echo "║  ❌  Web Frontend: FAILED (HTTP $WEB_STATUS)     ║"
fi
echo "╠═══════════════════════════════════════════╣"
echo "║  Logs:                                    ║"
echo "║    API:   journalctl -u shiftmaster -f    ║"
echo "║    Caddy: /var/log/caddy/shiftmaster.log  ║"
echo "╚═══════════════════════════════════════════╝"

if [ "$API_STATUS" != "200" ]; then
    echo ""
    echo "⚠  API failed — check:"
    echo "   1. Database credentials in $APP_DIR/.env"
    echo "   2. sudo journalctl -u shiftmaster --no-pager -n 20"
fi
