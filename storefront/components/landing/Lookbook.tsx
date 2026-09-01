import Link from "next/link";
import Reveal from "@/components/Reveal";
import { imageSrc } from "@/lib/format";
import type { SiteImage } from "@/lib/api";
import s from "./sections.module.css";

// Default tiles — shared with the full /lookbook page. Each is overridden by
// the admin lookbook_{n} site-image slot when customised.
export const LOOK_DEFAULTS = [
  { img: "https://images.pexels.com/photos/716411/pexels-photo-716411.jpeg?auto=compress&cs=tinysrgb&w=1400", a: "01 — The Studio", b: "Thailand" },
  { img: "https://images.unsplash.com/photo-1499678329028-101435549a4e?auto=format&fit=crop&w=1200&q=80", a: "02 — Daylight", b: "Morning at the atelier" },
  { img: "https://images.unsplash.com/photo-1521572163474-6864f9cf17ab?auto=format&fit=crop&w=1000&q=80", a: "03 — Everyday Tee", b: "Soft cotton jersey" },
  { img: "https://images.unsplash.com/photo-1503376780353-7e6692767b70?auto=format&fit=crop&w=1200&q=80", a: "04 — In Transit", b: "Made for the long wear" },
  { img: "https://images.unsplash.com/photo-1582719508461-905c673771fd?auto=format&fit=crop&w=1600&q=80", a: "05 — Quiet Tailoring", b: "Clean, considered lines" },
  { img: "https://images.pexels.com/photos/1300550/pexels-photo-1300550.jpeg?auto=compress&cs=tinysrgb&w=1200", a: "06 — Hand-Finished", b: "Detail you can feel" },
];

type Look = { cls: string; img: string; a: string; b: string; d?: 2 | 3 };

const PLACEMENT: { cls: string; d?: 2 | 3 }[] = [
  { cls: s.l1 }, { cls: s.l2, d: 2 }, { cls: s.l3, d: 3 },
  { cls: s.l4 }, { cls: s.l5, d: 2 }, { cls: s.l6, d: 3 },
];

const LOOKS: Look[] = LOOK_DEFAULTS.map((l, i) => ({ ...l, ...PLACEMENT[i] }));

export default function Lookbook({ site }: { site?: Record<string, SiteImage> }) {
  // Merge each tile with its admin-managed slot (lookbook_1…6); fall back to
  // the built-in default image and captions when a slot is unset.
  const looks = LOOKS.map((l, i) => {
    const slot = site?.[`lookbook_${i + 1}`];
    return {
      ...l,
      img: slot?.image_url ? imageSrc(slot.image_url) : l.img,
      a: slot?.caption_a || l.a,
      b: slot?.caption_b || l.b,
    };
  });

  return (
    <section className={`${s.look} section`} id="lookbook">
      <div className="wrap">
        <Reveal className="sec-head">
          <div className="num">III.</div>
          <div className="right">
            <span className="kicker">Lookbook — The Collection</span>
            <h2>
              Notes from the
              <br />
              <em>atelier.</em>
            </h2>
          </div>
        </Reveal>

        <div className={s.lookGrid}>
          {looks.map((l, i) => (
            <Reveal
              as="figure"
              key={i}
              delay={l.d}
              className={l.cls}
            >
              <div className={s.lookImg} style={{ backgroundImage: `url('${l.img}')` }} />
              <figcaption>
                <span>{l.a}</span>
                <span>{l.b}</span>
              </figcaption>
            </Reveal>
          ))}
        </div>

        <Reveal className={s.endline}>
          <h3>&ldquo;What endures is rarely the thing that announced itself.&rdquo;</h3>
          <Link href="/lookbook" className="qlink">
            View the Full Lookbook <span className="arrow">→</span>
          </Link>
        </Reveal>
      </div>
    </section>
  );
}
