import type { Metadata } from "next";
import { notFound } from "next/navigation";
import Link from "next/link";
import { getProduct } from "@/lib/api";
import { money, imageSrc } from "@/lib/format";
import AddToBag from "@/components/AddToBag";
import styles from "./product.module.css";

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
      `${product.name} — finished by a single hand at the Bruno Collective atelier.`,
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
        <div className={styles.imgbox}>
          <div
            className={styles.img}
            style={{
              backgroundImage: product.image_url
                ? `url('${imageSrc(product.image_url)}')`
                : undefined,
            }}
          />
        </div>

        <div className={styles.detail}>
          <span className="kicker">Bruno Collective — Spring MMXXVI</span>
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
              <dd>By a single hand</dd>
            </div>
          </dl>
        </div>
      </div>
    </main>
  );
}
