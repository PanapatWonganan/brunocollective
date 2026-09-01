"use client";

import { useMemo, useState } from "react";
import Link from "next/link";
import ProductCard from "@/components/ProductCard";
import type { Product } from "@/lib/types";
import styles from "./shop.module.css";

// THB price buckets for the filter sidebar.
const PRICE_BUCKETS = [
  { label: "ต่ำกว่า ฿500", min: 0, max: 499 },
  { label: "฿500 – ฿999", min: 500, max: 999 },
  { label: "฿1,000 – ฿1,999", min: 1000, max: 1999 },
  { label: "฿2,000 ขึ้นไป", min: 2000, max: Infinity },
];

// Sizes offered by a product — variant sizes, else the legacy single size.
function productSizes(p: Product): string[] {
  const fromVariants = (p.variants || []).map((v) => v.size).filter(Boolean);
  if (fromVariants.length) return Array.from(new Set(fromVariants));
  return p.size ? [p.size] : [];
}

function toggle(list: string[], value: string): string[] {
  return list.includes(value) ? list.filter((v) => v !== value) : [...list, value];
}

// Collapsible filter group (design .fgroup).
function FilterGroup({
  title,
  defaultOpen,
  children,
}: {
  title: string;
  defaultOpen?: boolean;
  children: React.ReactNode;
}) {
  const [open, setOpen] = useState(!!defaultOpen);
  return (
    <div className={`${styles.fgroup} ${open ? styles.fgroupOpen : ""}`}>
      <button
        type="button"
        className={styles.fh}
        onClick={() => setOpen((o) => !o)}
        aria-expanded={open}
      >
        {title} <span className={styles.car}>⌄</span>
      </button>
      <div className={styles.fb} hidden={!open}>
        {children}
      </div>
    </div>
  );
}

function CheckOption({
  label,
  count,
  on,
  onClick,
}: {
  label: string;
  count?: number;
  on: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      className={`${styles.fopt} ${on ? styles.foptOn : ""}`}
      onClick={onClick}
      aria-pressed={on}
    >
      <span className={styles.bx} />
      <span className={styles.foptLabel}>{label}</span>
      {typeof count === "number" && <span className={styles.foptCount}>{count}</span>}
    </button>
  );
}

export default function ShopClient({
  products,
  initialCat,
  initialSort,
}: {
  products: Product[];
  initialCat: string;
  initialSort: "default" | "new";
}) {
  const [cats, setCats] = useState<string[]>(initialCat ? [initialCat] : []);
  const [sizes, setSizes] = useState<string[]>([]);
  const [prices, setPrices] = useState<number[]>([]); // indexes into PRICE_BUCKETS
  const [inStockOnly, setInStockOnly] = useState(false);
  const [sort, setSort] = useState<"default" | "new">(initialSort);

  // Facets from the live catalogue.
  const catFacet = useMemo(() => {
    const m = new Map<string, number>();
    for (const p of products) {
      const c = (p.category || "").trim();
      if (!c) continue;
      m.set(c, (m.get(c) || 0) + 1);
    }
    return Array.from(m.entries()).sort((a, b) => b[1] - a[1]);
  }, [products]);

  const sizeFacet = useMemo(() => {
    const s = new Set<string>();
    for (const p of products) for (const v of productSizes(p)) s.add(v);
    // Common apparel order first, anything else alphabetical after.
    const order = ["XS", "S", "M", "L", "XL", "2XL", "XXL", "3XL"];
    return Array.from(s).sort((a, b) => {
      const ia = order.indexOf(a.toUpperCase());
      const ib = order.indexOf(b.toUpperCase());
      if (ia !== -1 && ib !== -1) return ia - ib;
      if (ia !== -1) return -1;
      if (ib !== -1) return 1;
      return a.localeCompare(b, "th");
    });
  }, [products]);

  const shown = useMemo(() => {
    let list = products.filter((p) => {
      if (cats.length && !cats.includes((p.category || "").trim())) return false;
      if (sizes.length) {
        const ps = productSizes(p);
        if (!sizes.some((s) => ps.includes(s))) return false;
      }
      if (prices.length) {
        const inBucket = prices.some((i) => {
          const b = PRICE_BUCKETS[i];
          return p.price >= b.min && p.price <= b.max;
        });
        if (!inBucket) return false;
      }
      if (inStockOnly) {
        const stock = p.variants?.length ? p.total_stock : p.stock;
        if (stock <= 0) return false;
      }
      return true;
    });
    if (sort === "new") {
      list = [...list].sort(
        (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
      );
    }
    return list;
  }, [products, cats, sizes, prices, inStockOnly, sort]);

  const anyFilter = cats.length > 0 || sizes.length > 0 || prices.length > 0 || inStockOnly;

  return (
    <main className={styles.page}>
      <div className={styles.crumb}>
        <Link href="/">Home</Link>
        <span>/</span>
        <Link href="/shop">Shop</Link>
        <span>/</span>
        {cats.length === 1 ? cats[0] : "All Pieces"}
      </div>

      <div className={styles.head}>
        <h1 className={`display ${styles.title}`}>
          The <em>Collection.</em>
        </h1>
        <div className={styles.count}>
          {shown.length} {shown.length === 1 ? "piece" : "pieces"}
        </div>
      </div>

      {products.length === 0 ? (
        <p className={styles.empty}>
          The atelier is between collections. Please return shortly.
        </p>
      ) : (
        <div className={styles.layout}>
          <aside className={styles.filters} aria-label="Filters">
            <FilterGroup title="Category" defaultOpen>
              <CheckOption
                label="All"
                on={cats.length === 0}
                onClick={() => setCats([])}
              />
              {catFacet.map(([c, n]) => (
                <CheckOption
                  key={c}
                  label={c}
                  count={n}
                  on={cats.includes(c)}
                  onClick={() => setCats((prev) => toggle(prev, c))}
                />
              ))}
            </FilterGroup>

            {sizeFacet.length > 0 && (
              <FilterGroup title="Size">
                {sizeFacet.map((s) => (
                  <CheckOption
                    key={s}
                    label={s}
                    on={sizes.includes(s)}
                    onClick={() => setSizes((prev) => toggle(prev, s))}
                  />
                ))}
              </FilterGroup>
            )}

            <FilterGroup title="Price">
              {PRICE_BUCKETS.map((b, i) => (
                <CheckOption
                  key={b.label}
                  label={b.label}
                  on={prices.includes(i)}
                  onClick={() =>
                    setPrices((prev) =>
                      prev.includes(i) ? prev.filter((x) => x !== i) : [...prev, i]
                    )
                  }
                />
              ))}
            </FilterGroup>

            <FilterGroup title="Availability">
              <CheckOption
                label="In stock only — เฉพาะมีของ"
                on={inStockOnly}
                onClick={() => setInStockOnly((v) => !v)}
              />
            </FilterGroup>

            <FilterGroup title="Sort">
              <CheckOption
                label="Featured — แนะนำ"
                on={sort === "default"}
                onClick={() => setSort("default")}
              />
              <CheckOption
                label="Newest first — มาใหม่"
                on={sort === "new"}
                onClick={() => setSort("new")}
              />
            </FilterGroup>

            {anyFilter && (
              <button
                type="button"
                className={styles.clear}
                onClick={() => {
                  setCats([]);
                  setSizes([]);
                  setPrices([]);
                  setInStockOnly(false);
                }}
              >
                Clear filters ✕
              </button>
            )}
          </aside>

          {shown.length === 0 ? (
            <p className={styles.empty}>
              ไม่พบสินค้าตามตัวกรองที่เลือก — ลองล้างตัวกรองดูครับ
            </p>
          ) : (
            <div className={styles.grid}>
              {shown.map((p) => (
                <ProductCard key={p.id} product={p} />
              ))}
            </div>
          )}
        </div>
      )}
    </main>
  );
}
