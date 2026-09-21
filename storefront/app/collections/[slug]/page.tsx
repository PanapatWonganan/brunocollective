import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { getProducts } from "@/lib/api";
import { categoryFromSlug, categorySlug } from "@/lib/categories";
import { absoluteUrl } from "@/lib/site";
import type { Product } from "@/lib/types";
import JsonLd from "@/components/JsonLd";
import ShopClient from "@/app/shop/ShopClient";

// Category landing page: /collections/{slug} (e.g. /collections/t-shirts).
// A crawlable, canonical URL per category — the old /shop?cat=… filter still
// works but links now point here so each category can rank on its own.

interface Params {
  params: Promise<{ slug: string }>;
}

async function load(slug: string): Promise<{ products: Product[]; category: string } | null> {
  let products: Product[] = [];
  try {
    products = await getProducts({ includeOut: true });
  } catch {
    return null;
  }
  const categories = Array.from(
    new Set(products.map((p) => (p.category || "").trim()).filter(Boolean))
  );
  const category = categoryFromSlug(slug, categories);
  if (!category) return null;
  return { products, category };
}

export async function generateMetadata({ params }: Params): Promise<Metadata> {
  const { slug } = await params;
  const data = await load(slug);
  if (!data) return { title: "Not found" };
  const { category, products } = data;
  const count = products.filter((p) => (p.category || "").trim() === category).length;
  const canonical = `/collections/${categorySlug(category)}`;
  return {
    title: category,
    description: `${category} by Bruno Collective — ${count} ${count === 1 ? "piece" : "pieces"}, cut and finished by hand in Thailand. ${category} คุณภาพ ตัดเย็บในไทย จำนวนจำกัด.`,
    alternates: { canonical },
    openGraph: { type: "website", url: canonical, title: category },
  };
}

export default async function CollectionPage({ params }: Params) {
  const { slug } = await params;
  const data = await load(slug);
  if (!data) notFound();
  const { products, category } = data;

  const breadcrumbLd = {
    "@context": "https://schema.org",
    "@type": "BreadcrumbList",
    itemListElement: [
      { "@type": "ListItem", position: 1, name: "Home", item: absoluteUrl("/") },
      { "@type": "ListItem", position: 2, name: "Shop", item: absoluteUrl("/shop") },
      {
        "@type": "ListItem",
        position: 3,
        name: category,
        item: absoluteUrl(`/collections/${categorySlug(category)}`),
      },
    ],
  };

  return (
    <>
      <JsonLd data={breadcrumbLd} />
      <ShopClient products={products} initialCat={category} initialSort="default" />
    </>
  );
}
