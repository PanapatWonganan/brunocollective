import type { Metadata } from "next";
import { notFound } from "next/navigation";
import Link from "next/link";
import { getProduct } from "@/lib/api";
import { money, imageSrc } from "@/lib/format";
import AddToBag from "@/components/AddToBag";
import ProductGallery from "@/components/ProductGallery";
import styles from "./product.module.css";

// Build the gallery list: prefer the multi-image array, fall back to the
// legacy single image_url, de-duplicated and stripped of blanks.
function galleryImages(product: { images?: string[]; image_url?: string }): string[] {
  const list = [...(product.images || [])];
  if (product.image_url && !list.includes(product.image_url)) {
    list.unshift(product.image_url);
  }
  return list.filter(Boolean);
}

interface Params {
  params: Promise<{ id: string }>;
}

export async function generateMetadata({ params }: Params): Promise<Metadata> {
  const { id } = await params;
  const product = await getProduct(id).catch(() => null);
  if (!product) return { title: "Not found" };
  return {
    title: product.name,
    description:
      product.description ||
      `${product.name} — cut and finished by hand at the Bruno Collective atelier in Khon Kaen, Thailand.`,
    openGraph: {
      title: product.name,
      description: product.description || product.name,
      images: product.image_url ? [imageSrc(product.image_url)] : undefined,
    },
  };
}

export default async function ProductPage({ params }: Params) {
  const { id } = await params;
  const product = await getProduct(id).catch(() => null);
  if (!product) notFound();

  return (
    <main className={styles.page}>
      <div className={styles.crumbs}>
        <Link href="/shop">The Collection</Link>
        <span>/</span>
        <span>{product.name}</span>
      </div>

      <div className={styles.grid}>
        <ProductGallery images={galleryImages(product)} alt={product.name} />

        <div className={styles.detail}>
          <span className="kicker">Bruno Collective — Made in Thailand</span>
          <h1 className={styles.name}>{product.name}</h1>
          <div className={styles.price}>{money(product.price)}</div>

          {product.description && (
            <p className={styles.desc}>{product.description}</p>
          )}

          <AddToBag product={product} />

          <dl className={styles.specs}>
            {product.sku && (
              <div>
                <dt>Reference</dt>
                <dd>{product.sku}</dd>
              </div>
            )}
            {product.size && (
              <div>
                <dt>Size</dt>
                <dd>{product.size}</dd>
              </div>
            )}
            <div>
              <dt>Availability</dt>
              <dd>
                {product.stock > 0
                  ? `${product.stock} in atelier stock`
                  : "Currently sold out"}
              </dd>
            </div>
            <div>
              <dt>Finishing</dt>
              <dd>Made &amp; finished by hand in Thailand</dd>
            </div>
          </dl>
        </div>
      </div>
    </main>
  );
}
