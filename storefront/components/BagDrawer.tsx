"use client";

import Link from "next/link";
import { useCart } from "@/lib/cart";
import { money, imageSrc } from "@/lib/format";
import styles from "./BagDrawer.module.css";

export default function BagDrawer() {
  const { lines, total, count, setQuantity, remove, open, setOpen } = useCart();

  return (
    <>
      <div
        className={`${styles.scrim} ${open ? styles.scrimOn : ""}`}
        onClick={() => setOpen(false)}
        aria-hidden={!open}
      />
      <aside
        className={`${styles.drawer} ${open ? styles.drawerOn : ""}`}
        aria-label="Shopping bag"
        aria-hidden={!open}
      >
        <header className={styles.head}>
          <span className="label">The Bag {count > 0 ? `— ${count}` : ""}</span>
          <button className={styles.close} onClick={() => setOpen(false)} aria-label="Close bag">
            ✕
          </button>
        </header>

        {lines.length === 0 ? (
          <div className={styles.empty}>
            <p className={styles.emptyNote}>Your bag is quiet for now.</p>
            <Link href="/shop" className="qlink" onClick={() => setOpen(false)}>
              Explore the Collection <span className="arrow">→</span>
            </Link>
          </div>
        ) : (
          <>
            <div className={styles.lines}>
              {lines.map((l) => (
                <div key={l.product.id} className={styles.line}>
                  <div
                    className={styles.thumb}
                    style={{
                      backgroundImage: l.product.image_url
                        ? `url("${imageSrc(l.product.image_url)}")`
                        : undefined,
                    }}
                  />
                  <div className={styles.lineBody}>
                    <div className={styles.lineName}>{l.product.name}</div>
                    <div className={styles.lineMeta}>{money(l.product.price)}</div>
                    <div className={styles.qtyRow}>
                      <div className={styles.stepper}>
                        <button
                          onClick={() => setQuantity(l.product.id, l.quantity - 1)}
                          aria-label="Decrease"
                        >
                          −
                        </button>
                        <span>{l.quantity}</span>
                        <button
                          onClick={() => setQuantity(l.product.id, l.quantity + 1)}
                          aria-label="Increase"
                          disabled={l.quantity >= l.product.stock}
                        >
                          +
                        </button>
                      </div>
                      <button
                        className={styles.removeBtn}
                        onClick={() => remove(l.product.id)}
                      >
                        Remove
                      </button>
                    </div>
                  </div>
                </div>
              ))}
            </div>

            <footer className={styles.foot}>
              <div className={styles.totalRow}>
                <span className="label label--quiet">Subtotal</span>
                <span className={styles.total}>{money(total)}</span>
              </div>
              <Link
                href="/checkout"
                className={styles.checkout}
                onClick={() => setOpen(false)}
              >
                Proceed to Checkout <span className="arrow">→</span>
              </Link>
            </footer>
          </>
        )}
      </aside>
    </>
  );
}
