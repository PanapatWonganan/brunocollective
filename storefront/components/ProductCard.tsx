"use client";

import Link from "next/link";
import { useCart } from "@/lib/cart";
import { money, imageSrc } from "@/lib/format";
import type { Product } from "@/lib/types";
import styles from "./ProductCard.module.css";

export default function ProductCard({ product }: { product: Product }) {
  const { add } = useCart();
  const hasVariants = (product.variants?.length ?? 0) > 0;
  const stock = hasVariants ? product.total_stock : product.stock;
  const soldOut = stock <= 0;
  const low = !soldOut && stock <= 5;
  const cover = product.image_url || product.images?.[0] || "";

  return (
    <figure className={styles.card}>
      <Link href={`/product/${product.id}`} className={styles.imgbox} aria-label={product.name}>
        <div
          className={styles.img}
          style={{
            backgroundImage: cover ? `url('${imageSrc(cover)}')` : undefined,
          }}
        />
        {soldOut && <span className={styles.tag}>Sold Out</span>}
        {low && <span className={styles.tag}>Only {stock} left</span>}
      </Link>
      <figcaption className={styles.cap}>
        <div className={styles.name}>
          {product.sku && <small>{product.sku}</small>}
          <Link href={`/product/${product.id}`}>{product.name}</Link>
        </div>
        <div className={styles.price}>{money(product.price)}</div>
      </figcaption>
      {hasVariants ? (
        <Link href={`/product/${product.id}`} className={styles.add} aria-disabled={soldOut}>
          {soldOut ? "Sold Out" : "Choose Options"}
        </Link>
      ) : (
        <button className={styles.add} onClick={() => add(product, null)} disabled={soldOut}>
          {soldOut ? "Sold Out" : "Add to Bag"}
        </button>
      )}
    </figure>
  );
}
