import type { Metadata } from "next";
import Link from "next/link";
import { getSiteImages } from "@/lib/api";
import { imageSrc } from "@/lib/format";
import { ESSAYS, em } from "@/lib/journal";
import Reveal from "@/components/Reveal";
import styles from "./journal.module.css";

export const metadata: Metadata = {
  title: "The Journal",
  description:
    "Essays from the Bruno Collective atelier — on making clothes in Thailand, building a quiet wardrobe, and what quiet luxury really means.",
};

export default async function JournalIndex() {
  const site = await getSiteImages();

  return (
    <main className={styles.page}>
      <header className={styles.head}>
        <span className="kicker">The Journal</span>
        <h1 className={`display ${styles.title}`}>
          Stories from <em>the house.</em>
        </h1>
        <p className={styles.sub}>
          Occasional essays from the atelier — on cloth, care, and dressing
          quietly. เรื่องเล่าจากห้องทำงานของเรา
        </p>
      </header>

      <div className={styles.grid}>
        {ESSAYS.map((e, i) => {
          const slot = site[`journal_${i + 1}`];
          const cover = slot?.image_url ? imageSrc(slot.image_url) : e.cover;
          return (
            <Reveal
              as="article"
              key={e.slug}
              delay={([undefined, 2, 3][i % 3]) as 2 | 3 | undefined}
              className={styles.card}
            >
              <Link href={`/journal/${e.slug}`} className={styles.imgbox} aria-label={e.title.replace(/\*/g, "")}>
                <div className={styles.img} style={{ backgroundImage: `url('${cover}')` }} />
              </Link>
              <div className={styles.meta}>
                <span>{slot?.caption_a || e.tag}</span>
                <span>{slot?.caption_b || e.read}</span>
              </div>
              <h2 className={styles.cardTitle}>
                <Link href={`/journal/${e.slug}`}>{em(e.title)}</Link>
              </h2>
              <p className={styles.summary}>{e.summary}</p>
              <Link href={`/journal/${e.slug}`} className={`qlink qlink--ghost ${styles.read}`}>
                Read — <span className="arrow">→</span>
              </Link>
            </Reveal>
          );
        })}
      </div>
    </main>
  );
}
