import Reveal from "@/components/Reveal";
import { imageSrc } from "@/lib/format";
import type { SiteImage } from "@/lib/api";
import s from "./sections.module.css";

type Look = { cls: string; img: string; a: string; b: string; d?: 2 | 3 };

const LOOKS: Look[] = [
  { cls: s.l1, img: "https://images.pexels.com/photos/716411/pexels-photo-716411.jpeg?auto=compress&cs=tinysrgb&w=1400", a: "01 — The Studio", b: "Khon Kaen, Thailand" },
  { cls: s.l2, d: 2, img: "https://images.unsplash.com/photo-1499678329028-101435549a4e?auto=format&fit=crop&w=1200&q=80", a: "02 — Daylight", b: "Morning at the atelier" },
  { cls: s.l3, d: 3, img: "https://images.unsplash.com/photo-1521572163474-6864f9cf17ab?auto=format&fit=crop&w=1000&q=80", a: "03 — Everyday Tee", b: "Soft cotton jersey" },
  { cls: s.l4, img: "https://images.unsplash.com/photo-1503376780353-7e6692767b70?auto=format&fit=crop&w=1200&q=80", a: "04 — In Transit", b: "Made for the long wear" },
  { cls: s.l5, d: 2, img: "https://images.unsplash.com/photo-1582719508461-905c673771fd?auto=format&fit=crop&w=1600&q=80", a: "05 — Quiet Tailoring", b: "Clean, considered lines" },
  { cls: s.l6, d: 3, img: "https://images.pexels.com/photos/1300550/pexels-photo-1300550.jpeg?auto=compress&cs=tinysrgb&w=1200", a: "06 — Hand-Finished", b: "Detail you can feel" },
];

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
          <a href="#" className="qlink">
            View the Full Lookbook <span className="arrow">→</span>
          </a>
        </Reveal>
      </div>
    </section>
  );
}
