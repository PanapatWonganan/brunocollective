// Public site origin used for canonical URLs, sitemap, robots and JSON-LD.
// Override with NEXT_PUBLIC_SITE_URL when running a staging copy.
export const SITE_URL = (
  process.env.NEXT_PUBLIC_SITE_URL || "https://brunocollective.io"
).replace(/\/$/, "");

export const SITE_NAME = "Bruno Collective";

export function absoluteUrl(path: string): string {
  return `${SITE_URL}${path.startsWith("/") ? path : `/${path}`}`;
}
