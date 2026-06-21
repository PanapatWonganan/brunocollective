# Deploying the Storefront to Production

This guide adds the new **Next.js storefront** to the existing VPS, and moves
the **Vue admin** from `/` to `/admin`. The Go backend is unchanged except for
the new public shop API (`/api/shop/*`).

After this, the site looks like:

| URL | Served by |
|-----|-----------|
| `https://yourdomain.com/` | Next.js storefront (Node, port 3000) |
| `https://yourdomain.com/admin` | Vue admin (static) |
| `https://yourdomain.com/api/...` | Go backend (port 8080) |
| `https://yourdomain.com/uploads/...` | Go backend (payment slips) |

> Replace `yourdomain.com` and the cert paths with your real values. All
> commands run on the VPS as root (or with `sudo`).

---

## ⚠️ Database safety — read first

The live data lives in `/opt/inventory/backend/inventory.db` (SQLite). **This
deploy does not touch it:**

- `inventory.db` is gitignored, so `git pull` never overwrites it.
- No schema changes this release — GORM `AutoMigrate` only *adds* missing
  tables/columns, it never drops or rewrites data. The models are unchanged.
- The storefront writes orders to the **same** database the admin already uses
  (that's intended — one shared inventory).

Still, **always back up before deploying.** The deploy script (step 4) does this
automatically as its first action, but you can also do it manually right now:

```bash
sqlite3 /opt/inventory/backend/inventory.db \
  ".backup '/opt/inventory/backups/inventory_$(date +%F_%H%M%S).db'"
```

To restore if anything goes wrong: stop the backend, copy a backup over
`inventory.db`, start the backend again.

```bash
systemctl stop inventory.service
cp /opt/inventory/backups/<the-backup>.db /opt/inventory/backend/inventory.db
systemctl start inventory.service
```

---

## 0. Prerequisites (one-time)

The storefront needs **Node.js 20+** on the server. Check / install:

```bash
node -v   # need >= 20
# If missing or too old (Ubuntu):
curl -fsSL https://deb.nodesource.com/setup_22.x | bash -
apt-get install -y nodejs
```

Go is already installed (used by the backend). Confirm `go version`.

---

## 1. Push the code (from your laptop)

The storefront, the `/admin` base-path change, and the deploy files are all in
the repo. Commit and push to `main`:

```bash
git add .
git commit -m "Add Next.js storefront + move admin to /admin"
git push origin main
```

---

## 2. Pull on the server

```bash
cd /opt/inventory
git pull origin main
```

---

## 3. Install the storefront systemd service (one-time)

```bash
cp /opt/inventory/deploy/storefront.service /etc/systemd/system/storefront.service
systemctl daemon-reload
systemctl enable storefront.service
```

The unit runs `npm run start` in `/opt/inventory/storefront` on port 3000 with
`BACKEND_URL=http://localhost:8080` (used for server-side rendering fetches).

---

## 4. Build all three apps

Either run the deploy script (recommended)…

```bash
cp /opt/inventory/deploy/deploy.sh /opt/inventory/deploy.sh
chmod +x /opt/inventory/deploy.sh
/opt/inventory/deploy.sh
```

…or build manually:

```bash
cd /opt/inventory/backend   && go build -o server .
cd /opt/inventory/frontend  && npm ci && npm run build      # -> dist/ with /admin/ asset paths
cd /opt/inventory/storefront && npm ci && npm run build      # -> .next/ production build
```

---

## 5. Update Nginx (one-time)

```bash
cp /opt/inventory/deploy/nginx.conf /etc/nginx/sites-available/bruno
# Edit server_name + ssl_certificate paths to your real values:
nano /etc/nginx/sites-available/bruno

ln -sf /etc/nginx/sites-available/bruno /etc/nginx/sites-enabled/bruno
# If an old default/inventory site points at the root, disable it:
# rm /etc/nginx/sites-enabled/inventory   (or whatever the old one is called)

nginx -t            # must pass
systemctl reload nginx
```

---

## 6. Start the services

```bash
systemctl restart inventory.service     # Go backend (existing)
systemctl restart storefront.service    # Next.js storefront (new)
systemctl status inventory.service storefront.service --no-pager
```

---

## 7. Verify

```bash
curl -I  https://yourdomain.com/                 # 200 — storefront
curl -I  https://yourdomain.com/admin/           # 200 — admin SPA
curl -s  https://yourdomain.com/api/shop/products # JSON product list
```

Then in a browser:

- `/` shows the storefront landing page.
- `/shop` lists real products from the backend.
- Add to bag → checkout → upload a slip → **Place Order**.
- Confirm the order appears in the admin at `/admin` and stock decremented.

---

## Subsequent deploys

Once the service + Nginx are installed, every future deploy is just:

```bash
/opt/inventory/deploy.sh
```

It pulls `main`, rebuilds all three, and restarts the services.

---

## Rollback / troubleshooting

- **Storefront won't start:** `journalctl -u storefront.service -n 50 --no-pager`
  (common cause: Node too old, or `npm run build` was not run).
- **Admin assets 404 under /admin:** confirm `frontend/dist/index.html`
  references `/admin/assets/...` (it does after the Vite `base` change). Re-run
  `npm run build` in `frontend/`.
- **Slip upload fails with 413:** confirm `client_max_body_size 10M;` is in the
  Nginx server block (it is in the provided config).
- **Revert to admin-at-root:** point the Nginx `location /` back at
  `frontend/dist` and stop `storefront.service`. The backend is backward
  compatible, so nothing else needs changing.
