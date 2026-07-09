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
#
# Ordering matters (learned from the 2026-06-29 deploy): the backend is built
# AND restarted before the frontends, so a frontend build failure can never
# leave a stale Go binary serving old routes. The storefront is built as root
# (www-data has no writable npm cache) and .next is chown'd back to www-data,
# whose service would otherwise crash-loop with EACCES on .next/cache.

set -euo pipefail

APP_DIR=/opt/inventory
cd "$APP_DIR"

# ---- Safety: back up the live SQLite database before doing anything ----
# inventory.db is gitignored and never touched by `git pull`; GORM AutoMigrate
# only adds tables/columns. We snapshot anyway so a deploy can always be undone.
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

# ---- Backend first: build, restart, verify — never leave a stale binary ----
echo "==> Building Go backend"
cd "$APP_DIR/backend"
go build -o server .

echo "==> Restarting backend"
systemctl restart inventory.service
sleep 2
# A public route must answer 200 — catches "old binary still running" instantly.
BACKEND_CODE=$(curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:8080/api/shop/products || true)
if [ "$BACKEND_CODE" != "200" ]; then
  echo "!! Backend check failed (/api/shop/products -> $BACKEND_CODE). See: journalctl -u inventory.service -n 50"
  exit 1
fi
echo "==> Backend OK (:8080)"

# ---- Admin (static — a failure here can't take anything down) ----
echo "==> Building Vue admin (base=/admin/)"
cd "$APP_DIR/frontend"
npm ci
npm run build

# ---- Storefront: build as root, hand .next to www-data, then restart ----
echo "==> Building Next.js storefront"
cd "$APP_DIR/storefront"
npm ci
npm run build
if [ ! -f .next/BUILD_ID ]; then
  echo "!! Storefront build produced no .next/BUILD_ID — aborting before restart"
  exit 1
fi
chown -R www-data:www-data .next

echo "==> Restarting storefront"
systemctl restart storefront.service
sleep 3
STORE_CODE=$(curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:3000/ || true)
if [ "$STORE_CODE" != "200" ]; then
  echo "!! Storefront check failed (:3000 -> $STORE_CODE). See: journalctl -u storefront.service -n 50"
  exit 1
fi
echo "==> Storefront OK (:3000)"

echo "==> Reloading Nginx"
nginx -t && systemctl reload nginx

echo "==> Done. Status:"
systemctl --no-pager --lines=0 status inventory.service storefront.service || true
