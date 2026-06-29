import Hero from "@/components/landing/Hero";
import MarqueeStrip from "@/components/landing/MarqueeStrip";
import Philosophy from "@/components/landing/Philosophy";
import CollectionGrid from "@/components/landing/CollectionGrid";
import Atelier from "@/components/landing/Atelier";
import Lookbook from "@/components/landing/Lookbook";
import Journal from "@/components/landing/Journal";
import Newsletter from "@/components/landing/Newsletter";
import { getProducts, getSiteImages, type SiteImage } from "@/lib/api";
import type { Product } from "@/lib/types";

export default async function HomePage() {
  // Real catalogue powers the featured collection; if the backend is
  // unreachable we still render the editorial page.
  let products: Product[] = [];
  try {
    products = await getProducts();
  } catch {
    products = [];
  }

  // Editable hero/lookbook/journal images; empty map → components use defaults.
  let siteImages: Record<string, SiteImage> = {};
  try {
    siteImages = await getSiteImages();
  } catch {
    siteImages = {};
  }

  return (
    <main>
      <Hero site={siteImages} />
      <MarqueeStrip />
      <Philosophy />
      <CollectionGrid products={products} />
      <Atelier />
      <Lookbook site={siteImages} />
      <Journal site={siteImages} />
      <Newsletter />
    </main>
  );
}
