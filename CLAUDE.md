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
- **Product variants + per-variant price:** `ProductVariant{Size, Color, SKU, Stock, Price}` (`models/variant.go`). `Price` is an optional override (0 = inherit `Product.Price`) for cases like a ring whose gold colour costs more than silver; `ProductVariant.UnitPrice(base)` resolves it and `Product.PriceRange()` gives min/max. `buildOrderItem` (`handlers/order.go`, shared by every creation path) prices a variant line at its `UnitPrice` unless a sale-page offer/bump `priceOverride` is passed — so sale-page offer prices still apply to every colour. Client side: `storefront/lib/format.ts` has `unitPrice(product, variant)` / `priceRange` / `priceLabel` ("฿590 – ฿790"); cart totals, bag drawer, checkout, cards, the product page header (range) + `AddToBag` (exact price once a variant is chosen), JSON-LD (`AggregateOffer` when the range differs) and the sale-page catalog price all go through them. Admin: variant row has a Price field (placeholder = base price); OrdersView/ChatView subtotals and variant labels are variant-price-aware. Stale carts whose variant predates the field fall back to the product price.
- **Size charts:** Site Image slots `size_chart` (shirts, default), `size_chart_shoes` (category contains รองเท้า/shoe) and `size_chart_rings` (category contains แหวน/เครื่องประดับ/จิวเวอ/ring/jewel) — resolved by `sizeChartKey` in `storefront/lib/api.ts`; the chart is shown under the size swatches on the product page and sale-page order form only when the product has sizes and the slot has an upload.
- **Coupons:** Percent/fixed discount codes with min-order, start/expiry window, total + per-customer usage limits (`handlers/coupon.go`). Codes are matched case-insensitively (stored uppercase). Validation + quota claim happen inside the order-creation transaction (`applyCouponToOrder`); the total-usage limit is enforced with a guarded UPDATE so concurrent checkouts can't oversubscribe. Deleting an order releases the coupon usage (like stock). Orders store `subtotal`, `discount_amount`, and a `coupon_code` snapshot; legacy orders have `subtotal = 0` — readers fall back to `total_amount`. Client-side discounts are previews only; the server recomputes from DB prices. Coupon validation error messages are Thai (shown to shoppers as-is).
- **Membership:** Customers carry `is_member`/`member_since`/`password_hash`. Members get a flat 5% discount (`handlers/member.go`, `MemberDiscountPercent`) on every order in all three creation paths (admin, storefront checkout, sale page), stored in `Order.MemberDiscount` — **separate from coupons**: both discounts are computed on the subtotal and stack (`TotalAmount = Subtotal - MemberDiscount - DiscountAmount`; coupon capped so the total can't go negative). Membership comes from (a) storefront register/login by phone + password (`/member`, `/member/account` pages; token in localStorage `bc_member_token`, JWT claims `{customer_id, role: "member"}`, 30-day expiry), (b) admin toggle (`POST /api/customers/:id/member`, switch in CustomersView), or (c) auto-membership: `ensureMembership` promotes any customer with a prior order at their next checkout (so admin-revoking a customer with order history only lasts until their next order). Admin middleware rejects member tokens (role check) and vice versa; the chat WS also requires `user_id` claims. Guest checkout shows the 5% preview via `POST /api/shop/members/check` (phone → boolean only, no PII). Member Thai error messages are shown to shoppers as-is.
- **Sale pages (funnels):** ClickFunnels-style landing pages built in the admin ("Sale Pages" menu) and rendered by the storefront at `/s/{slug}` (`app/s/[slug]`). Content is an ordered JSON `sections` array (hero, pain, story, showcase, offer stack, testimonials, guarantee, faq) whose field shapes are agreed between `SalePagesView.vue` and `SalePageClient.tsx` — the backend just stores them. Offer price and order-bump price override catalog prices and are resolved server-side in `POST /api/shop/sale-pages/:slug/order` (client prices are display-only). A countdown is a hard deadline: expired pages reject orders. Draft pages 404 publicly; `?preview=1` shows drafts and skips the view counter. Stats (views, orders_count, bump_count) update in the order transaction; orders carry `sale_page_id` for attribution.
- **Analytics:** `handlers/analytics.go` powers the 4 dashboard tabs (`frontend/src/views/dashboard/`). Products carry `cost` (unit cost; 0 = unknown, surfaced as `cost_coverage`) and `category`; orders carry `channel` (stamped server-side: storefront checkout → `storefront`, sale-page order → `sale-page`; the admin order form sends a chosen channel). SQLite gotchas: `MIN`/`MAX` on datetime columns lose the column type and won't scan into `time.Time` — load rows and aggregate in Go; new columns are NULL on legacy rows, so `CASE WHEN col = ''` needs `COALESCE(col, '')`.
- **Chat inbox:** `Conversation`/`ChatMessage` (`models/chat.go`) are platform-agnostic (platform + external_id unique). LINE: webhook `POST /api/webhooks/line` verifies `X-Line-Signature` (HMAC-SHA256, base64); replies via Push API (`services/line.go`). Facebook Messenger + Instagram DM share one Meta app: `GET /api/webhooks/meta` answers the hub.challenge handshake, `POST` verifies `X-Hub-Signature-256` (HMAC hex) and maps `object: page` → platform `facebook`, `object: instagram` → `instagram`; replies via the Send API (`services/meta.go`, page access token serves both). Echo events (`is_echo`) record replies typed in the FB/IG inbox apps as `out` messages; our own Send API mids are stored so echoes dedupe. All webhooks 200 valid events (platforms retry non-2xx), dedupe by message id, download media to `uploads/chat_*`, and replies are saved only when the platform accepted them. **LINE has no echo events** — replies typed in the LINE OA app never reach the webhook (module channels that would sync them are LINE-corporate-partner-only), so those threads stay "รอตอบ" until the admin replies in-system or uses the "ตอบจากที่อื่นแล้ว" button (`POST /api/chats/:id/answered` — clears waiting/unread, sets `last_direction=out`). Realtime: `services/chathub.go` broadcasts to admin WebSockets at `/api/ws/chat?token=<JWT>` (query param — browsers can't set WS headers); `ChatView.vue` falls back to 20s polling when the socket is down. Workflow state on `Conversation`: `status` (open/done — closing clears unread + waiting; an inbound message or a reply reopens), `last_direction` + `waiting_since` (set on first unanswered inbound, cleared on reply — drives the "รอตอบ" tab, oldest-first, and the sidebar badge via `/api/chats/summary`), `tags` (JSON TEXT like Product.Images), plus `CannedReply` templates inserted from the composer.
- **Comment auto-reply:** the Meta webhook also receives `entry.changes` — FB `feed` (item=comment, verb=add only) and IG `comments` — normalized into `CommentEvent` and handed to `handlers/autoreply.go` in a goroutine (webhooks must answer fast). The first enabled `AutoReplyRule` (priority ASC, id ASC) whose keywords substring-match (case-insensitive; empty = catch-all) wins; actions are public reply, private reply (Send API `comment_id` recipient, one per comment), and hide. `{name}` in texts = commenter name. Loop guard: comments where `from.id` equals the entry's page/IG id are skipped (our own replies come back as webhook events). One `AutoReplyLog` row per comment id doubles as the redelivery dedupe; status success/partial/failed with the first error kept. Rules with `apply_to_chats` also answer inbound **chat messages** (LINE/FB/IG DM, platform `line` is chat-only): `HandleChatMessage` sends `reply_text` ({name} = display name) via the platform client, saves the out message only when accepted, logs action `chat_reply` with `comment_id = "chat:{msgID}"`, and has a 10-min per-rule/per-person cooldown. Deliberate: a bot chat reply does NOT clear `waiting_since`/`last_direction` — the thread stays in the "รอตอบ" queue for a human.
- **Order from chat + payment link:** the chat inbox can close a sale in-thread (`handlers/pay.go`). `POST /api/chats/:id/order` (admin) creates an order for the conversation's **linked customer** (400 with a Thai error if unlinked) using the same pipeline as the other creation paths (stock, membership, coupon), stamps `channel` from the conversation platform, and sets `Order.ConversationID` (attribution) + an unguessable `Order.PaymentToken`; it returns `{order, pay_url}` where `pay_url = {BASE_URL}/pay/{token}` — so **BASE_URL must be the public site root in production**. ChatView then pushes a Thai summary + link through the normal reply endpoint (send failure falls back to staging the message in the composer). The storefront page `app/pay/[token]` (public, no-store, noindex) shows items/amounts only — never phone/address — and uploads slips via `POST /api/pay/:token/slip` (image-only, re-upload allowed, blocked when cancelled; fires the Telegram slip notification). Stock is still deducted at creation, so an unpaid chat order holds inventory until it's deleted. **Follow-ups (ดีลค้าง):** `GET /api/chats/followups` lists conversations with pending no-slip chat orders (oldest first) → ChatView's "ดีลค้าง" tab + an in-thread alert with quick actions; `POST /api/orders/:id/followup-discount` (`{type, value, expires_hours}`) creates a single-use `CHAT{orderID}-XXXX` coupon and applies it to the order in one transaction (rejected if the order already has a coupon or isn't pending), so the /pay page shows the new total without the customer typing a code.
- **Chat waiting-state hygiene (LINE has no echo):** because replies typed in the LINE OA app are invisible, the "รอตอบ" state is treated as a hint, not truth: (1) `expireChatWaiting` (same minute ticker as SLA, `handlers/sla.go`) clears `waiting_since`/sets `last_direction=out` on open threads waiting longer than `CHAT_WAITING_EXPIRE_HOURS` (default 12, 0 = off) with no newer inbound, and zeroes `unread_count` on threads whose last message is older than the same cutoff (stale unread would otherwise pile up in the badge), then broadcasts `{"type":"conversations"}` on the WS so ChatView refetches; (2) the sidebar badge uses `/api/chats/summary` → `unread_threads` (threads with `unread_count > 0`), never `waiting`; (3) ChatView's รอตอบ tab distinguishes unread (bold, red/amber chip) from seen-but-unanswered (muted, grey "อ่านแล้ว" chip), and has "ตอบที่อื่นแล้วทั้งหมด" → `POST /api/chats/answered` `{ids}` or `{all, platform}` (bulk version of `/chats/:id/answered`).
- **Meta chat niceties:** (1) read receipts — webhook events with `read.watermark` (fields `message_reads` FB / `messaging_seen` IG, **must be subscribed** on the page/IG account) store `Conversation.CustomerReadAt`; ChatView shows "ลูกค้าอ่านแล้ว" under the newest outbound message at or before the watermark. (2) `MetaClient.SenderAction` sends `mark_seen` when the admin opens a thread with unread messages (`MarkRead`) and `typing_on` via `POST /api/chats/:id/typing` (ChatView throttles to one ping per 12s while composing); both no-ops for LINE. (3) 24-hour window: `SendText` detects Meta error code 10 / subcode 2018278, retries once with `messaging_type: MESSAGE_TAG, tag: HUMAN_AGENT` (works only if the page has the Human Agent permission from App Review) and otherwise returns a Thai "เกิน 24 ชม." error; ChatView also warns above the composer when the last inbound is older than 24h. (4) An echo (reply typed in the Meta inbox app) clears `unread_count` as well as the waiting state.
- **Chat SLA alerts:** `handlers/sla.go` checks every minute for open conversations with `last_direction = 'in'` waiting longer than `CHAT_SLA_MINUTES` (default 10, 0 = off) and sends one Telegram summary per waiting period — `Conversation.SlaAlertedAt` suppresses repeats; replying clears `waiting_since`, so only a new unanswered inbound re-alerts.
- **LINE broadcast:** `handlers/broadcast.go` + `BroadcastView.vue` ("Broadcast" menu). Audience = every **LINE** conversation (FB/IG deliberately excluded — Meta's 24h messaging window bars promo pushes), bucketed by the linked customer's RFM segment via the same `segmentOf` as analytics, plus special buckets "ยังไม่เคยซื้อ" and "ยังไม่ผูกลูกค้า". `POST /api/broadcasts` creates a `Broadcast` row (status `sending`) and pushes in a goroutine (150ms pacing, `renderTemplate` for `{name}`); sent/failed tick up for the UI's 4s poll and status flips to `done`. Each delivered message is saved as an out `ChatMessage` with `Source: "broadcast"` and refreshes the thread preview **without** clearing waiting state. LINE OA message quota applies to pushes.
- **AI chat assistant:** `services/ai.go` (official Go SDK `anthropic-sdk-go`) + `handlers/aichat.go`. Enabled by `ANTHROPIC_API_KEY` (model via `AI_MODEL`, default `claude-opus-5`; effort low). When an inbound chat message matches **no** keyword rule, `tryAIReply` answers from a live catalog snapshot (`buildAIShopContext`: prices + per-variant stock, ≤150 products) with the last 12 messages as history. Guardrails: Thai system prompt forbids invented info/promises; the model outputs `[HANDOFF]` (payment confirmation, shipping status, bargaining, complaints, unknown info) → no reply is sent and the thread stays with the human; `stop_reason: "refusal"` is treated as handoff too. A staleness check skips the reply if a newer inbound arrived during generation. AI replies keep `waiting_since` (thread stays in รอตอบ), save with `ChatMessage.Source: "ai"` (UI shows an "AI" label; `"rule"`/`"broadcast"` likewise), and log to `AutoReplyLog` as `ai_reply`/`ai_handoff` (`comment_id = "chat-ai:{msgID}"`). Per-thread kill switch: `Conversation.AiDisabled` via `POST /api/chats/:id/ai` (ChatView toggle, shown when `/api/chats/summary` reports `ai_enabled`).
- **Virtual try-on (ลองใส่ดู):** `services/tryon.go` (Gemini image model via REST, no SDK) + `handlers/tryon.go`. Enabled by `GEMINI_API_KEY` (model `TRYON_MODEL`, default `gemini-3.1-flash-image`); **image generation needs billing enabled on the Google AI Studio project — the free tier allows 0 image requests (HTTP 429)**. `GET /api/shop/try-on` → `{enabled, members_only, remaining, limit, member}` (storefront hides the button when disabled); **members only by default** (`TRYON_MEMBERS_ONLY`, owner decision 2026-09-23 on cost): guests see the button, the dialog footer links to `/member?next=<product path>`, and POST answers 401 `{members_only: true}` without a member JWT. Cost ≈ $0.067 (~฿2.3) per generated 1K image on `gemini-3.1-flash-image` (input tokens negligible); `gemini-3.1-flash-lite-image` ≈ $0.034 and `gemini-2.5-flash-image` ≈ $0.039 are the cheaper options via `TRYON_MODEL` (quality untested); the one-time garment cutout warm-up costs the same per product image; `POST /api/shop/try-on` multipart `photo` + `product_id` + optional `image` (one of the product's own gallery URLs, for colourways) → **202 `{job_id}`**; the storefront polls `GET /api/shop/try-on/jobs/:id` every 2s (202 pending / 200 `{image: data-URL, remaining}` / mapped error status). Async because a generation takes **60–90s** (measured 2026-09-23 on both flash-image models), longer than nginx/Cloudflare keep a request open; jobs live in memory for 10 min. Daily quota in memory (Bangkok day): per IP for guests (`TRYON_DAILY_LIMIT`, default 5; X-Forwarded-For aware) or per customer when a member JWT is sent (`TRYON_MEMBER_DAILY_LIMIT`, default 20; `middleware.OptionalMemberID`); a failed generation refunds the try. Privacy: the customer photo is normalised in memory (`services.NormalizeJPEG`, 1280px) and never written to disk/DB; the privacy page says so. Prompt placement depends on category (ring → close-up of the hand wearing it, jewellery → crop to where it's worn, shoes, bag, else garment); the dialog's photo hint is category-aware too (rings: photograph your hand). **Garment cutouts:** Bruno product photos often show a model wearing the piece, which made the try-on dress the *model* instead of the customer (2026-09-23 bug). Fix: `TryOnClient.Cutout` turns each product image into an "item only on white" reference once (cached at `uploads/tryon_garment_<sha1(url)[:8]>.jpg`, generated lazily by `garmentImage`, pre-warmed for all products at startup by `WarmGarments`, serialised by `cutoutMu`; raw photo used as fallback if generation fails), and the try-on request interleaves text labels ("Image 1 — CUSTOMER…", "Image 2 — PRODUCT…") with the images and tells the model to ignore any person in image 2. `GET /api/shop/products/:id/garment?image=` serves the cutout (public, 1h cache) and is what the live try-on sends to Decart as its reference. Storefront `components/TryOn.tsx` (button under AddToBag on the product page; hidden for poster/booklet/gift categories) downscales the photo in-browser via canvas (bakes EXIF orientation), shows a 2-pane dialog (your photo / product image with gallery thumbs), result with save button, and the remaining-tries + member upsell.
- **Live webcam try-on (ลองใส่สด):** `services/decart.go` + `handlers/livetryon.go` + storefront `components/LiveTryOn.tsx` (button under the photo try-on, garment categories only — jewellery/bags/hats/shoes/print are excluded because Decart's model is a garment model). The browser streams the webcam straight to Decart's realtime `lucy-vton-latest` model over WebRTC via `@decartai/sdk` (lazy-imported); our backend only mints capped client tokens: `GET /api/shop/live-tryon` → `{enabled, remaining, limit, session_seconds, member}`; **Members only by default** (`LIVE_TRYON_MEMBERS_ONLY`, owner decision 2026-09-23 after a ฿486 test bill): guests see the button but the footer shows a sign-up link to `/member?next=<product path>` (the member page honours a same-site `?next=` after login/register) and the token endpoint answers 401 with `members_only: true`. `POST /api/shop/live-tryon/token?product_id=` consumes one session from the daily budget (members `LIVE_TRYON_MEMBER_SESSIONS`×`LIVE_TRYON_MEMBER_SECONDS`, default 4×20s; guests `LIVE_TRYON_DAILY_SESSIONS`×`LIVE_TRYON_SESSION_SECONDS`, only when members-only is off; refunded if Decart refuses), creates a `LiveTryOnSession` row (`models/tryon.go`) and returns `session_id` and calls `POST https://api.decart.ai/v1/client/tokens` (`x-api-key`, body `{expiresIn, allowedModels, allowedOrigins, constraints.realtime.maxSessionDuration}`) so **Decart itself cuts the stream at the cap** (billing is per streamed second, ~$0.02/s ≈ ฿40/min). Set `LIVE_TRYON_ALLOWED_ORIGINS=https://brunocollective.io` in prod so tokens can't be replayed elsewhere. Client: camera first (a declined permission spends no session), then token, then `client.realtime.connect(stream, {model, mirror:"auto", onRemoteStream, onConnectionChange, initialState:{prompt, image: garment Blob}})`; gallery thumbs call `set({prompt, image, enhance:false})` to swap colourways; prompt follows Decart's VTON guide ("Substitute the upper/lower body garment / outfit with the garment shown in the reference image…"); local countdown mirrors the cap, `sessionEnded` event ends the session, closing the dialog disconnects + stops camera tracks. **Cost guards in the browser:** the session also stops when the tab is hidden (`visibilitychange`) and when MediaPipe pose detection (`@mediapipe/tasks-vision`, lazy-loaded, wasm from jsDelivr, model from Google storage — same setup as Decart's person-detection example) sees no person for 3 consecutive 1s checks. **Cost log:** the SDK's `generationTick`/`generationEnded` seconds are reported once via `POST /api/shop/live-tryon/sessions/:id/end {seconds, reason}` (keepalive; reasons timeout/hidden/no_person/closed/server/error/unmount; clamped to cap+5s; unreported sessions count as the full cap) and `GET /api/live-tryon/usage?days=` (admin) returns per-day sessions/seconds/USD at $0.02/s plus totals — no admin UI yet. Decart wants clean garment images (item only, plain background, ≥512px) — multi-colour flat lays confuse it.
- **Storefront URLs / SEO:** products live at `/products/{slug}` (`Product.Slug`, ASCII-only, generated from the name by `handlers/slug.go` — `Slugify`, SKU/`product-{id}` fallbacks for Thai-only names, de-duplicated with `-2`, `-3`; owner can hand-edit in the admin form; backfilled at startup by `EnsureProductSlugs`). `GET /api/shop/products/:id` accepts a numeric id **or** a slug. Legacy `/product/{id}` and `/products/{id}` 308 to the slug URL. Categories get crawlable pages at `/collections/{slug}` via `lib/categories.ts` (Thai→English slug map — add a line when a new Thai category appears; Latin names slugify; unknown Thai falls back to percent-encoding); `/shop?cat=` still works but links use `collectionPath()`. URL builders live in `lib/paths.ts` (`productPath` falls back to `/product/{id}` for pre-slug cart lines). `lib/site.ts` holds `SITE_URL` (env `NEXT_PUBLIC_SITE_URL`, default brunocollective.io) used by `metadataBase`, the self-referencing canonical (`alternates.canonical: "./"` in the root layout), `app/robots.ts` (disallows cart/checkout/pay/member/affiliate/admin/api), `app/sitemap.ts` (force-dynamic: statics + collections + all products incl. sold-out + journal; sale pages deliberately excluded), and JSON-LD via `components/JsonLd.tsx` (Organization + WebSite in layout, Product + BreadcrumbList on product pages, BreadcrumbList on collections, Article on journal). **No AggregateRating in JSON-LD** — the stars are sales-derived, not reviews, and Google penalises fake review markup. cart/checkout/member have `layout.tsx` with `robots: noindex`.
- **Storefront star ratings:** customers don't review; `handlers/rating.go` sets `Product.Rating`/`RatingCount` (not persisted, `gorm:"-"`). Rating = the persisted `Product.RatingOverride` when set (admin product form "ดาวหน้าร้าน", nil = auto), otherwise derived from real units sold on non-cancelled orders — deterministic, always 4.5–5.0 (`4.5 + 0.5·u/(u+8)` + tiny per-id jitter, one decimal; ~8 units → 4.8, ~30 → 4.9, ~90 → 5.0). Count is **always** real units sold (override never fakes the count). Attached by the public shop handlers (`/api/shop/products`, `/:id`, `/suggest`) and the admin product list/get (so the form hint can show the current auto value). Storefront `Rating` component hides when rating = 0 (no sales, no override) and omits the "(n)" when count = 0 (override before first sale). Shown on `ProductCard` (under the price) and the product page (right of the price).
- **Cross-sell suggestions:** `GET /api/shop/products/suggest?ids=1,2` (public, `handlers/suggest.go`) ranks in-stock products by same-order co-occurrence with the given ids (same signal as the analytics bought-together pairs), topped up with overall best sellers when pair data is thin; input ids excluded. Used by the storefront checkout ("ซื้อคู่กันบ่อย" — variant products link to the product page, variant-less ones add straight to the bag with the drawer suppressed) and the ChatView order dialog (suggestion chips that append an item row). Registered before `/api/shop/products/:id` (Fiber literal-before-param).
- **Chat analytics:** `GET /api/analytics/chats?days=` (`handlers/analytics_chat.go`) powers the dashboard "แชท & ช่องทาง" tab: revenue/orders/AOV per channel, chat funnel (conversations with inbound → chat-attributed orders via `Order.ConversationID`), time-to-close buckets (first contact → first order, median), reply-speed buckets (per-conversation avg first-response time ≤10m/≤60m/≤24h/>24h) vs conversion + revenue, bot activity from `AutoReplyLog`, and an all-time pending-deals snapshot. All datetime math in Go (SQLite gotcha).
- **Notifications:** `GET /api/notifications` (handlers/notification.go) aggregates pending orders + low/out-of-stock (variant-aware) for the bell dropdown in `DefaultLayout.vue` (refetch on open + 90s poll).
- **Default admin:** On first run, seeds `admin` / `admin123`.
- **Frontend proxy:** Vite dev server proxies `/api` and `/uploads` to `http://localhost:8080`.
- **Handler dependency injection:** Handlers receive config and services via constructor (e.g., `NewOrderHandler(cfg, telegramNotifier)`).
- **Graceful degradation:** Telegram notifications are skipped silently when `TELEGRAM_BOT_TOKEN` or `TELEGRAM_CHAT_ID` are not set; LINE chat behaves the same when `LINE_CHANNEL_SECRET` / `LINE_CHANNEL_ACCESS_TOKEN` are not set (webhook 200s and ignores, replies return a Thai error). All notification calls use goroutines to avoid blocking the HTTP response.

## Frontend Patterns

- **Path alias:** `@` maps to `frontend/src/` in both Vite and TypeScript configs.
- **Vuetify theme:** White/neutral surfaces with brand accents — primary `#1A1714` (dark brown) and secondary `#C4A24D` (gold) are reserved for the sidebar, buttons, and active states; background `#F6F7F9`, text `#111827`/`#6B7280`, borders `#E5E7EB`. Status colors: success `#15803D`, warning `#B45309`, error `#DC2626`, info `#2A78D6`. Component defaults set globally in `plugins/vuetify.ts` (rounded cards, outlined inputs).
- **Chart colors:** Charts use a CVD-validated categorical palette in fixed order: `#2a78d6` (blue) → `#1baf7a` (mint) → `#eda100` (yellow) → `#008300` (green) → `#4a3aa7` (violet) → `#e34948` (red). Color follows the entity, never the rank: Facebook=blue, LINE=green, Instagram=violet, TikTok=black, storefront=yellow, sale-page=red; order statuses pending=amber/confirmed=blue/shipped=violet/delivered=green/cancelled=red. The owner finds low-chroma warm tones hard to distinguish — don't add new charts in the old brown/gold ramp.
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

Messages use HTML parse mode (`<b>` for bold). Images are sent via `sendPhoto` with the public URL. To get the chat ID, message the bot and check `https://api.telegram.org/bot<token>/getUpdates`.

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
| POST | `/api/products/:id/merge` | Yes | Merge duplicate product `:id` into `{target_id}` (orders, stock, sale pages move; duplicate deleted) |
| PUT | `/api/products/reorder` | Yes | Set storefront display order: `{ids: [...]}` → display_order 1..N (unordered products sort last) |
| GET/POST | `/api/customers` | Yes | List / Create customer |
| GET/PUT/DELETE | `/api/customers/:id` | Yes | Get / Update / Delete customer |
| GET/POST | `/api/orders` | Yes | List / Create order (supports multipart form with slip) |
| GET/DELETE | `/api/orders/:id` | Yes | Get / Delete order |
| PUT | `/api/orders/:id/status` | Yes | Update order status |
| POST | `/api/orders/:id/slip` | Yes | Upload payment slip |
| GET/POST | `/api/coupons` | Yes | List / Create coupon |
| POST | `/api/coupons/validate` | Yes | Preview a coupon against a cart (admin order form) |
| GET/PUT/DELETE | `/api/coupons/:id` | Yes | Get / Update / Delete coupon |
| POST | `/api/coupons/:id/toggle` | Yes | Pause/resume coupon |
| GET | `/api/coupons/:id/redemptions` | Yes | Coupon usage history |
| POST | `/api/shop/coupons/validate` | No | Storefront coupon preview (code + subtotal + phone) |
| POST | `/api/shop/members/register` | No | Member signup (name + phone + password), returns member JWT |
| POST | `/api/shop/members/login` | No | Member login by phone, returns member JWT |
| POST | `/api/shop/members/check` | No | Does this phone get the member discount? (boolean only) |
| GET/PUT | `/api/shop/members/me` | Member JWT | Get / Update member profile (PUT can change password) |
| GET | `/api/shop/members/me/orders` | Member JWT | Member's order history |
| POST | `/api/customers/:id/member` | Yes | Admin toggle customer membership |
| GET/POST | `/api/sale-pages` | Yes | List / Create sale page |
| POST | `/api/sale-pages/upload` | Yes | Upload a section image, returns `{url}` |
| GET/PUT/DELETE | `/api/sale-pages/:id` | Yes | Get / Update / Delete sale page |
| POST | `/api/sale-pages/:id/toggle` | Yes | Publish/unpublish |
| POST | `/api/sale-pages/:id/duplicate` | Yes | Clone page as draft (`-copy` slug) |
| GET | `/api/shop/sale-pages/:slug` | No | Public page data (counts a view; `?preview=1` shows drafts, no count) |
| POST | `/api/shop/sale-pages/:slug/order` | No | Place order from sale page (multipart, slip required, bump + coupon) |
| POST | `/api/daily-summary` | Yes | Manually trigger daily Telegram summary |
| GET | `/api/analytics/overview` | Yes | Period KPIs + prev-period compare, channel/category/coupon/sale-page splits (`?days=`) |
| GET | `/api/analytics/inventory` | Yes | Sell-through, stock aging, size curve, stock cover (`?category=`) |
| GET | `/api/analytics/customers` | Yes | RFM segments, repeat rate, new-vs-returning, top spenders |
| GET | `/api/analytics/products` | Yes | ABC classification + bought-together pairs |
| GET | `/api/analytics/chats` | Yes | Chat→sales analytics: channel split, reply speed vs conversion, time-to-close (`?days=`) |
| GET | `/api/shop/try-on` | No | Virtual try-on status: enabled + tries left today |
| POST | `/api/shop/try-on` | No | Start a try-on generation (multipart photo + product_id) → 202 `{job_id}` |
| GET | `/api/shop/try-on/jobs/:id` | No | Poll a try-on job (202 pending, 200 image, error status) |
| GET | `/api/shop/products/:id/garment` | No | Clean "item only" cutout of a product image (`?image=`), cached; reference for both try-ons |
| GET | `/api/shop/live-tryon` | No | Live webcam try-on status: enabled + sessions left + seconds per session |
| POST | `/api/shop/live-tryon/token` | Member JWT (default) | Consume one live session, returns a capped Decart client token + `session_id` |
| POST | `/api/shop/live-tryon/sessions/:id/end` | No | Browser reports generated seconds + end reason for cost tracking |
| GET | `/api/live-tryon/usage` | Yes | Per-day live try-on sessions / seconds / USD estimate (`?days=`) |
| GET | `/api/shop/products/suggest` | No | Cross-sell suggestions for `?ids=` (bought-together + best sellers, in stock) |
| GET | `/api/notifications` | Yes | Bell-menu alerts (pending orders, low/out-of-stock) |
| GET | `/api/chats` | Yes | Conversation list (most recent first) |
| GET | `/api/chats/summary` | Yes | Sidebar counters: waiting-for-reply + unread totals |
| GET | `/api/chats/followups` | Yes | ดีลค้าง: conversations with unpaid chat-created orders |
| POST | `/api/orders/:id/followup-discount` | Yes | Create + apply a single-use follow-up coupon to a pending order |
| GET | `/api/chats/:id/messages` | Yes | Conversation + messages oldest-first |
| POST | `/api/chats/:id/reply` | Yes | Push a text reply to the platform, then record it |
| POST | `/api/chats/:id/order` | Yes | Create order for the linked customer; returns `{order, pay_url}` |
| GET | `/api/pay/:token` | No | Public payment-page data (items + amounts only, no PII) |
| POST | `/api/pay/:token/slip` | No | Customer uploads payment slip by token (image only) |
| POST | `/api/chats/:id/read` | Yes | Zero the unread counter (+ Meta `mark_seen`) |
| POST | `/api/chats/:id/typing` | Yes | Forward a typing indicator to Messenger/IG (no-op for LINE) |
| PUT | `/api/chats/:id/status` | Yes | Flip thread between open and done |
| PUT | `/api/chats/:id/tags` | Yes | Replace thread labels |
| PUT | `/api/chats/:id/customer` | Yes | Link/unlink a customer record |
| POST | `/api/chats/:id/ai` | Yes | Toggle the AI assistant for one thread |
| POST | `/api/chats/:id/answered` | Yes | Clear waiting state for a reply made outside the system (LINE OA app) |
| POST | `/api/chats/answered` | Yes | Bulk clear waiting state: `{ids:[…]}` or `{all:true, platform?}` |
| GET | `/api/broadcasts` | Yes | Broadcast history (newest 50) |
| GET | `/api/broadcasts/audience` | Yes | LINE contacts bucketed by RFM segment |
| POST | `/api/broadcasts` | Yes | Send a LINE broadcast to selected segments (async) |
| GET/POST | `/api/canned-replies` | Yes | List / Create saved reply templates |
| PUT/DELETE | `/api/canned-replies/:id` | Yes | Update / Delete a template |
| GET/POST | `/api/auto-replies` | Yes | List / Create comment auto-reply rules |
| GET | `/api/auto-replies/logs` | Yes | Newest 100 engine runs |
| PUT/DELETE | `/api/auto-replies/:id` | Yes | Update / Delete a rule |
| POST | `/api/auto-replies/:id/toggle` | Yes | Enable/disable a rule |
| POST | `/api/webhooks/line` | Signature | LINE Messaging API webhook (HMAC-verified) |
| GET | `/api/webhooks/meta` | Verify token | Meta webhook subscribe handshake (hub.challenge) |
| POST | `/api/webhooks/meta` | Signature | FB Messenger/IG DM messages + FB/IG comment events (HMAC-verified) |
| GET | `/api/ws/chat` | JWT query | Admin realtime WebSocket (`?token=<JWT>`) |

## Environment Variables

All optional, with defaults:
- `PORT` — API port (default: `8080`)
- `DB_PATH` — SQLite file path (default: `inventory.db`)
- `JWT_SECRET` — HMAC signing key (default: `change-me-in-production`)
- `UPLOAD_DIR` — Slip upload directory (default: `./uploads`)
- `TELEGRAM_BOT_TOKEN` — Telegram Bot API token from @BotFather (default: empty, disables notifications)
- `TELEGRAM_CHAT_ID` — Telegram group/chat ID to send notifications to (default: empty, disables notifications)
- `BASE_URL` — Public base URL for image links in Telegram notifications (default: `http://localhost:8080`)
- `LINE_CHANNEL_SECRET` — LINE Messaging API channel secret, used to verify webhook signatures (default: empty, disables LINE chat)
- `LINE_CHANNEL_ACCESS_TOKEN` — LINE Messaging API long-lived access token for push replies/profile/media (default: empty, disables LINE chat)
- `META_APP_SECRET` — Meta app secret, verifies webhook signatures (default: empty, disables FB/IG chat)
- `META_VERIFY_TOKEN` — owner-chosen string echoed in the Meta webhook subscribe handshake (default: empty)
- `META_PAGE_ACCESS_TOKEN` — Facebook page access token; sends replies for the page and its linked Instagram account (default: empty, disables FB/IG chat)
- `CHAT_SLA_MINUTES` — Telegram alert when a chat waits this long for a reply (default: `10`, `0` disables)
- `CHAT_WAITING_EXPIRE_HOURS` — auto-drop a chat from the "รอตอบ" queue after waiting this long with no new inbound (assumed answered in the LINE app; default: `12`, `0` disables)
- `ANTHROPIC_API_KEY` — Claude API key for the AI chat assistant (default: empty, disables AI)
- `GEMINI_API_KEY` — Google AI Studio key for the virtual try-on (default: empty, hides the feature; project needs billing for image models)
- `TRYON_MEMBERS_ONLY` — only signed-in members may run the photo try-on (default: `true`)
- `TRYON_MODEL` — Gemini image model (default: `gemini-3.1-flash-image`; `gemini-3-pro-image` for higher quality at higher cost)
- `TRYON_DAILY_LIMIT` — try-on generations per guest IP per day (default: `5`, `0` = unlimited)
- `TRYON_MEMBER_DAILY_LIMIT` — per signed-in member per day (default: `20`, `0` = unlimited)
- `DECART_API_KEY` — Decart platform key (`dct_…`) for the live webcam try-on (default: empty, hides the feature)
- `LIVE_TRYON_MODEL` — Decart realtime model (default: `lucy-vton-latest`)
- `LIVE_TRYON_MEMBERS_ONLY` — only signed-in members may start a live session (default: `true`; `false` opens it to guests)
- `LIVE_TRYON_SESSION_SECONDS` / `LIVE_TRYON_MEMBER_SECONDS` — streaming cap per session, guest / member (default: `20` / `20`)
- `LIVE_TRYON_DAILY_SESSIONS` / `LIVE_TRYON_MEMBER_SESSIONS` — sessions per day per guest IP / per member (default: `2` / `4`, `0` = unlimited)
- `LIVE_TRYON_ALLOWED_ORIGINS` — comma-separated web origins the client tokens may be used from (default: empty = any; set to the public site in prod)
- `AI_MODEL` — Claude model for the AI assistant (default: `claude-opus-5`; e.g. `claude-haiku-4-5` for lower per-message cost)

## Deployment

Production runs on Vultr VPS (Ubuntu) with:
- **Nginx** reverse proxy serving frontend static files from `frontend/dist/` and proxying `/api/`, `/uploads/` to Go backend on port 8080
- **Cloudflare** DNS + SSL (Full Strict mode with Origin Certificate)
- **systemd** service (`inventory.service`) running the Go binary as `www-data`
- **EnvironmentFile** at `/opt/inventory/backend/.env` — values with special characters (like `/` or `=`) must be quoted
- **Deploy script** at `/opt/inventory/deploy.sh` — pulls from GitHub, rebuilds frontend + backend, restarts service
- **Backup cron** at `/opt/inventory/backup.sh` — daily at 2:00 AM, copies `inventory.db`, retains 30 days
