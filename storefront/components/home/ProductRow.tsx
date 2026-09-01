import Link from "next/link";
import type { ReactNode } from "react";
import Reveal from "@/components/Reveal";
import ProductCard from "@/components/ProductCard";
import type { Product } from "@/lib/types";
import s from "./home.module.css";

// A titled 4-up product row ("Best sellers." / "New arrivals.") mirroring the
// design's .prow sections. Hidden when there is nothing to show.
export default function ProductRow({
  title,
  products,
  viewAllHref = "/shop",
  bare = false,
}: {
  title: ReactNode;
  products: Product[];
  viewAllHref?: string;
  // bare: caller already provides the page padding/max-width (e.g. the
  // product page) — skip the .wrap container.
  bare?: boolean;
}) {
  if (products.length === 0) return null;
  return (
    <section className={s.prow}>
      <div className={bare ? undefined : "wrap"}>
        <Reveal className={s.rowHead}>
          <h2>{title}</h2>
          <Link href={viewAllHref} className="qlink">
            View All <span className="arrow">→</span>
          </Link>
        </Reveal>
        <div className={s.pgrid}>
          {products.slice(0, 4).map((p) => (
            <ProductCard key={p.id} product={p} />
          ))}
        </div>
      </div>
    </section>
  );
}
