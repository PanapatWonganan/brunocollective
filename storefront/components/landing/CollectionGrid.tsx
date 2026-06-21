"use client";

import Link from "next/link";
import Reveal from "@/components/Reveal";
import { useCart } from "@/lib/cart";
import { money, imageSrc } from "@/lib/format";
import type { Product } from "@/lib/types";
import s from "./CollectionGrid.module.css";

// The design lays seven pieces out asymmetrically (c-1 … c-7). We map however
// many real products exist onto that rhythm, cycling the placement classes.
const PLACEMENTS = [s.c1, s.c2, s.c3, s.c4, s.c5, s.c6, s.c7];
const NUMERALS = ["01", "02", "03", "04", "05", "06", "07", "08", "09", "10"];

export default function CollectionGrid({ products }: { products: Product[] }) {
  const { add } = useCart();
  const shown = products.slice(0, 7);

  return (
    <section className={`${s.collection} section`} id="collection">
      <div className="wrap">
        <Reveal className="sec-head">
          <div className="num">II.</div>
          <div className="right">
            <span className="kicker">Featured — Spring Edit</span>
            <h2>
              The <em>essentials,</em>
              <br />
              quietly considered.
            </h2>
            <p className={s.intro}>
              Pieces from this season&apos;s atelier — the quiet uniform of a
              considered life, cut in limited runs and finished by a single hand
              from button to lining.
            </p>
          </div>
        </Reveal>

        {shown.length === 0 ? (
          <p className={s.empty}>The atelier is between collections. Please return shortly.</p>
        ) : (
          <div className={s.grid}>
            {shown.map((p, i) => (
              <Reveal
                as="figure"
                key={p.id}
                delay={([undefined, 2, 3][i % 3]) as 2 | 3 | undefined}
                className={PLACEMENTS[i % PLACEMENTS.length]}
              >
                <Link href={`/product/${p.id}`} className={s.imgbox} aria-label={p.name}>
                  <div
                    className={s.img}
                    style={{
                      backgroundImage: p.image_url
                        ? `url('${imageSrc(p.image_url)}')`
                        : undefined,
                    }}
                  />
                </Link>
                <figcaption className={s.cap}>
                  <div className={s.name}>
                    <small>N° {NUMERALS[i]} — Bruno Collective</small>
                    <Link href={`/product/${p.id}`}>{p.name}</Link>
                  </div>
                  <div className={s.price}>{money(p.price)}</div>
                </figcaption>
                <button
                  className={s.add}
                  onClick={() => add(p)}
                  disabled={p.stock <= 0}
                >
                  {p.stock <= 0 ? "Sold Out" : "Add to Bag"}
                </button>
              </Reveal>
            ))}
          </div>
        )}

        <div className={s.viewAll}>
          <Link href="/shop" className="qlink">
            View the Full Collection <span className="arrow">→</span>
          </Link>
        </div>
      </div>
    </section>
  );
}
