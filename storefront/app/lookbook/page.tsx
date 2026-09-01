import type { Metadata } from "next";
import Link from "next/link";
import { getSiteImages } from "@/lib/api";
import { imageSrc } from "@/lib/format";
import { LOOK_DEFAULTS } from "@/components/landing/Lookbook";
import Reveal from "@/components/Reveal";
import styles from "./lookbook.module.css";

export const metadata: Metadata = {
  title: "Lookbook",
  description:
    "The Bruno Collective lookbook — notes from the atelier in Khon Kaen, Thailand. Photography of the collection, quietly considered.",
};

// Full lookbook — every tile is admin-managed via the lookbook_1..6 site-image
// slots (with the same built-in defaults as the story-page grid).
export default async function LookbookPage() {
  const site = await getSiteImages();
  const looks = LOOK_DEFAULTS.map((l, i) => {
    const slot = site[`lookbook_${i + 1}`];
    return {
      img: slot?.image_url ? imageSrc(slot.image_url) : l.img,
      a: slot?.caption_a || l.a,
      b: slot?.caption_b || l.b,
    };
  });

  return (
    <main className={styles.page}>
      <header className={styles.head}>
        <span className="kicker">Lookbook — The Collection</span>
        <h1 className={`display ${styles.title}`}>
          Notes from the <em>atelier.</em>
        </h1>
        <p className={styles.sub}>
          ภาพจากสตูดิโอของเราที่ขอนแก่น — the collection as it is actually worn,
          photographed slowly.
        </p>
      </header>

      <div className={styles.grid}>
        {looks.map((l, i) => (
          <Reveal
            as="figure"
            key={i}
            delay={([undefined, 2, 3][i % 3]) as 2 | 3 | undefined}
            className={`${styles.fig} ${styles[`f${(i % 6) + 1}`]}`}
          >
            <div className={styles.img} style={{ backgroundImage: `url('${l.img}')` }} />
            <figcaption className={styles.cap}>
              <span>{l.a}</span>
              <span>{l.b}</span>
            </figcaption>
          </Reveal>
        ))}
      </div>

      <Reveal className={styles.endline}>
        <h2 className={styles.quote}>
          &ldquo;What endures is rarely the thing that announced itself.&rdquo;
        </h2>
        <Link href="/shop" className="qlink">
          Shop the Collection <span className="arrow">→</span>
        </Link>
      </Reveal>
    </main>
  );
}
