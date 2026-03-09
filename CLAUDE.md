# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Inventory management system for small e-commerce. Features: product stock tracking, customer records, order management with payment slip uploads, and an admin dashboard.

## Tech Stack

- **Backend:** Go (Fiber v2) + GORM + SQLite
- **Frontend:** Vue 3 + Vuetify 3 + TypeScript + Vite
- **Auth:** JWT (HS256, 24h expiry)

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
vue-tsc --noEmit            # Type-check only
```

### Running both together (development)
Terminal 1: `cd backend && go run .`
Terminal 2: `cd frontend && npm run dev`
Then open http://localhost:5173

## Architecture

```
backend/
├── main.go              # Fiber app setup, route registration, CORS, static files
├── config/              # Environment-based configuration (PORT, DB_PATH, JWT_SECRET, UPLOAD_DIR)
├── database/            # GORM connection, auto-migration, admin seed
├── models/              # Data models: User, Product, Customer, Order, OrderItem
├── handlers/            # HTTP handlers grouped by resource (auth, product, customer, order, dashboard)
├── middleware/           # JWT authentication middleware
└── uploads/             # Uploaded slip images (served at /uploads/)

frontend/
├── src/
│   ├── plugins/vuetify.ts   # Vuetify theme and component defaults
│   ├── router/              # Vue Router with auth guard
│   ├── stores/auth.ts       # Pinia auth store (login, logout, token management)
│   ├── services/api.ts      # Axios instance with JWT interceptor + 401 redirect
│   ├── layouts/             # DefaultLayout with sidebar navigation
│   └── views/               # Dashboard, Products, Customers, Orders, Login
└── vite.config.ts           # Dev proxy to backend
```

## Key Patterns

- **Database:** GORM AutoMigrate runs on startup; no manual migration files. SQLite database file at `backend/inventory.db`.
- **Auth flow:** POST `/api/login` returns JWT. All other `/api/*` routes require `Authorization: Bearer <token>`. 401 responses auto-redirect to login on the frontend.
- **File uploads:** Slip images uploaded via multipart form to `POST /api/orders/:id/slip`, stored in `backend/uploads/`, served as static files at `/uploads/`.
- **Stock management:** Creating an order deducts stock in a transaction. Deleting an order restores stock.
- **Default admin:** On first run, seeds `admin` / `admin123`.
- **Frontend proxy:** Vite dev server proxies `/api` and `/uploads` to `http://localhost:8080`, so no CORS issues in development.

## API Routes

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/login` | No | Login, returns JWT |
| GET | `/api/dashboard` | Yes | Stats summary |
| GET/POST | `/api/products` | Yes | List / Create product |
| GET/PUT/DELETE | `/api/products/:id` | Yes | Get / Update / Delete product |
| GET/POST | `/api/customers` | Yes | List / Create customer |
| GET/PUT/DELETE | `/api/customers/:id` | Yes | Get / Update / Delete customer |
| GET/POST | `/api/orders` | Yes | List / Create order |
| GET/DELETE | `/api/orders/:id` | Yes | Get / Delete order |
| PUT | `/api/orders/:id/status` | Yes | Update order status |
| POST | `/api/orders/:id/slip` | Yes | Upload payment slip |

## Environment Variables

All optional, with defaults:
- `PORT` — API port (default: `8080`)
- `DB_PATH` — SQLite file path (default: `inventory.db`)
- `JWT_SECRET` — HMAC signing key (default: `change-me-in-production`)
- `UPLOAD_DIR` — Slip upload directory (default: `./uploads`)
