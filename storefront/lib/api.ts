import type {
  Product,
  CheckoutPayload,
  CouponPreview,
  SalePage,
  MemberProfile,
  MemberOrder,
} from "./types";

// On the server we hit the backend directly; in the browser we use same-origin
// paths that Next rewrites (see next.config.ts) to the backend.
const BASE =
  typeof window === "undefined"
    ? process.env.BACKEND_URL || "http://localhost:8080"
    : "";

export async function getProducts(opts?: {
  includeOut?: boolean;
  search?: string;
}): Promise<Product[]> {
  const params = new URLSearchParams();
  if (opts?.includeOut) params.set("include_out", "1");
  if (opts?.search) params.set("search", opts.search);
  const qs = params.toString();

  const res = await fetch(`${BASE}/api/shop/products${qs ? `?${qs}` : ""}`, {
    // Storefront catalogue can be lightly cached; revalidate often so stock
    // changes from the admin show up quickly.
    next: { revalidate: 30 },
  });
  if (!res.ok) throw new Error("Failed to load products");
  return res.json();
}

export interface SiteImage {
  key: string;
  image_url: string;
  caption_a: string;
  caption_b: string;
}

// Pick the inline size chart for a product by its category: รองเท้า (shoes)
// and แหวน/เครื่องประดับ (rings, jewellery) get their own charts, every other
// sized category uses the shirt chart. Returns "" when the matching chart
// hasn't been uploaded (chart hidden).
export function sizeChartKey(category: string | undefined): string {
  const cat = (category || "").toLowerCase();
  if (cat.includes("รองเท้า") || cat.includes("shoe")) return "size_chart_shoes";
  if (
    cat.includes("แหวน") ||
    cat.includes("เครื่องประดับ") ||
    cat.includes("จิวเวอ") ||
    cat.includes("ring") ||
    cat.includes("jewel")
  ) {
    return "size_chart_rings";
  }
  return "size_chart";
}

export function sizeChartFor(
  category: string | undefined,
  site: Record<string, SiteImage>
): string {
  return site[sizeChartKey(category)]?.image_url || "";
}

// Editable storefront images keyed by slot (hero, lookbook_1…6, journal_1…3).
// Only customised slots are returned; callers fall back to built-in defaults.
export async function getSiteImages(): Promise<Record<string, SiteImage>> {
  try {
    const res = await fetch(`${BASE}/api/shop/site-images`, {
      next: { revalidate: 30 },
    });
    if (!res.ok) return {};
    return res.json();
  } catch {
    return {};
  }
}

export async function getProduct(id: number | string): Promise<Product | null> {
  const res = await fetch(`${BASE}/api/shop/products/${id}`, {
    next: { revalidate: 30 },
  });
  if (res.status === 404) return null;
  if (!res.ok) throw new Error("Failed to load product");
  return res.json();
}

// Best sellers for the home page — the suggest endpoint with no ids returns
// overall best sellers, topped up with the catalogue display order when the
// shop is young. In-stock only. Server-side (lightly cached).
export async function getBestSellers(limit = 4): Promise<Product[]> {
  try {
    const res = await fetch(`${BASE}/api/shop/products/suggest?limit=${limit}`, {
      next: { revalidate: 60 },
    });
    if (!res.ok) return [];
    return res.json();
  } catch {
    return [];
  }
}

// Server-side cross-sell for the product page ("You may also consider").
export async function getRelated(id: number | string, limit = 4): Promise<Product[]> {
  try {
    const res = await fetch(
      `${BASE}/api/shop/products/suggest?ids=${id}&limit=${limit}`,
      { next: { revalidate: 60 } }
    );
    if (!res.ok) return [];
    return res.json();
  } catch {
    return [];
  }
}

// Cross-sell: products most often bought together with the given ones (falls
// back to best sellers). In-stock only; the given ids are excluded.
export async function getSuggestions(ids: number[]): Promise<Product[]> {
  if (!ids.length) return [];
  try {
    const res = await fetch(
      `/api/shop/products/suggest?ids=${ids.join(",")}`,
      { cache: "no-store" }
    );
    if (!res.ok) return [];
    return res.json();
  } catch {
    return [];
  }
}

export interface CheckoutResult {
  ok: boolean;
  orderId?: number;
  error?: string;
}

// Client-side checkout — posts multipart/form-data to the public order endpoint
// so the payment slip can be attached. The slip is required.
export async function checkout(
  payload: CheckoutPayload,
  slip: File
): Promise<CheckoutResult> {
  const form = new FormData();
  form.set("name", payload.name);
  form.set("phone", payload.phone);
  form.set("address", payload.address);
  if (payload.email) form.set("email", payload.email);
  if (payload.notes) form.set("notes", payload.notes);
  if (payload.coupon_code) form.set("coupon_code", payload.coupon_code);
  if (payload.affiliate_code) form.set("affiliate_code", payload.affiliate_code);
  form.set("items", JSON.stringify(payload.items));
  form.set("slip", slip);

  // Note: do NOT set Content-Type — the browser sets the multipart boundary.
  const res = await fetch(`/api/shop/orders`, { method: "POST", body: form });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    return { ok: false, error: data?.error || "Order failed" };
  }
  return { ok: true, orderId: data?.id };
}

// Sale/landing page for /s/{slug}. Fetched fresh every request — countdowns
// and draft previews must not be cached. Preview mode shows drafts and skips
// the view counter.
export async function getSalePage(
  slug: string,
  preview = false
): Promise<SalePage | null> {
  const res = await fetch(
    `${BASE}/api/shop/sale-pages/${encodeURIComponent(slug)}${preview ? "?preview=1" : ""}`,
    { cache: "no-store" }
  );
  if (!res.ok) return null;
  return res.json();
}

export interface SalePageOrderPayload {
  name: string;
  phone: string;
  email?: string;
  address: string;
  notes?: string;
  quantity: number;
  variantId: number | null;
  bump: boolean;
  bumpVariantId: number | null;
  couponCode?: string;
  affiliateCode?: string;
}

// Places an order from a sale page. Multipart so the payment slip attaches;
// the backend prices the items from the page config, never from the client.
export async function salePageOrder(
  slug: string,
  payload: SalePageOrderPayload,
  slip: File
): Promise<CheckoutResult> {
  const form = new FormData();
  form.set("name", payload.name);
  form.set("phone", payload.phone);
  form.set("address", payload.address);
  if (payload.email) form.set("email", payload.email);
  if (payload.notes) form.set("notes", payload.notes);
  form.set("quantity", String(payload.quantity));
  if (payload.variantId) form.set("variant_id", String(payload.variantId));
  if (payload.bump) form.set("bump", "1");
  if (payload.bumpVariantId) form.set("bump_variant_id", String(payload.bumpVariantId));
  if (payload.couponCode) form.set("coupon_code", payload.couponCode);
  if (payload.affiliateCode) form.set("affiliate_code", payload.affiliateCode);
  form.set("slip", slip);

  const res = await fetch(`/api/shop/sale-pages/${encodeURIComponent(slug)}/order`, {
    method: "POST",
    body: form,
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    return { ok: false, error: data?.error || "สั่งซื้อไม่สำเร็จ กรุณาลองใหม่" };
  }
  return { ok: true, orderId: data?.id };
}

// ---- Public payment page (/pay/{token}) ----
// Orders created from the chat inbox carry an unguessable payment token; the
// customer opens /pay/{token} to see the summary and upload a slip. The
// backend response deliberately excludes phone/address.

export interface PayOrderItem {
  product_id: number;
  name: string;
  size: string;
  color: string;
  quantity: number;
  price: number;
}

export interface PayOrder {
  order_no: number;
  status: string;
  created_at: string;
  customer_name: string;
  items: PayOrderItem[];
  subtotal: number;
  member_discount: number;
  discount_amount: number;
  coupon_code: string;
  total_amount: number;
  has_slip: boolean;
}

export async function getPayOrder(token: string): Promise<PayOrder | null> {
  try {
    const res = await fetch(`${BASE}/api/pay/${encodeURIComponent(token)}`, {
      cache: "no-store",
    });
    if (!res.ok) return null;
    return res.json();
  } catch {
    return null;
  }
}

export async function uploadPaySlip(
  token: string,
  slip: File
): Promise<{ ok: boolean; order?: PayOrder; error?: string }> {
  const form = new FormData();
  form.set("slip", slip);
  try {
    const res = await fetch(`/api/pay/${encodeURIComponent(token)}/slip`, {
      method: "POST",
      body: form,
    });
    const data = await res.json().catch(() => ({}));
    if (!res.ok) {
      return { ok: false, error: data?.error || "อัปโหลดสลิปไม่สำเร็จ กรุณาลองใหม่" };
    }
    return { ok: true, order: data };
  } catch {
    return { ok: false, error: "อัปโหลดสลิปไม่สำเร็จ กรุณาลองใหม่" };
  }
}

// ---- Membership ----
// Members log in by phone; the token lives in localStorage (see lib/member.tsx)
// and unlocks a flat 5% discount that the backend applies server-side.

export const MEMBER_TOKEN_KEY = "bc_member_token";

export function getMemberToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(MEMBER_TOKEN_KEY);
}

export interface MemberAuthResult {
  ok: boolean;
  member?: MemberProfile;
  token?: string;
  error?: string;
}

async function memberAuthRequest(
  path: string,
  body: Record<string, string>
): Promise<MemberAuthResult> {
  try {
    const res = await fetch(path, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    const data = await res.json().catch(() => ({}));
    if (!res.ok) {
      return { ok: false, error: data?.error || "ไม่สำเร็จ กรุณาลองใหม่" };
    }
    return { ok: true, member: data.member, token: data.token };
  } catch {
    return { ok: false, error: "ไม่สำเร็จ กรุณาลองใหม่" };
  }
}

export function memberRegister(payload: {
  name: string;
  phone: string;
  email?: string;
  password: string;
}): Promise<MemberAuthResult> {
  return memberAuthRequest("/api/shop/members/register", {
    name: payload.name,
    phone: payload.phone,
    email: payload.email || "",
    password: payload.password,
  });
}

export function memberLogin(phone: string, password: string): Promise<MemberAuthResult> {
  return memberAuthRequest("/api/shop/members/login", { phone, password });
}

// Fetches the logged-in member's profile; null when the token is missing,
// expired, or rejected (callers should then clear the stored token).
export async function memberMe(): Promise<MemberProfile | null> {
  const token = getMemberToken();
  if (!token) return null;
  try {
    const res = await fetch("/api/shop/members/me", {
      headers: { Authorization: `Bearer ${token}` },
    });
    if (!res.ok) return null;
    return res.json();
  } catch {
    return null;
  }
}

export async function memberUpdate(payload: {
  name: string;
  email: string;
  address: string;
  current_password?: string;
  new_password?: string;
}): Promise<MemberAuthResult> {
  const token = getMemberToken();
  if (!token) return { ok: false, error: "กรุณาเข้าสู่ระบบใหม่" };
  try {
    const res = await fetch("/api/shop/members/me", {
      method: "PUT",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify(payload),
    });
    const data = await res.json().catch(() => ({}));
    if (!res.ok) {
      return { ok: false, error: data?.error || "บันทึกไม่สำเร็จ กรุณาลองใหม่" };
    }
    return { ok: true, member: data };
  } catch {
    return { ok: false, error: "บันทึกไม่สำเร็จ กรุณาลองใหม่" };
  }
}

export async function memberOrders(): Promise<MemberOrder[]> {
  const token = getMemberToken();
  if (!token) return [];
  try {
    const res = await fetch("/api/shop/members/me/orders", {
      headers: { Authorization: `Bearer ${token}` },
    });
    if (!res.ok) return [];
    return res.json();
  } catch {
    return [];
  }
}

// Asks whether a phone number qualifies for the member discount — used by the
// checkout page for shoppers who aren't logged in (returning customers get
// the discount automatically by phone match).
export async function memberCheck(
  phone: string
): Promise<{ is_member: boolean; discount_percent: number }> {
  try {
    const res = await fetch("/api/shop/members/check", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ phone }),
    });
    if (!res.ok) return { is_member: false, discount_percent: 0 };
    return res.json();
  } catch {
    return { is_member: false, discount_percent: 0 };
  }
}

export interface CouponCheckResult {
  ok: boolean;
  coupon?: CouponPreview;
  error?: string;
}

// Previews a coupon against the current cart subtotal. The backend rechecks
// everything at checkout, so this is purely for immediate shopper feedback.
// Errors come back in Thai, written for shoppers — show them as-is.
export async function validateCoupon(
  code: string,
  subtotal: number,
  phone?: string
): Promise<CouponCheckResult> {
  try {
    const res = await fetch(`/api/shop/coupons/validate`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ code, subtotal, phone: phone || "" }),
    });
    const data = await res.json().catch(() => ({}));
    if (!res.ok) {
      return { ok: false, error: data?.error || "ใช้โค้ดไม่สำเร็จ กรุณาลองใหม่" };
    }
    return { ok: true, coupon: data };
  } catch {
    return { ok: false, error: "ใช้โค้ดไม่สำเร็จ กรุณาลองใหม่" };
  }
}

// ---- Affiliate program ----
// Referrers share ?ref=CODE links (see lib/affiliate.ts) and log in to their
// own portal at /affiliate. role:"affiliate" tokens are isolated from both
// the admin and member APIs.

export const AFFILIATE_TOKEN_KEY = "bc_affiliate_token";

export function getAffiliateToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(AFFILIATE_TOKEN_KEY);
}

export interface AffiliateProfile {
  id: number;
  code: string;
  name: string;
  phone: string;
  email: string;
  commission_percent: number;
}

export interface AffiliateStats {
  affiliate: AffiliateProfile;
  clicks: number;
  pending_amount: number;
  confirmed_amount: number;
  paid_amount: number;
  cancelled_amount: number;
  orders_count: number;
}

export interface AffiliateOrderRow {
  order_id: number;
  created_at: string;
  order_status: string;
  item_count: number;
  order_total: number;
  commission: number;
  commission_status: string;
}

// Pre-checkout validation for a typed referral code. Thai errors, shown as-is.
export async function validateAffiliate(
  code: string
): Promise<{ ok: boolean; name?: string; code?: string; error?: string }> {
  try {
    const res = await fetch("/api/shop/affiliates/validate", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ code }),
    });
    const data = await res.json().catch(() => ({}));
    if (!res.ok) {
      return { ok: false, error: data?.error || "ตรวจสอบรหัสไม่สำเร็จ กรุณาลองใหม่" };
    }
    return { ok: true, name: data.name, code: data.code };
  } catch {
    return { ok: false, error: "ตรวจสอบรหัสไม่สำเร็จ กรุณาลองใหม่" };
  }
}

export async function affiliateLogin(
  phone: string,
  password: string
): Promise<{ ok: boolean; token?: string; affiliate?: AffiliateProfile; error?: string }> {
  try {
    const res = await fetch("/api/shop/affiliates/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ phone, password }),
    });
    const data = await res.json().catch(() => ({}));
    if (!res.ok) {
      return { ok: false, error: data?.error || "เข้าสู่ระบบไม่สำเร็จ กรุณาลองใหม่" };
    }
    return { ok: true, token: data.token, affiliate: data.affiliate };
  } catch {
    return { ok: false, error: "เข้าสู่ระบบไม่สำเร็จ กรุณาลองใหม่" };
  }
}

// Profile + real-time stats; null when the token is missing/expired.
export async function affiliateMe(): Promise<AffiliateStats | null> {
  const token = getAffiliateToken();
  if (!token) return null;
  try {
    const res = await fetch("/api/shop/affiliates/me", {
      headers: { Authorization: `Bearer ${token}` },
      cache: "no-store",
    });
    if (!res.ok) return null;
    return res.json();
  } catch {
    return null;
  }
}

export async function affiliateOrders(): Promise<AffiliateOrderRow[]> {
  const token = getAffiliateToken();
  if (!token) return [];
  try {
    const res = await fetch("/api/shop/affiliates/me/orders", {
      headers: { Authorization: `Bearer ${token}` },
      cache: "no-store",
    });
    if (!res.ok) return [];
    return (await res.json()) || [];
  } catch {
    return [];
  }
}

export async function affiliateChangePassword(
  currentPassword: string,
  newPassword: string
): Promise<{ ok: boolean; error?: string }> {
  const token = getAffiliateToken();
  if (!token) return { ok: false, error: "กรุณาเข้าสู่ระบบใหม่" };
  try {
    const res = await fetch("/api/shop/affiliates/me", {
      method: "PUT",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({ current_password: currentPassword, new_password: newPassword }),
    });
    const data = await res.json().catch(() => ({}));
    if (!res.ok) {
      return { ok: false, error: data?.error || "บันทึกไม่สำเร็จ กรุณาลองใหม่" };
    }
    return { ok: true };
  } catch {
    return { ok: false, error: "บันทึกไม่สำเร็จ กรุณาลองใหม่" };
  }
}

// ── Virtual try-on (ลองใส่ดู) ─────────────────────────────────────────────
// Backed by a Gemini image model; hidden when the backend has no key.

export interface TryOnStatus {
  enabled: boolean;
  members_only?: boolean;
  remaining?: number;
  limit?: number;
  member?: boolean;
}

export async function getTryOnStatus(): Promise<TryOnStatus> {
  try {
    const token = getMemberToken();
    const res = await fetch("/api/shop/try-on", {
      cache: "no-store",
      headers: token ? { Authorization: `Bearer ${token}` } : undefined,
    });
    if (!res.ok) return { enabled: false };
    return res.json();
  } catch {
    return { enabled: false };
  }
}

export interface TryOnResult {
  ok: boolean;
  image?: string; // data: URL
  remaining?: number;
  error?: string;
}

// photo: a JPEG blob already downscaled by the caller (see TryOn component).
// Generation takes ~1 minute, so the backend hands back a job id and we poll
// it every 2s; onTick reports elapsed seconds for the UI.
export async function generateTryOn(
  productId: number,
  photo: Blob,
  image?: string,
  onTick?: (elapsedSec: number) => void
): Promise<TryOnResult> {
  const fd = new FormData();
  fd.append("product_id", String(productId));
  fd.append("photo", photo, "photo.jpg");
  if (image) fd.append("image", image);
  const token = getMemberToken();
  const headers = token ? { Authorization: `Bearer ${token}` } : undefined;
  let jobId: string;
  try {
    const res = await fetch("/api/shop/try-on", { method: "POST", body: fd, headers });
    const data = await res.json().catch(() => ({}));
    if (!res.ok || !data.job_id) {
      return { ok: false, error: data.error || "ระบบลองใส่ขัดข้อง กรุณาลองใหม่", remaining: data.remaining };
    }
    jobId = data.job_id;
  } catch {
    return { ok: false, error: "เชื่อมต่อไม่ได้ กรุณาลองใหม่" };
  }

  const started = Date.now();
  const deadline = started + 4 * 60 * 1000;
  let misses = 0;
  while (Date.now() < deadline) {
    await new Promise((r) => setTimeout(r, 2000));
    onTick?.(Math.round((Date.now() - started) / 1000));
    try {
      const res = await fetch(`/api/shop/try-on/jobs/${jobId}`, { cache: "no-store" });
      const data = await res.json().catch(() => ({}));
      if (res.status === 202) continue;
      if (res.ok && data.image) return { ok: true, image: data.image, remaining: data.remaining };
      return { ok: false, error: data.error || "ระบบลองใส่ขัดข้อง กรุณาลองใหม่", remaining: data.remaining };
    } catch {
      // Transient network blip while polling — tolerate a few before giving up.
      if (++misses >= 5) return { ok: false, error: "เชื่อมต่อไม่ได้ กรุณาลองใหม่" };
    }
  }
  return { ok: false, error: "ใช้เวลานานผิดปกติ กรุณาลองใหม่อีกครั้ง" };
}

// ── Live webcam try-on (ลองใส่สด, Decart realtime) ─────────────────────────

export interface LiveTryOnStatus {
  enabled: boolean;
  members_only?: boolean;
  remaining?: number;
  limit?: number;
  session_seconds?: number;
  member?: boolean;
  model?: string;
}

export async function getLiveTryOnStatus(): Promise<LiveTryOnStatus> {
  try {
    const token = getMemberToken();
    const res = await fetch("/api/shop/live-tryon", {
      cache: "no-store",
      headers: token ? { Authorization: `Bearer ${token}` } : undefined,
    });
    if (!res.ok) return { enabled: false };
    return res.json();
  } catch {
    return { enabled: false };
  }
}

export interface LiveTryOnToken {
  ok: boolean;
  api_key?: string;
  model?: string;
  session_seconds?: number;
  remaining?: number;
  session_id?: number;
  members_only?: boolean;
  error?: string;
}

// Report the seconds the SDK says were generated (cost tracking). Uses
// keepalive so it survives the tab closing.
export function endLiveTryOnSession(sessionId: number, seconds: number, reason: string): void {
  const token = getMemberToken();
  try {
    fetch(`/api/shop/live-tryon/sessions/${sessionId}/end`, {
      method: "POST",
      keepalive: true,
      headers: {
        "Content-Type": "application/json",
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
      },
      body: JSON.stringify({ seconds, reason }),
    }).catch(() => {});
  } catch {
    /* ignore */
  }
}

// One session = one short-lived Decart client token, counted against the
// caller's daily budget server-side.
export async function createLiveTryOnToken(productId: number): Promise<LiveTryOnToken> {
  const token = getMemberToken();
  try {
    const res = await fetch(`/api/shop/live-tryon/token?product_id=${productId}`, {
      method: "POST",
      headers: token ? { Authorization: `Bearer ${token}` } : undefined,
    });
    const data = await res.json().catch(() => ({}));
    if (!res.ok) {
      return { ok: false, error: data.error || "ระบบลองใส่สดขัดข้อง กรุณาลองใหม่", remaining: data.remaining, members_only: data.members_only };
    }
    return { ok: true, ...data };
  } catch {
    return { ok: false, error: "เชื่อมต่อไม่ได้ กรุณาลองใหม่" };
  }
}
