// Category URL slugs (/collections/{slug}).
//
// Categories are free-text in the admin (mostly Thai, since that's what the
// nav shows shoppers). URLs should be English, so known Thai names map to an
// English slug here; Latin names are slugified directly; anything unknown
// falls back to the percent-encoded name so the link still resolves.
// Add a line when a new Thai category appears in the admin.
const THAI_TO_SLUG: Record<string, string> = {
  "เสื้อยืด": "t-shirts",
  "เสื้อ": "tops",
  "เสื้อเชิ้ต": "shirts",
  "เสื้อโปโล": "polo-shirts",
  "เสื้อกล้าม": "tank-tops",
  "เสื้อกันหนาว": "sweaters",
  "แจ็คเก็ต": "jackets",
  "เสื้อคลุม": "outerwear",
  "กางเกง": "pants",
  "กางเกงขายาว": "trousers",
  "กางเกงขาสั้น": "shorts",
  "กางเกงยีนส์": "jeans",
  "เดรส": "dresses",
  "กระโปรง": "skirts",
  "ชุด": "sets",
  "รองเท้า": "shoes",
  "รองเท้าผ้าใบ": "sneakers",
  "กระเป๋า": "bags",
  "หมวก": "hats",
  "เข็มขัด": "belts",
  "ถุงเท้า": "socks",
  "เครื่องประดับ": "accessories",
  "ของตกแต่ง": "decor",
  "โปสเตอร์": "posters",
  "หนังสือ": "books",
};

export function slugify(s: string): string {
  return s
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

export function categorySlug(name: string): string {
  const n = name.trim();
  if (THAI_TO_SLUG[n]) return THAI_TO_SLUG[n];
  const ascii = slugify(n);
  return ascii || encodeURIComponent(n);
}

// Resolve a URL slug back to the live category name (case-insensitive, and
// tolerant of the encoded-Thai fallback).
export function categoryFromSlug(slug: string, categories: string[]): string | null {
  let decoded = slug;
  try {
    decoded = decodeURIComponent(slug);
  } catch {
    /* keep raw */
  }
  for (const c of categories) {
    if (categorySlug(c) === slug || c.trim() === decoded) return c;
  }
  return null;
}
