# แผนติดตั้ง Meta Conversions API (CAPI)

> สถานะ: **แผน — ยังไม่ได้ทำ** (เขียน 2026-09-04)
> เป้าหมาย: ส่ง Purchase จากฝั่ง server ให้ Meta ตรงๆ เพื่อเก็บ conversion ที่ browser pixel เก็บไม่ได้
> (ad blocker, iOS ATT, Safari ITP) พร้อม dedupe กับ pixel เดิม — signal ฝั่ง Purchase จะเข้าใกล้ 100%

## ภาพรวม

ตอนนี้ browser pixel (id `2535256213646692`, โค้ดที่ `storefront/lib/fbq.ts` + `components/MetaPixel.tsx`)
ยิง Purchase จาก 3 จุด: checkout, sale page, /pay (สลิปแรก) — CAPI คือการยิง event เดียวกันซ้ำจาก Go backend
โดยใส่ `event_id` ตรงกันทั้งสองฝั่ง ให้ Meta dedupe เอง ใครโดนบล็อกฝั่ง browser ก็ยังนับจากฝั่ง server ได้

```
Browser  ── fbq('track','Purchase', {...}, {eventID: 'order-123'}) ──▶ Meta
Backend  ── POST /v23.0/{pixel}/events  event_id: 'order-123'      ──▶ Meta   → นับครั้งเดียว
```

## Step 1 — Access token (ทำใน Events Manager, ไม่ใช่โค้ด)

Events Manager → Data Sources → เลือก pixel → Settings → Conversions API → **Generate access token**
(ใช้ system user เดิมจากตอนตั้ง Meta webhook ได้ — ดู memory `meta-fb-setup`)

Env ใหม่ 2 ตัว (pattern เดียวกับ Telegram: ว่าง = ปิดเงียบๆ):

```
META_PIXEL_ID=2535256213646692
META_CAPI_ACCESS_TOKEN=EAAxxxx...
```

- เพิ่มใน `backend/config/config.go` (`MetaPixelID`, `MetaCAPIToken`)
- บน VPS: `/opt/inventory/backend/.env` — **อย่าลืม quote ค่า** (กฎ EnvironmentFile เดิม) แล้ว `systemctl restart inventory.service`

## Step 2 — `backend/services/metacapi.go` (ไฟล์ใหม่)

Struct + constructor ตามแบบ `services/telegram.go` (รับ config, มี `enabled()` เช็ค token ว่าง)

ยิง `POST https://graph.facebook.com/v23.0/{PIXEL_ID}/events?access_token={TOKEN}` body:

```json
{
  "data": [{
    "event_name": "Purchase",
    "event_time": 1757000000,
    "event_id": "order-123",
    "action_source": "website",
    "event_source_url": "https://brunocollective.io/checkout",
    "user_data": {
      "ph": ["<sha256 ของเบอร์ normalize แล้ว>"],
      "em": ["<sha256 ของอีเมล lowercase>"],
      "client_ip_address": "...",
      "client_user_agent": "...",
      "fbp": "_fbp cookie ถ้ามี",
      "fbc": "_fbc cookie ถ้ามี"
    },
    "custom_data": {
      "currency": "THB",
      "value": 1290.00,
      "content_ids": ["5", "12"],
      "content_type": "product",
      "num_items": 2
    }
  }],
  "test_event_code": "ใส่เฉพาะตอนเทสต์ แล้วลบออก"
}
```

กติกา user_data (สำคัญต่อ match quality / EMQ):
- **ph**: ตัดทุกอย่างที่ไม่ใช่ตัวเลข, เบอร์ไทย `08x...` → `668x...` (แทน 0 นำหน้าด้วย 66), แล้ว SHA-256 hex
- **em**: trim + lowercase แล้ว SHA-256 hex (ลูกค้าส่วนใหญ่ไม่มีอีเมล — ข้ามได้ถ้าว่าง)
- **fbp/fbc/ip/user_agent**: ส่งดิบ **ห้าม hash** — อ่านจาก HTTP request ได้เลย (ดู Step 4)
- ยิงใน goroutine เหมือน Telegram — ห้าม block HTTP response; log error พอ ไม่ต้อง retry ซับซ้อน

## Step 3 — จุดที่ยิงฝั่ง backend (event_id = `order-{orderID}` ทุกจุด)

| จุด | ไฟล์:บรรทัด (อ้างอิง ณ วันเขียนแผน) | เงื่อนไข |
|---|---|---|
| Storefront checkout | `handlers/shop.go:233` (ข้าง `NotifyNewOrder`) | ยิงเลย — สลิปแนบมาตอนสร้างออเดอร์ |
| Sale page order | `handlers/salepage.go:479` | ยิงเลย — สลิปแนบมาแล้วเช่นกัน |
| /pay slip upload | `handlers/pay.go:377` (ข้าง `NotifySlipUploaded`) | ยิงเฉพาะ**สลิปแรก** (ก่อนหน้านี้ `SlipImage == ""`) — กันอัปโหลดซ้ำนับซ้ำ ตรงกับ logic ฝั่ง browser ใน `PayClient.tsx` |
| สร้างออเดอร์จากแชท | `handlers/pay.go:136` | **ไม่ยิง** — ยังไม่จ่าย จะไปนับตอนสลิปเข้าที่ /pay |
| Admin สร้างออเดอร์ / อัปสลิปในแอดมิน | `handlers/order.go:283,436` | **ไม่ยิง** (ไม่ใช่ conversion จากเว็บ) — ถ้าอยากนับยอดแชทที่ลูกค้าส่งสลิปในแชทแทน /pay ค่อยเพิ่มทีหลังด้วย `action_source: "chat"` |

หมายเหตุ: `event_source_url` ของ /pay ให้ใช้ `{BASE_URL}/pay` เฉยๆ **อย่าใส่ token** ในลิงก์ที่ส่งให้ Meta

## Step 4 — fbp/fbc/IP/UA จาก request

ออเดอร์ทั้ง 3 จุดถูก POST มาจากเบราว์เซอร์ลูกค้าแบบ same-origin (brunocollective.io) → cookie `_fbp`/`_fbc`
ติดมากับ request อยู่แล้ว อ่านใน handler ได้เลย:

- `c.Cookies("_fbp")`, `c.Cookies("_fbc")`
- IP: nginx อยู่หลัง Cloudflare — ใช้ `CF-Connecting-IP` ก่อน, fallback `X-Forwarded-For` ตัวแรก (เช็คใน `deploy/nginx.conf` ว่า proxy header ส่งครบ)
- UA: `c.Get("User-Agent")`

ส่งเป็น struct/param เข้า service ตอนเรียก (handler เป็นคนอ่าน เพราะ service ไม่เห็น fiber.Ctx)

## Step 5 — ฝั่ง storefront: ผูก event_id ให้ dedupe ได้

1. `lib/fbq.ts` — เพิ่ม param ที่ 3:
   ```ts
   export function fbqTrack(event: string, params?: Record<string, unknown>, eventId?: string) {
     ...
     window.fbq("track", event, params ?? {}, eventId ? { eventID: eventId } : undefined);
   }
   ```
2. อัปเดตจุดยิง Purchase 3 จุดให้ส่ง `` `order-${orderId}` ``:
   - `app/checkout/page.tsx` (ใน onSubmit หลัง res.ok — มี `res.orderId` แล้ว ต้องย้าย fbqTrack ให้ใช้ id จริง)
   - `app/s/[slug]/SalePageClient.tsx` (ใช้ `res.orderId`)
   - `app/pay/[token]/PayClient.tsx` (ใช้ `res.order.order_no`)
3. PageView/AddToCart/InitiateCheckout ไม่ต้องมี eventID (ไม่ได้ยิงจาก server)

## Step 6 — เทสต์

1. ใส่ `test_event_code` (จากแท็บ Test Events) ใน payload ชั่วคราว หรือทำเป็น env `META_CAPI_TEST_CODE`
2. สั่งซื้อจริง 1 รายการ (เว็บ + /pay) → ใน Test Events ต้องเห็น event **2 แหล่ง (Browser + Server)
   ถูก dedupe เป็นอันเดียว** (Meta โชว์ "Deduplicated")
3. ลองปิด JS / เปิด ad blocker แล้วสั่งซื้อ → ต้องเห็นเฉพาะฝั่ง Server เข้า
4. เอา test code ออก, deploy, ผ่านไป 1-2 วันเช็ค **Event Match Quality (EMQ)** ของ Purchase ใน
   Events Manager — เป้า 6.0+ (เบอร์โทร hash ช่วยตรงนี้เยอะ)

## เช็คลิสต์ deploy

- [ ] Generate CAPI token + ใส่ .env บน VPS (quote ค่า!)
- [ ] config.go + services/metacapi.go + hook 3 จุด + frontend eventID
- [ ] `go build` / `npx tsc --noEmit` / `npm run build` ผ่าน
- [ ] deploy.sh → เทสต์ตาม Step 6 → เอา test code ออก → deploy อีกรอบ
- [ ] อัปเดต CLAUDE.md (env ใหม่ 2 ตัว + pattern CAPI) และ memory

## สิ่งที่**ไม่**ต้องทำ

- ไม่มี "Andromeda API" — Andromeda คือระบบ retrieval ภายในของ Meta ฝั่งเราแค่ป้อน signal ดีๆ
  ผ่าน CAPI + Advanced Matching ตามแผนนี้ก็ได้ประโยชน์จากมันแล้ว
- ไม่ต้องส่ง PageView/ViewContent จาก server — ได้ไม่คุ้มความซับซ้อน มาตรฐานทั่วไปทำแค่
  Purchase (บาง shop เพิ่ม InitiateCheckout) ก็พอ
