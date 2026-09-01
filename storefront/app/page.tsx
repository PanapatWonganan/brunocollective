import HeroCarousel from "@/components/home/HeroCarousel";
import CategoryTiles from "@/components/home/CategoryTiles";
import ProductRow from "@/components/home/ProductRow";
import Values from "@/components/home/Values";
import Statement from "@/components/home/Statement";
import AtelierNotes from "@/components/home/AtelierNotes";
import CommunityGrid from "@/components/home/CommunityGrid";
import ServiceStrip from "@/components/home/ServiceStrip";
import Newsletter from "@/components/landing/Newsletter";
import { getBestSellers, getProducts, getSiteImages, type SiteImage } from "@/lib/api";
import type { Product } from "@/lib/types";

// The shop home (design: "Bruno Shop"). All imagery is admin-managed: product
// photos from the product uploads, hero/community photos from Site Images.
export default async function HomePage() {
  let products: Product[] = [];
  let best: Product[] = [];
  let siteImages: Record<string, SiteImage> = {};
  try {
    [products, best, siteImages] = await Promise.all([
      getProducts().catch(() => [] as Product[]),
      getBestSellers(4).catch(() => [] as Product[]),
      getSiteImages().catch(() => ({}) as Record<string, SiteImage>),
    ]);
  } catch {
    /* backend unreachable — render the static sections */
  }

  // New arrivals: newest first, regardless of the owner's display order.
  const newest = [...products]
    .sort(
      (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
    )
    .slice(0, 4);

  // Avoid showing the exact same 4 pieces twice when the shop is small.
  const bestIds = new Set(best.map((p) => p.id));
  const arrivals = newest.filter((p) => !bestIds.has(p.id)).length >= 2
    ? newest.filter((p) => !bestIds.has(p.id))
    : newest;

  return (
    <main>
      <HeroCarousel site={siteImages} />
      <CategoryTiles products={products} />
      <ProductRow
        title={
          <>
            Best <em>sellers.</em>
          </>
        }
        products={best}
      />
      <ProductRow
        title={
          <>
            New <em>arrivals.</em>
          </>
        }
        products={arrivals}
      />
      <Values />
      <Statement />
      <AtelierNotes />
      <CommunityGrid site={siteImages} />
      <ServiceStrip />
      <Newsletter />
    </main>
  );
}
