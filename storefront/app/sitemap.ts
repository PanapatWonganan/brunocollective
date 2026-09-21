import type { MetadataRoute } from "next";
import { getProducts } from "@/lib/api";
import { ESSAYS } from "@/lib/journal";
import { categorySlug } from "@/lib/categories";
import { productPath } from "@/lib/paths";
import { absoluteUrl } from "@/lib/site";
import type { Product } from "@/lib/types";

// Always rendered on request so a build with the backend down never bakes in
// an empty product list; getProducts() itself caches for 30s.
export const dynamic = "force-dynamic";

// Sitemap: static pages, every product (incl. sold-out — the page still
// ranks and says "sold out"), one URL per category, and the journal essays.
// Sale pages (/s/…) are deliberately left out: they're ad landing pages and
// would compete with the product pages for the same queries.
export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  let products: Product[] = [];
  try {
    products = await getProducts({ includeOut: true });
  } catch {
    products = [];
  }

  const statics: MetadataRoute.Sitemap = [
    { url: absoluteUrl("/"), changeFrequency: "weekly", priority: 1 },
    { url: absoluteUrl("/shop"), changeFrequency: "weekly", priority: 0.9 },
    { url: absoluteUrl("/story"), changeFrequency: "monthly", priority: 0.6 },
    { url: absoluteUrl("/lookbook"), changeFrequency: "monthly", priority: 0.5 },
    { url: absoluteUrl("/service"), changeFrequency: "monthly", priority: 0.5 },
    { url: absoluteUrl("/journal"), changeFrequency: "monthly", priority: 0.6 },
    { url: absoluteUrl("/privacy"), changeFrequency: "yearly", priority: 0.1 },
  ];

  const categories = Array.from(
    new Set(products.map((p) => (p.category || "").trim()).filter(Boolean))
  ).map((c) => ({
    url: absoluteUrl(`/collections/${categorySlug(c)}`),
    changeFrequency: "weekly" as const,
    priority: 0.8,
  }));

  const productUrls = products.map((p) => ({
    url: absoluteUrl(productPath(p)),
    lastModified: p.updated_at ? new Date(p.updated_at) : undefined,
    changeFrequency: "weekly" as const,
    priority: 0.8,
  }));

  const essays = ESSAYS.map((e) => ({
    url: absoluteUrl(`/journal/${e.slug}`),
    changeFrequency: "yearly" as const,
    priority: 0.5,
  }));

  return [...statics, ...categories, ...productUrls, ...essays];
}
