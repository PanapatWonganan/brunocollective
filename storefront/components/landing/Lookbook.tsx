import Reveal from "@/components/Reveal";
import s from "./sections.module.css";

type Look = { cls: string; img: string; a: string; b: string; d?: 2 | 3 };

const LOOKS: Look[] = [
  { cls: s.l1, img: "https://images.pexels.com/photos/716411/pexels-photo-716411.jpeg?auto=compress&cs=tinysrgb&w=1400", a: "01 — The Reading Room", b: "Milano, MMXXVI" },
  { cls: s.l2, d: 2, img: "https://images.unsplash.com/photo-1499678329028-101435549a4e?auto=format&fit=crop&w=1200&q=80", a: "02 — Setting", b: "Bay of Camogli" },
  { cls: s.l3, d: 3, img: "https://images.unsplash.com/photo-1521572163474-6864f9cf17ab?auto=format&fit=crop&w=1000&q=80", a: "03 — The Bellagio Tee", b: "Ivory cotton jersey" },
  { cls: s.l4, img: "https://images.unsplash.com/photo-1503376780353-7e6692767b70?auto=format&fit=crop&w=1200&q=80", a: "04 — In Transit", b: "The Lakes, MMXXVI" },
  { cls: s.l5, d: 2, img: "https://images.unsplash.com/photo-1582719508461-905c673771fd?auto=format&fit=crop&w=1600&q=80", a: "05 — The Estate", b: "Villa Serbelloni" },
  { cls: s.l6, d: 3, img: "https://images.pexels.com/photos/1300550/pexels-photo-1300550.jpeg?auto=compress&cs=tinysrgb&w=1200", a: "06 — The Maroon Suit", b: "Tuscan late afternoon" },
];

export default function Lookbook() {
  return (
    <section className={`${s.look} section`} id="lookbook">
      <div className="wrap">
        <Reveal className="sec-head">
          <div className="num">III.</div>
          <div className="right">
            <span className="kicker">Lookbook — Spring MMXXVI</span>
            <h2>
              Notes from the
              <br />
              <em>Italian Riviera.</em>
            </h2>
          </div>
        </Reveal>

        <div className={s.lookGrid}>
          {LOOKS.map((l, i) => (
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
