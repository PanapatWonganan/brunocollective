import Link from "next/link";
import Reveal from "@/components/Reveal";
import { imageSrc } from "@/lib/format";
import { ESSAYS, em } from "@/lib/journal";
import type { SiteImage } from "@/lib/api";
import s from "./sections.module.css";

// Entries come from the shared essay data (lib/journal) so the cards, the
// /journal index and the article pages always agree.
const DELAYS = [undefined, 2, 3] as const;
const ENTRIES = ESSAYS.map((e, i) => ({
  slug: e.slug,
  img: e.cover,
  tag: e.tag,
  read: e.read,
  d: DELAYS[i % 3],
  title: em(e.title),
  body: e.summary,
}));

export default function Journal({ site }: { site?: Record<string, SiteImage> }) {
  // Merge each entry with its admin-managed slot (journal_1…3); caption_a maps
  // to the tag, caption_b to the read time. Title/body stay code-defined.
  const entries = ENTRIES.map((e, i) => {
    const slot = site?.[`journal_${i + 1}`];
    return {
      ...e,
      img: slot?.image_url ? imageSrc(slot.image_url) : e.img,
      tag: slot?.caption_a || e.tag,
      read: slot?.caption_b || e.read,
    };
  });

  return (
    <section className={`${s.journal} section`} id="journal">
      <div className="wrap">
        <Reveal className="sec-head">
          <div className="num">IV.</div>
          <div className="right">
            <span className="kicker">The Journal</span>
            <h2>
              Stories from <em>the house.</em>
            </h2>
          </div>
        </Reveal>

        <div className={s.journGrid}>
          {entries.map((e, i) => (
            <Reveal as="article" key={e.slug} delay={e.d}>
              <div className={s.journImgbox}>
                <div className={s.journImg} style={{ backgroundImage: `url('${e.img}')` }} />
              </div>
              <div className={s.meta}>
                <span>{e.tag}</span>
                <span>{e.read}</span>
              </div>
              <h3>
                <Link href={`/journal/${e.slug}`}>{e.title}</Link>
              </h3>
              <p>{e.body}</p>
              <Link href={`/journal/${e.slug}`} className="qlink qlink--ghost read">
                Read — <span className="arrow">→</span>
              </Link>
            </Reveal>
          ))}
        </div>
      </div>
    </section>
  );
}
