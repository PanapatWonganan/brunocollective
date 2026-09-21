import type { Product } from "./types";
import { categorySlug } from "./categories";

// Central URL builders so a route change is a one-line edit.
//
// Products live at /products/{slug}. Cart lines saved in localStorage before
// slugs existed carry no slug — those fall back to the legacy /product/{id}
// route, which redirects to the slug URL.
export function productPath(p: Pick<Product, "id" | "slug">): string {
  return p.slug ? `/products/${p.slug}` : `/product/${p.id}`;
}

export function collectionPath(category: string): string {
  return `/collections/${categorySlug(category)}`;
}
