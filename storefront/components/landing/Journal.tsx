import Reveal from "@/components/Reveal";
import { imageSrc } from "@/lib/format";
import type { SiteImage } from "@/lib/api";
import s from "./sections.module.css";

const ENTRIES = [
  {
    img: "https://images.unsplash.com/photo-1531572753322-ad063cecc140?auto=format&fit=crop&w=1200&q=80",
    tag: "Essay — N° 03", read: "8 min",
    title: <>Why we make it <em>in Thailand</em></>,
    body: "On choosing to design and finish every piece in Khon Kaen — what local making lets us control, and why slower, smaller runs make better clothes.",
  },
  {
    img: "https://images.pexels.com/photos/2474308/pexels-photo-2474308.jpeg?auto=compress&cs=tinysrgb&w=1200",
    tag: "Essay — N° 02", read: "12 min", d: 2 as const,
    title: <>The art of the <em>wardrobe</em></>,
    body: "A few good garments, worn often — on building a quiet wardrobe that travels with you and outlives the season that bore it.",
  },
  {
    img: "https://images.unsplash.com/photo-1481627834876-b7833e8f5570?auto=format&fit=crop&w=1200&q=80",
    tag: "Essay — N° 01", read: "6 min", d: 3 as const,
    title: <>What &ldquo;quiet luxury&rdquo; <em>really means</em></>,
    body: "Less logo, more cloth. On the small details — the hand of a fabric, a clean seam, a hem that hangs right — that separate considered clothing from the rest.",
  },
];

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
            <Reveal as="article" key={i} delay={e.d}>
              <div className={s.journImgbox}>
                <div className={s.journImg} style={{ backgroundImage: `url('${e.img}')` }} />
              </div>
              <div className={s.meta}>
                <span>{e.tag}</span>
                <span>{e.read}</span>
              </div>
              <h3>{e.title}</h3>
              <p>{e.body}</p>
              <a href="#" className="qlink qlink--ghost read">
                Read — <span className="arrow">→</span>
              </a>
            </Reveal>
          ))}
        </div>
      </div>
    </section>
  );
}
