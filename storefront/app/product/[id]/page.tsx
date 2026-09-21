import { permanentRedirect, notFound } from "next/navigation";
import { getProduct } from "@/lib/api";

// Legacy product URL (/product/{id}) → canonical /products/{slug}. Kept so
// old links in chats, ads and Google keep working, with a 308 so search
// engines transfer the ranking to the slug URL.
export default async function LegacyProductPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  const product = await getProduct(id).catch(() => null);
  if (!product) notFound();
  permanentRedirect(`/products/${product.slug || product.id}`);
}
