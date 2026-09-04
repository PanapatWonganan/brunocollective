"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import ProductCard from "@/components/ProductCard";
import { getSuggestions } from "@/lib/api";
import type { Product } from "@/lib/types";
import styles from "./ThankYou.module.css";

// Shared post-purchase "thank you" screen — used after checkout, sale-page
// orders, and the /pay slip upload. Pulls "bought together" suggestions for
// the purchased products as a gentle upsell (non-fatal when the API fails).
export default function ThankYou({
  orderNo,
  title,
  titleEm,
  copy,
  productIds = [],
  children,
}: {
  orderNo?: number | null;
  title: string;
  titleEm?: string;
  copy: string;
  productIds?: number[];
  children?: React.ReactNode;
}) {
  const [suggestions, setSuggestions] = useState<Product[]>([]);
  const idsKey = productIds.join(",");

  useEffect(() => {
    if (!idsKey) return;
    let cancelled = false;
    getSuggestions(idsKey.split(",").map(Number))
      .then((res) => {
        if (!cancelled) setSuggestions(res.slice(0, 4));
      })
      .catch(() => {});
    return () => {
      cancelled = true;
    };
  }, [idsKey]);

  return (
    <main className={styles.page}>
      <div className={styles.confirm}>
        <span className="kicker">Order Received · ขอบคุณค่ะ</span>
        <h1 className={`display ${styles.title}`}>
          {title}
          {titleEm ? (
            <>
              {" "}
              <em>{titleEm}</em>
            </>
          ) : null}
        </h1>
        <p className={styles.copy}>
          {orderNo ? `คำสั่งซื้อ N° ${orderNo} — ` : ""}
          {copy}
        </p>
        {children}
        <Link href="/shop" className="qlink">
          Explore the Collection <span className="arrow">→</span>
        </Link>
      </div>

      {suggestions.length > 0 && (
        <section className={styles.upsell}>
          <div className={styles.upsellHead}>
            <span className="kicker">While You Wait</span>
            <h2 className={`display ${styles.upsellTitle}`}>
              คุณอาจชอบ <em>You may also like</em>
            </h2>
          </div>
          <div className={styles.grid}>
            {suggestions.map((p) => (
              <ProductCard key={p.id} product={p} />
            ))}
          </div>
        </section>
      )}
    </main>
  );
}
