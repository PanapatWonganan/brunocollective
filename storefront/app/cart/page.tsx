"use client";

import Link from "next/link";
import { useCart } from "@/lib/cart";
import { money, imageSrc } from "@/lib/format";
import styles from "./cart.module.css";

export default function CartPage() {
  const { lines, total, setQuantity, remove } = useCart();

  return (
    <main className={styles.page}>
      <header className={styles.head}>
        <span className="kicker">Your Selection</span>
        <h1 className={`display ${styles.title}`}>The Bag</h1>
      </header>

      {lines.length === 0 ? (
        <div className={styles.empty}>
          <p className={styles.emptyNote}>Your bag is quiet for now.</p>
          <Link href="/shop" className="qlink">
            Explore the Collection <span className="arrow">→</span>
          </Link>
        </div>
      ) : (
        <div className={styles.layout}>
          <div className={styles.lines}>
            {lines.map((l) => (
              <div key={l.product.id} className={styles.line}>
                <Link href={`/product/${l.product.id}`} className={styles.thumb}>
                  <div
                    style={{
                      backgroundImage: l.product.image_url
                        ? `url('${imageSrc(l.product.image_url)}')`
                        : undefined,
                    }}
                  />
                </Link>
                <div className={styles.body}>
                  <Link href={`/product/${l.product.id}`} className={styles.name}>
                    {l.product.name}
                  </Link>
                  <div className={styles.unit}>{money(l.product.price)}</div>
                  <div className={styles.controls}>
                    <div className={styles.stepper}>
                      <button onClick={() => setQuantity(l.product.id, l.quantity - 1)} aria-label="Decrease">−</button>
                      <span>{l.quantity}</span>
                      <button
                        onClick={() => setQuantity(l.product.id, l.quantity + 1)}
                        aria-label="Increase"
                        disabled={l.quantity >= l.product.stock}
                      >
                        +
                      </button>
                    </div>
                    <button className={styles.remove} onClick={() => remove(l.product.id)}>
                      Remove
                    </button>
                  </div>
                </div>
                <div className={styles.lineTotal}>{money(l.product.price * l.quantity)}</div>
              </div>
            ))}
          </div>

          <aside className={styles.summary}>
            <h2 className={styles.sumTitle}>Summary</h2>
            <div className={styles.sumRow}>
              <span>Subtotal</span>
              <span>{money(total)}</span>
            </div>
            <div className={styles.sumRow}>
              <span>Shipping</span>
              <span>Calculated at checkout</span>
            </div>
            <div className={`${styles.sumRow} ${styles.sumTotal}`}>
              <span>Total</span>
              <span>{money(total)}</span>
            </div>
            <Link href="/checkout" className={styles.checkout}>
              Proceed to Checkout <span className="arrow">→</span>
            </Link>
            <Link href="/shop" className={`qlink qlink--ghost ${styles.cont}`}>
              Continue Shopping
            </Link>
          </aside>
        </div>
      )}
    </main>
  );
}
