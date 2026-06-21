#!/usr/bin/env bash
#
# Bruno Collective deploy script — run on the VPS as root (or with sudo).
# Pulls latest code, rebuilds all three apps, and restarts services.
#
#   storefront (Next.js)  -> systemd: storefront.service  (port 3000)
#   backend    (Go)       -> systemd: inventory.service   (port 8080)
#   admin      (Vue)      -> static files served by Nginx from frontend/dist
#
# Install once: copy to /opt/inventory/deploy.sh and `chmod +x`.

set -euo pipefail

APP_DIR=/opt/inventory
cd "$APP_DIR"

# ---- Safety: back up the live SQLite database before doing anything ----
# inventory.db is gitignored and never touched by `git pull`, and this deploy
# changes no schema — but we snapshot it anyway so a deploy can always be undone.
DB_FILE="$APP_DIR/backend/inventory.db"
if [ -f "$DB_FILE" ]; then
  BACKUP_DIR="$APP_DIR/backups"
  mkdir -p "$BACKUP_DIR"
  STAMP=$(date +%Y%m%d_%H%M%S)
  # Use sqlite's online backup if available (consistent even while running),
  # else fall back to a plain copy.
  if command -v sqlite3 >/dev/null 2>&1; then
    sqlite3 "$DB_FILE" ".backup '$BACKUP_DIR/inventory_predeploy_$STAMP.db'"
  else
    cp "$DB_FILE" "$BACKUP_DIR/inventory_predeploy_$STAMP.db"
  fi
  echo "==> Backed up database to $BACKUP_DIR/inventory_predeploy_$STAMP.db"
else
  echo "==> No existing database at $DB_FILE (fresh install) — skipping backup"
fi

echo "==> Pulling latest from GitHub"
git pull origin main

echo "==> Building Go backend"
cd "$APP_DIR/backend"
go build -o server .

echo "==> Building Vue admin (base=/admin/)"
cd "$APP_DIR/frontend"
npm ci
npm run build

echo "==> Building Next.js storefront"
cd "$APP_DIR/storefront"
npm ci
npm run build

echo "==> Restarting services"
systemctl restart inventory.service
systemctl restart storefront.service

echo "==> Reloading Nginx"
nginx -t && systemctl reload nginx

echo "==> Done. Status:"
systemctl --no-pager --lines=0 status inventory.service storefront.service || true
