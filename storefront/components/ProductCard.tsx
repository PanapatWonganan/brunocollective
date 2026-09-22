"use client";

import Link from "next/link";
import { useCart } from "@/lib/cart";
import { imageSrc, priceLabel } from "@/lib/format";
import type { Product } from "@/lib/types";
import Rating from "./Rating";
import { productPath } from "@/lib/paths";
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
      <Link href={productPath(product)} className={styles.imgbox} aria-label={product.name}>
        {cover ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            className={styles.img}
            src={imageSrc(cover)}
            alt={product.name}
            loading="lazy"
            decoding="async"
          />
        ) : (
          <div className={styles.img} />
        )}
        {soldOut && <span className={styles.tag}>Sold Out</span>}
        {low && <span className={styles.tag}>Only {stock} left</span>}
      </Link>
      <figcaption className={styles.cap}>
        <div className={styles.name}>
          {product.sku && <small>{product.sku}</small>}
          <Link href={productPath(product)}>{product.name}</Link>
        </div>
        <div className={styles.side}>
          <div className={styles.price}>{priceLabel(product)}</div>
          <Rating value={product.rating} count={product.rating_count} />
        </div>
      </figcaption>
      {hasVariants ? (
        <Link href={productPath(product)} className={styles.add} aria-disabled={soldOut}>
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
