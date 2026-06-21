"use client";

import { useState } from "react";
import { useCart } from "@/lib/cart";
import type { Product } from "@/lib/types";
import styles from "./AddToBag.module.css";

export default function AddToBag({ product }: { product: Product }) {
  const { add } = useCart();
  const [qty, setQty] = useState(1);
  const soldOut = product.stock <= 0;

  return (
    <div className={styles.wrap}>
      <div className={styles.row}>
        <div className={styles.stepper}>
          <button onClick={() => setQty((q) => Math.max(1, q - 1))} aria-label="Decrease" disabled={soldOut}>
            −
          </button>
          <span>{soldOut ? 0 : qty}</span>
          <button
            onClick={() => setQty((q) => Math.min(product.stock, q + 1))}
            aria-label="Increase"
            disabled={soldOut || qty >= product.stock}
          >
            +
          </button>
        </div>
        <button
          className={styles.add}
          onClick={() => add(product, qty)}
          disabled={soldOut}
        >
          {soldOut ? "Sold Out" : "Add to Bag"} <span className="arrow">→</span>
        </button>
      </div>
      {!soldOut && product.stock <= 5 && (
        <p className={styles.note}>Only {product.stock} remaining in this run.</p>
      )}
    </div>
  );
}
