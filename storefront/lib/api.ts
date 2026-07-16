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
