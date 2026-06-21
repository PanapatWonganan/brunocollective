import type { Metadata } from "next";
import { getProducts } from "@/lib/api";
import type { Product } from "@/lib/types";
import ProductCard from "@/components/ProductCard";
import styles from "./shop.module.css";

export const metadata: Metadata = {
  title: "The Collection",
  description:
    "Shop the full Bruno Collective collection — limited runs, finished by a single hand.",
};

export default async function ShopPage() {
  let products: Product[] = [];
  try {
    products = await getProducts({ includeOut: true });
  } catch {
    products = [];
  }

  return (
    <main className={styles.page}>
      <header className={styles.head}>
        <span className="kicker">The Collection — Spring MMXXVI</span>
        <h1 className={`display ${styles.title}`}>
          Pieces, <em>quietly considered.</em>
        </h1>
        <p className={styles.sub}>
          Cut in limited runs and finished by a single hand from button to
          lining. Every order is reserved against our atelier stock.
        </p>
      </header>

      {products.length === 0 ? (
        <p className={styles.empty}>
          The atelier is between collections. Please return shortly.
        </p>
      ) : (
        <div className={styles.grid}>
          {products.map((p) => (
            <ProductCard key={p.id} product={p} />
          ))}
        </div>
      )}
    </main>
  );
}
