# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Inventory management system for small e-commerce (Bruno Collective). Features: product stock tracking, customer records, order management with payment slip uploads, shipping label printing (single + batch), admin dashboard with charts, and LINE Messaging API notifications.

## Tech Stack

- **Backend:** Go (Fiber v2) + GORM + SQLite
- **Frontend:** Vue 3 + Vuetify 3 + TypeScript + Vite
- **Auth:** JWT (HS256, 24h expiry)
- **Notifications:** LINE Messaging API (optional, gracefully disabled when unconfigured)

## Common Commands

### Backend (run from `backend/`)
```bash
go run .                    # Start API server on :8080
go build -o server .        # Build binary
go mod tidy                 # Sync dependencies
```

### Frontend (run from `frontend/`)
```bash
npm run dev                 # Start dev server on :5173 (proxies /api and /uploads to :8080)
npm run build               # Type-check + production build
npx vue-tsc --noEmit        # Type-check only
```

### Running both together (development)
Terminal 1: `cd backend && go run .`
Terminal 2: `cd frontend && npm run dev`
Then open http://localhost:5173

## Architecture

```
backend/
├── main.go              # Fiber app setup, route registration, CORS, static files, LINE webhook, daily summary scheduler
├── config/              # Environment-based configuration
├── database/            # GORM connection, auto-migration, admin seed
├── models/              # Data models: User, Product, Customer, Order, OrderItem
├── handlers/            # HTTP handlers grouped by resource (auth, product, customer, order, dashboard)
├── services/            # External integrations (LINE Messaging API notifier)
├── middleware/           # JWT authentication middleware
└── uploads/             # Uploaded slip images (served at /uploads/)

frontend/
├── src/
│   ├── plugins/vuetify.ts   # Vuetify theme (brown/gold palette) and component defaults
│   ├── router/              # Vue Router with auth guard
│   ├── stores/auth.ts       # Pinia auth store (login, logout, token management)
│   ├── services/api.ts      # Axios instance with JWT interceptor + 401 redirect
│   ├── layouts/             # DefaultLayout with sidebar navigation + change password dialog
│   └── views/               # Dashboard, Products, Customers, Orders, Login
└── vite.config.ts           # Dev proxy to backend
```

## Key Patterns

- **Database:** GORM AutoMigrate runs on startup; no manual migration files. SQLite database file at `backend/inventory.db`.
- **Auth flow:** POST `/api/login` returns JWT. All other `/api/*` routes require `Authorization: Bearer <token>`. 401 responses auto-redirect to login on the frontend.
- **File uploads:** Slip images can be uploaded during order creation (multipart form) or separately via `POST /api/orders/:id/slip`. Stored in `backend/uploads/`, served as static files at `/uploads/`. 10MB body limit. Filenames: `slip_{orderID}_{timestamp}{ext}`.
- **Order creation:** `POST /api/orders` accepts both JSON and multipart form data. When using multipart, `customer_id`, `notes`, and `items` (JSON string) are form fields, with optional `slip` file attachment.
- **Stock management:** Creating an order deducts stock atomically in a DB transaction. Deleting an order restores stock via `gorm.Expr("stock + ?", qty)`.
- **Default admin:** On first run, seeds `admin` / `admin123`.
- **Frontend proxy:** Vite dev server proxies `/api` and `/uploads` to `http://localhost:8080`.
- **Handler dependency injection:** Handlers receive config and services via constructor (e.g., `NewOrderHandler(cfg, lineNotifier)`).
- **Graceful degradation:** LINE notifications are skipped silently when `LINE_CHANNEL_TOKEN` or `LINE_GROUP_ID` are not set. All notification calls use goroutines to avoid blocking the HTTP response.

## LINE Notifications

`services/line.go` sends messages to a LINE group via Messaging API on these events:
- **New order created** — includes item list, stock levels, today's sales summary, and slip image if attached
- **Order status changed** — includes new status, stock levels, and today's sales summary
- **Payment slip uploaded** — includes order/customer summary and slip image
- **Daily summary** — sent automatically at 8:00 AM Bangkok time (Asia/Bangkok); summarizes previous day's orders, revenue, and status breakdown. Can be manually triggered via `POST /api/daily-summary`.

Stock warnings are included automatically: `LOW` when stock <=5, `OUT OF STOCK` when 0.

Today's sales summary appears on every new order and status change notification, counting only non-cancelled orders since midnight (server local time).

The webhook endpoint `POST /webhook/line` (public, no auth) logs the Group ID when the bot joins a group — used during initial setup only. Debug logging is enabled for webhook payloads.

## Shipping Label Printing

Both `OrdersView.vue` and `CustomersView.vue` generate 100mm x 150mm shipping labels via `window.open()` + `window.print()`. Sender address (Bruno Collective, Khon Kaen) is hardcoded in three places: OrdersView single print, OrdersView batch print (`printAllLabels()`), and CustomersView print.

**Batch print:** The "Print All Labels" button on the Orders page prints labels for all currently filtered orders that have customer addresses, with `page-break-after` between each label.

## API Routes

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/login` | No | Login, returns JWT |
| POST | `/webhook/line` | No | LINE platform webhook (logs group ID) |
| GET | `/api/dashboard` | Yes | Stats summary (totals, low stock, recent orders) |
| GET | `/api/dashboard/charts` | Yes | Chart data (revenue series, status distribution, stock overview, top products) |
| PUT | `/api/change-password` | Yes | Change authenticated user's password |
| GET/POST | `/api/products` | Yes | List / Create product |
| GET/PUT/DELETE | `/api/products/:id` | Yes | Get / Update / Delete product |
| GET/POST | `/api/customers` | Yes | List / Create customer |
| GET/PUT/DELETE | `/api/customers/:id` | Yes | Get / Update / Delete customer |
| GET/POST | `/api/orders` | Yes | List / Create order (supports multipart form with slip) |
| GET/DELETE | `/api/orders/:id` | Yes | Get / Delete order |
| PUT | `/api/orders/:id/status` | Yes | Update order status |
| POST | `/api/orders/:id/slip` | Yes | Upload payment slip |
| POST | `/api/daily-summary` | Yes | Manually trigger daily LINE summary |

## Environment Variables

All optional, with defaults:
- `PORT` — API port (default: `8080`)
- `DB_PATH` — SQLite file path (default: `inventory.db`)
- `JWT_SECRET` — HMAC signing key (default: `change-me-in-production`)
- `UPLOAD_DIR` — Slip upload directory (default: `./uploads`)
- `LINE_CHANNEL_TOKEN` — LINE Messaging API channel access token (default: empty, disables notifications)
- `LINE_GROUP_ID` — LINE group ID to send notifications to (default: empty, disables notifications)
- `BASE_URL` — Public base URL for image links in LINE notifications (default: `http://localhost:8080`)

## Deployment

Production runs on Vultr VPS (Ubuntu) with:
- **Nginx** reverse proxy serving frontend static files from `frontend/dist/` and proxying `/api/`, `/uploads/`, `/webhook/` to Go backend on port 8080
- **Cloudflare** DNS + SSL (Full Strict mode with Origin Certificate)
- **systemd** service (`inventory.service`) running the Go binary as `www-data`
- **EnvironmentFile** at `/opt/inventory/backend/.env` — values with special characters (like `/` or `=`) must be quoted
- **Deploy script** at `/opt/inventory/deploy.sh` — pulls from GitHub, rebuilds frontend + backend, restarts service
- **Backup cron** at `/opt/inventory/backup.sh` — daily at 2:00 AM, copies `inventory.db`, retains 30 days
