import { notFound } from "next/navigation";
import type { Metadata } from "next";
import { getSalePage, getSiteImages } from "@/lib/api";
import SalePageClient from "./SalePageClient";

// Funnel/landing pages are personal-campaign URLs — always render fresh so
// countdowns, stock and draft previews are never stale.
export const dynamic = "force-dynamic";

interface Props {
  params: Promise<{ slug: string }>;
  searchParams: Promise<{ preview?: string }>;
}

export async function generateMetadata({ params, searchParams }: Props): Promise<Metadata> {
  const { slug } = await params;
  const { preview } = await searchParams;
  const page = await getSalePage(slug, preview === "1");
  if (!page) return { title: "Not Found" };
  return {
    title: page.title,
    description: `${page.product?.name} — Bruno Collective`,
    robots: { index: false }, // campaign pages shouldn't compete with the shop in search
  };
}

export default async function SaleLandingPage({ params, searchParams }: Props) {
  const { slug } = await params;
  const { preview } = await searchParams;
  const [page, siteImages] = await Promise.all([
    getSalePage(slug, preview === "1"),
    getSiteImages(),
  ]);
  if (!page) notFound();
  return (
    <SalePageClient
      page={page}
      isPreview={preview === "1"}
      sizeChartUrl={siteImages["size_chart"]?.image_url || ""}
    />
  );
}
