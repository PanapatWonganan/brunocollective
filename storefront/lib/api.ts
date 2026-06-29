import type { Product, CheckoutPayload } from "./types";

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
