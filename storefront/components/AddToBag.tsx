"use client";

import { useMemo, useState } from "react";
import { useCart } from "@/lib/cart";
import type { Product, ProductVariant } from "@/lib/types";
import styles from "./AddToBag.module.css";

export default function AddToBag({ product }: { product: Product }) {
  const { add } = useCart();
  const variants = product.variants ?? [];
  const hasVariants = variants.length > 0;

  // Distinct sizes / colors across the available variants (skip blanks).
  const sizes = useMemo(
    () => Array.from(new Set(variants.map((v) => v.size).filter(Boolean))),
    [variants]
  );
  const colors = useMemo(
    () => Array.from(new Set(variants.map((v) => v.color).filter(Boolean))),
    [variants]
  );

  const [size, setSize] = useState<string>(sizes.length === 1 ? sizes[0] : "");
  const [color, setColor] = useState<string>(colors.length === 1 ? colors[0] : "");
  const [qty, setQty] = useState(1);

  // Resolve the chosen size+color to a concrete variant.
  const selected: ProductVariant | null = useMemo(() => {
    if (!hasVariants) return null;
    return (
      variants.find(
        (v) =>
          (sizes.length === 0 || v.size === size) &&
          (colors.length === 0 || v.color === color)
      ) ?? null
    );
  }, [hasVariants, variants, sizes.length, colors.length, size, color]);

  // Whether an option leads to any in-stock variant. Only constrain by the OTHER
  // dimension once the user has actually picked it — otherwise every option would
  // appear out of stock before any selection is made.
  const sizeInStock = (s: string) =>
    variants.some((v) => v.size === s && (!color || v.color === color) && v.stock > 0);
  const colorInStock = (c: string) =>
    variants.some((v) => v.color === c && (!size || v.size === size) && v.stock > 0);

  const needsSize = sizes.length > 0 && !size;
  const needsColor = colors.length > 0 && !color;

  const stock = hasVariants ? (selected ? selected.stock : 0) : Math.max(product.stock, 0);
  const soldOut = hasVariants ? product.total_stock <= 0 : product.stock <= 0;
  const canAdd = !soldOut && !needsSize && !needsColor && stock > 0;

  function handleAdd() {
    if (!canAdd) return;
    add(product, selected, qty);
  }

  return (
    <div className={styles.wrap}>
      {sizes.length > 0 && (
        <div className={styles.options}>
          <span className={styles.optLabel}>Size</span>
          <div className={styles.swatches}>
            {sizes.map((s) => (
              <button
                key={s}
                className={`${styles.swatch} ${size === s ? styles.swatchOn : ""}`}
                onClick={() => {
                  setSize(s);
                  setQty(1);
                }}
                disabled={!sizeInStock(s)}
              >
                {s}
              </button>
            ))}
          </div>
        </div>
      )}

      {colors.length > 0 && (
        <div className={styles.options}>
          <span className={styles.optLabel}>Color</span>
          <div className={styles.swatches}>
            {colors.map((c) => (
              <button
                key={c}
                className={`${styles.swatch} ${color === c ? styles.swatchOn : ""}`}
                onClick={() => {
                  setColor(c);
                  setQty(1);
                }}
                disabled={!colorInStock(c)}
              >
                {c}
              </button>
            ))}
          </div>
        </div>
      )}

      <div className={styles.row}>
        <div className={styles.stepper}>
          <button onClick={() => setQty((q) => Math.max(1, q - 1))} aria-label="Decrease" disabled={!canAdd}>
            −
          </button>
          <span>{canAdd ? qty : 0}</span>
          <button
            onClick={() => setQty((q) => Math.min(stock, q + 1))}
            aria-label="Increase"
            disabled={!canAdd || qty >= stock}
          >
            +
          </button>
        </div>
        <button className={styles.add} onClick={handleAdd} disabled={!canAdd}>
          {soldOut
            ? "Sold Out"
            : needsSize
            ? "Select a size"
            : needsColor
            ? "Select a color"
            : "Add to Bag"}{" "}
          <span className="arrow">→</span>
        </button>
      </div>

      {canAdd && stock <= 5 && (
        <p className={styles.note}>Only {stock} remaining in this run.</p>
      )}
    </div>
  );
}
