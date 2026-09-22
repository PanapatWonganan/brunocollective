// Thai Baht money formatter — the storefront prices everything in THB.
const fmt = new Intl.NumberFormat("th-TH", {
  style: "currency",
  currency: "THB",
  minimumFractionDigits: 0,
  maximumFractionDigits: 0,
});

export function money(amount: number): string {
  return fmt.format(amount);
}

type Priced = { price: number; variants?: { price?: number }[] | null };

// Selling price of one line: the variant's own price when it has one (per-colour
// pricing), otherwise the product price. Tolerates old cart lines whose
// variant predates the price field.
export function unitPrice(
  product: { price: number },
  variant: { price?: number } | null | undefined
): number {
  const v = variant?.price;
  return v && v > 0 ? v : product.price;
}

// Lowest and highest selling price across a product's variants.
export function priceRange(product: Priced): { min: number; max: number } {
  let min = product.price;
  let max = product.price;
  for (const v of product.variants || []) {
    const u = unitPrice(product, v);
    if (u < min) min = u;
    if (u > max) max = u;
  }
  return { min, max };
}

// Card/list price: a single figure, or "฿590 – ฿790" when variants differ.
export function priceLabel(product: Priced): string {
  const { min, max } = priceRange(product);
  return min === max ? money(min) : `${money(min)} – ${money(max)}`;
}

// Image source resolver: backend stores relative /uploads paths; absolute URLs
// (e.g. an external CDN) are passed through untouched.
export function imageSrc(url: string): string {
  if (!url) return "";
  if (/^https?:\/\//.test(url)) return url;
  return url.startsWith("/") ? url : `/${url}`;
}
