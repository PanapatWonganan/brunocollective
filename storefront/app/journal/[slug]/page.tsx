import type { Metadata } from "next";
import { notFound } from "next/navigation";
import Link from "next/link";
import { getSiteImages } from "@/lib/api";
import { imageSrc } from "@/lib/format";
import { ESSAYS, em, essayBySlug } from "@/lib/journal";
import Reveal from "@/components/Reveal";
import styles from "./article.module.css";

interface Params {
  params: Promise<{ slug: string }>;
}

export function generateStaticParams() {
  return ESSAYS.map((e) => ({ slug: e.slug }));
}

export async function generateMetadata({ params }: Params): Promise<Metadata> {
  const { slug } = await params;
  const essay = essayBySlug(slug);
  if (!essay) return { title: "Not found" };
  return {
    title: essay.title.replace(/\*/g, ""),
    description: essay.summary,
  };
}

export default async function JournalArticle({ params }: Params) {
  const { slug } = await params;
  const essay = essayBySlug(slug);
  if (!essay) notFound();

  const index = ESSAYS.indexOf(essay);
  // Cover photo + tag/read follow the admin journal_{n} slot when customised.
  const site = await getSiteImages();
  const slot = site[`journal_${index + 1}`];
  const cover = slot?.image_url ? imageSrc(slot.image_url) : essay.cover;
  const tag = slot?.caption_a || essay.tag;
  const read = slot?.caption_b || essay.read;
  const next = ESSAYS[(index + 1) % ESSAYS.length];

  return (
    <main className={styles.page}>
      <article>
        <header className={styles.head}>
          <div className={styles.crumb}>
            <Link href="/">Home</Link>
            <span>/</span>
            <Link href="/journal">Journal</Link>
          </div>
          <div className={styles.meta}>
            <span className="kicker">{tag}</span>
            <span className={styles.read}>{read}</span>
          </div>
          <h1 className={`display ${styles.title}`}>{em(essay.title)}</h1>
          <p className={styles.summary}>{essay.summary}</p>
        </header>

        <Reveal className={styles.coverBox}>
          <div
            className={styles.cover}
            style={{ backgroundImage: `url('${cover}')` }}
            role="img"
            aria-label={essay.title.replace(/\*/g, "")}
          />
        </Reveal>

        <div className={styles.body}>
          {essay.body.map((p, i) => (
            <p key={i} className={i === 0 ? styles.lead : undefined}>
              {i === 0 ? (
                <>
                  <span className={styles.drop}>{p.charAt(0)}</span>
                  {em(p.slice(1))}
                </>
              ) : (
                em(p)
              )}
            </p>
          ))}

          <div className={styles.sig}>
            <div className={styles.sigName}>Bruno Collective</div>
            <div className={styles.sigRole}>The Journal — Khon Kaen, Thailand</div>
          </div>
        </div>

        <footer className={styles.foot}>
          <Link href="/journal" className="qlink qlink--ghost">
            ← All Essays
          </Link>
          <Link href={`/journal/${next.slug}`} className="qlink">
            Next — {em(next.title)} <span className="arrow">→</span>
          </Link>
        </footer>
      </article>

      <div className={styles.cta}>
        <span className="kicker">The Collection</span>
        <h2 className={styles.ctaH}>Made the way we <em>write.</em></h2>
        <Link href="/shop" className="qlink">
          Shop the Collection <span className="arrow">→</span>
        </Link>
      </div>
    </main>
  );
}
