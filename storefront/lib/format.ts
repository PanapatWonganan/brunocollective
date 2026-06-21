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

// Image source resolver: backend stores relative /uploads paths; absolute URLs
// (e.g. an external CDN) are passed through untouched.
export function imageSrc(url: string): string {
  if (!url) return "";
  if (/^https?:\/\//.test(url)) return url;
  return url.startsWith("/") ? url : `/${url}`;
}
