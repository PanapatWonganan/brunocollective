import type { Metadata } from "next";
import { Cormorant_Garamond, Inter, Playfair_Display } from "next/font/google";
import { CartProvider } from "@/lib/cart";
import { MemberProvider } from "@/lib/member";
import { Suspense } from "react";
import TopBar, { type NavFeatured } from "@/components/TopBar";
import AffiliateRefCapture from "@/components/AffiliateRefCapture";
import AnnouncementBar from "@/components/AnnouncementBar";
import Footer from "@/components/Footer";
import BagDrawer from "@/components/BagDrawer";
import MetaPixel from "@/components/MetaPixel";
import { getProducts } from "@/lib/api";
import type { Product } from "@/lib/types";
import { SITE_URL, SITE_NAME } from "@/lib/site";
import JsonLd from "@/components/JsonLd";
import "./globals.css";

const cormorant = Cormorant_Garamond({
  subsets: ["latin"],
  weight: ["300", "400", "500", "600"],
  style: ["normal", "italic"],
  variable: "--font-cormorant",
});
const playfair = Playfair_Display({
  subsets: ["latin"],
  weight: ["400", "500", "600"],
  style: ["normal", "italic"],
  variable: "--font-playfair",
});
const inter = Inter({
  subsets: ["latin"],
  weight: ["300", "400", "500", "600"],
  variable: "--font-inter",
});

export const metadata: Metadata = {
  metadataBase: new URL(SITE_URL),
  title: {
    default: "Bruno Collective — Quietly Made in Thailand",
    template: "%s — Bruno Collective",
  },
  description:
    "Considered clothing, cut and finished by hand in Thailand — born from a love of fine cloth and quiet luxury. เสื้อผ้าคุณภาพ ตัดเย็บในไทย.",
  // Every page gets a self-referencing canonical unless it sets its own.
  alternates: { canonical: "./" },
  openGraph: {
    siteName: SITE_NAME,
    locale: "th_TH",
    title: "Bruno Collective — Quietly Made in Thailand",
    description:
      "Considered clothing, cut and finished by hand in Thailand — born from a love of fine cloth and quiet luxury.",
    type: "website",
  },
  robots: { index: true, follow: true },
};

// Site-wide entity data so search and answer engines can tie every page to
// one brand: who we are, where, and how to reach us.
const organizationLd = {
  "@context": "https://schema.org",
  "@type": "Organization",
  "@id": `${SITE_URL}/#organization`,
  name: SITE_NAME,
  url: SITE_URL,
  logo: `${SITE_URL}/icon.png`,
  description:
    "Bruno Collective — considered clothing, cut and finished by hand in Thailand.",
  address: { "@type": "PostalAddress", addressCountry: "TH" },
};
const websiteLd = {
  "@context": "https://schema.org",
  "@type": "WebSite",
  "@id": `${SITE_URL}/#website`,
  url: SITE_URL,
  name: SITE_NAME,
  publisher: { "@id": `${SITE_URL}/#organization` },
  inLanguage: ["th", "en"],
};

export default async function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  // Nav data for the mega menu — live categories plus a featured piece whose
  // photo comes from the admin product uploads. Non-fatal when the backend
  // is unreachable.
  let products: Product[] = [];
  try {
    products = await getProducts();
  } catch {
    products = [];
  }
  const categories = Array.from(
    new Set(products.map((p) => (p.category || "").trim()).filter(Boolean))
  );
  const featuredProduct = products.find((p) => p.image_url || p.images?.[0]);
  const featured: NavFeatured | null = featuredProduct
    ? {
        id: featuredProduct.id,
        slug: featuredProduct.slug,
        name: featuredProduct.name,
        image: featuredProduct.image_url || featuredProduct.images?.[0] || "",
      }
    : null;

  return (
    <html
      lang="en"
      className={`${cormorant.variable} ${playfair.variable} ${inter.variable}`}
    >
      <body>
        <JsonLd data={organizationLd} />
        <JsonLd data={websiteLd} />
        <MetaPixel />
        <MemberProvider>
          <CartProvider>
            <Suspense fallback={null}>
              <AffiliateRefCapture />
            </Suspense>
            <AnnouncementBar />
            <TopBar categories={categories} featured={featured} />
            {children}
            <Footer />
            <BagDrawer />
          </CartProvider>
        </MemberProvider>
      </body>
    </html>
  );
}
