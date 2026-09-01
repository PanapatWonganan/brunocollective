import type { Metadata } from "next";
import { AffiliateProvider } from "@/lib/affiliate";

export const metadata: Metadata = {
  title: "Affiliate Portal",
  robots: { index: false },
};

// The affiliate portal lives under /affiliate — its auth provider is scoped
// here so the shop pages carry no extra weight.
export default function AffiliateLayout({ children }: { children: React.ReactNode }) {
  return <AffiliateProvider>{children}</AffiliateProvider>;
}
