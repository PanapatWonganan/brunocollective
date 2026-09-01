import Link from "next/link";
import Reveal from "@/components/Reveal";
import { imageSrc } from "@/lib/format";
import type { Product } from "@/lib/types";
import s from "./home.module.css";

// Category tiles derived from the live catalogue: distinct categories ranked
// by piece count, each covered by the first product photo in that category
// (all imagery comes from the admin uploads). Hidden entirely when the shop
// has fewer than two categories.
export default function CategoryTiles({ products }: { products: Product[] }) {
  const byCat = new Map<string, { count: number; cover: string }>();
  for (const p of products) {
    const cat = (p.category || "").trim();
    if (!cat) continue;
    const entry = byCat.get(cat) || { count: 0, cover: "" };
    entry.count += 1;
    if (!entry.cover) entry.cover = p.image_url || p.images?.[0] || "";
    byCat.set(cat, entry);
  }

  const cats = Array.from(byCat.entries())
    .sort((a, b) => b[1].count - a[1].count)
    .slice(0, 4);
  if (cats.length < 2) return null;

  return (
    <section className={s.cats}>
      <div className="wrap">
        <Reveal className={s.catsHead}>
          <span className="kicker">Find Your Essentials</span>
          <h2>
            Where shall we <em>begin?</em>
          </h2>
        </Reveal>
        <div className={s.catRow} data-count={cats.length}>
          {cats.map(([name, info], i) => (
            <Reveal
              as="div"
              key={name}
              delay={([undefined, 2, 3][i % 3]) as 2 | 3 | undefined}
            >
              <Link
                className={s.catCard}
                href={`/shop?cat=${encodeURIComponent(name)}`}
              >
                <div
                  className={s.catImg}
                  style={{
                    backgroundImage: info.cover
                      ? `url('${imageSrc(info.cover)}')`
                      : undefined,
                  }}
                />
                <div className={s.catName}>{name}</div>
                <div className={s.catCount}>
                  {info.count} {info.count === 1 ? "piece" : "pieces"}
                </div>
              </Link>
            </Reveal>
          ))}
        </div>
      </div>
    </section>
  );
}
