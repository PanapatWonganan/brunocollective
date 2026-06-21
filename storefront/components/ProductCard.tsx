"use client";

import Link from "next/link";
import { useCart } from "@/lib/cart";
import { money, imageSrc } from "@/lib/format";
import type { Product } from "@/lib/types";
import styles from "./ProductCard.module.css";

export default function ProductCard({ product }: { product: Product }) {
  const { add } = useCart();
  const soldOut = product.stock <= 0;
  const low = !soldOut && product.stock <= 5;

  return (
    <figure className={styles.card}>
      <Link href={`/product/${product.id}`} className={styles.imgbox} aria-label={product.name}>
        <div
          className={styles.img}
          style={{
            backgroundImage: product.image_url
              ? `url('${imageSrc(product.image_url)}')`
              : undefined,
          }}
        />
        {soldOut && <span className={styles.tag}>Sold Out</span>}
        {low && <span className={styles.tag}>Only {product.stock} left</span>}
      </Link>
      <figcaption className={styles.cap}>
        <div className={styles.name}>
          {product.sku && <small>{product.sku}</small>}
          <Link href={`/product/${product.id}`}>{product.name}</Link>
        </div>
        <div className={styles.price}>{money(product.price)}</div>
      </figcaption>
      <button className={styles.add} onClick={() => add(product)} disabled={soldOut}>
        {soldOut ? "Sold Out" : "Add to Bag"}
      </button>
    </figure>
  );
}
