// Meta (Facebook) Pixel helpers. The pixel script is injected by
// components/MetaPixel.tsx; everything here is a no-op until it loads,
// so tracking can never break the shop.

export const META_PIXEL_ID = "2535256213646692";

type Fbq = (...args: unknown[]) => void;

declare global {
  interface Window {
    fbq?: Fbq;
  }
}

export function fbqTrack(event: string, params?: Record<string, unknown>) {
  if (typeof window === "undefined" || typeof window.fbq !== "function") return;
  if (params) {
    window.fbq("track", event, params);
  } else {
    window.fbq("track", event);
  }
}
