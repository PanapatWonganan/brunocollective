# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Inventory management system for small e-commerce (Bruno Collective). Features: product stock tracking, customer records, order management with payment slip uploads, shipping label printing (single + batch), admin dashboard with charts, and Telegram Bot API notifications.

## Tech Stack

- **Backend:** Go (Fiber v2) + GORM + SQLite
- **Frontend:** Vue 3 + Vuetify 3 + TypeScript + Vite
- **Auth:** JWT (HS256, 24h expiry)
- **Notifications:** Telegram Bot API (optional, gracefully disabled when unconfigured)

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
├── main.go              # Fiber app setup, route registration, CORS, static files, daily summary scheduler
├── config/              # Environment-based configuration
├── database/            # GORM connection, auto-migration, admin seed
├── models/              # Data models: User, Product, Customer, Order, OrderItem
├── handlers/            # HTTP handlers grouped by resource (auth, product, customer, order, dashboard)
├── services/            # External integrations (Telegram Bot API notifier)
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
- **Auth flow:** POST `/api/login` returns JWT. All other `/api/*` routes require `Authorization: Bearer <token>`. 401 responses auto-redirect to login on the frontend. Token stored in `localStorage` (`token` and `username` keys).
- **File uploads:** Slip images can be uploaded during order creation (multipart form) or separately via `POST /api/orders/:id/slip`. Stored in `backend/uploads/`, served as static files at `/uploads/`. 10MB body limit. Filenames: `slip_{orderID}_{timestamp}{ext}`.
- **Order creation:** `POST /api/orders` accepts both JSON and multipart form data. When using multipart, `customer_id`, `notes`, and `items` (JSON string) are form fields, with optional `slip` file attachment.
- **Stock management:** Creating an order deducts stock atomically in a DB transaction. Deleting an order restores stock via `gorm.Expr("stock + ?", qty)`.
- **Default admin:** On first run, seeds `admin` / `admin123`.
- **Frontend proxy:** Vite dev server proxies `/api` and `/uploads` to `http://localhost:8080`.
- **Handler dependency injection:** Handlers receive config and services via constructor (e.g., `NewOrderHandler(cfg, telegramNotifier)`).
- **Graceful degradation:** Telegram notifications are skipped silently when `TELEGRAM_BOT_TOKEN` or `TELEGRAM_CHAT_ID` are not set. All notification calls use goroutines to avoid blocking the HTTP response.

## Frontend Patterns

- **Path alias:** `@` maps to `frontend/src/` in both Vite and TypeScript configs.
- **Vuetify theme:** Brown/gold palette — primary `#1A1714` (dark brown), secondary `#C4A24D` (gold), background `#F7F3EE` (warm cream). Component defaults set globally in `plugins/vuetify.ts` (rounded cards, outlined inputs).
- **State management:** Only auth uses Pinia (`stores/auth.ts`). All other views manage state locally with Composition API `ref`/`reactive` — no global stores for products, customers, or orders.
- **API calls:** All views call `api` (Axios instance from `services/api.ts`) directly. No dedicated service layer per resource — fetch/create/update/delete logic lives inside each view's `<script setup>`.
- **Router guard:** `router/index.ts` checks `localStorage.getItem('token')` before each navigation. Routes with `meta: { public: true }` skip the check (only Login).
- **Layout:** `DefaultLayout.vue` wraps all authenticated routes with sidebar navigation and includes the change password dialog. Login has no layout wrapper.
- **No test framework:** The frontend has no unit or e2e tests configured. The backend also has no test files. Validation is done via `npx vue-tsc --noEmit` (type-check) and `npm run build` (build check).

## Telegram Notifications

`services/telegram.go` sends messages to a Telegram group/chat via Bot API on these events:
- **New order created** — includes item list, stock levels, today's sales summary, and slip image if attached
- **Order status changed** — includes new status, stock levels, and today's sales summary
- **Payment slip uploaded** — includes order/customer summary and slip image
- **Daily summary** — sent automatically at 8:00 AM Bangkok time (Asia/Bangkok); summarizes previous day's orders, revenue, and status breakdown. Can be manually triggered via `POST /api/daily-summary`.

Stock warnings are included automatically: `LOW` when stock <=5, `OUT OF STOCK` when 0.

Today's sales summary appears on every new order and status change notification, counting only non-cancelled orders since midnight (server local time).

Messages use MarkdownV2 formatting. Images are sent via `sendPhoto` with the public URL. To get the chat ID, add the bot to a group and use `https://api.telegram.org/bot<token>/getUpdates`.

## Shipping Label Printing

Both `OrdersView.vue` and `CustomersView.vue` generate 100mm x 150mm shipping labels via `window.open()` + `window.print()`. Sender address (Bruno Collective, Khon Kaen) is hardcoded in three places: OrdersView single print, OrdersView batch print (`printAllLabels()`), and CustomersView print.

**Batch print:** The "Print All Labels" button on the Orders page prints labels for all currently filtered orders that have customer addresses, with `page-break-after` between each label.

## API Routes

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/login` | No | Login, returns JWT |
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
| POST | `/api/daily-summary` | Yes | Manually trigger daily Telegram summary |

## Environment Variables

All optional, with defaults:
- `PORT` — API port (default: `8080`)
- `DB_PATH` — SQLite file path (default: `inventory.db`)
- `JWT_SECRET` — HMAC signing key (default: `change-me-in-production`)
- `UPLOAD_DIR` — Slip upload directory (default: `./uploads`)
- `TELEGRAM_BOT_TOKEN` — Telegram Bot API token from @BotFather (default: empty, disables notifications)
- `TELEGRAM_CHAT_ID` — Telegram group/chat ID to send notifications to (default: empty, disables notifications)
- `BASE_URL` — Public base URL for image links in Telegram notifications (default: `http://localhost:8080`)

## Deployment

Production runs on Vultr VPS (Ubuntu) with:
- **Nginx** reverse proxy serving frontend static files from `frontend/dist/` and proxying `/api/`, `/uploads/` to Go backend on port 8080
- **Cloudflare** DNS + SSL (Full Strict mode with Origin Certificate)
- **systemd** service (`inventory.service`) running the Go binary as `www-data`
- **EnvironmentFile** at `/opt/inventory/backend/.env` — values with special characters (like `/` or `=`) must be quoted
- **Deploy script** at `/opt/inventory/deploy.sh` — pulls from GitHub, rebuilds frontend + backend, restarts service
- **Backup cron** at `/opt/inventory/backup.sh` — daily at 2:00 AM, copies `inventory.db`, retains 30 days
