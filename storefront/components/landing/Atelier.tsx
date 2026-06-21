import Reveal from "@/components/Reveal";
import s from "./sections.module.css";

export default function Atelier() {
  return (
    <section className={`${s.atelier} section`} id="atelier">
      <div className="wrap">
        <div className={s.atelierGrid}>
          <Reveal className={s.atelierImgbox}>
            <div className={s.atelierImg} />
          </Reveal>
          <Reveal delay={2}>
            <span className="kicker">The Atelier — Solomeo, Umbria</span>
            <h2>
              One hundred and
              <br />
              seventeen <em>hours.</em>
            </h2>
            <p>
              Each overcoat begins with a single bolt of cloth and the steady
              attention of a small group of cutters who have, between them, some
              four hundred years of practice.
            </p>
            <p>
              Linings are basted by hand. Buttonholes are worked closed — not
              stitched on, never machined. Edges are pressed under the iron in
              three separate passes, the way Tommaso&apos;s grandfather pressed
              them.
            </p>

            <div className={s.specs}>
              <div>
                <div className={s.k}>Provenance</div>
                <div className={s.v}>Solomeo, IT</div>
              </div>
              <div>
                <div className={s.k}>Hand-finishing</div>
                <div className={s.v}>117 hours</div>
              </div>
              <div>
                <div className={s.k}>Artisans</div>
                <div className={s.v}>eighty-three</div>
              </div>
            </div>

            <a href="#" className="qlink qlink--light">
              Inside the Atelier <span className="arrow">→</span>
            </a>
          </Reveal>
        </div>
      </div>
    </section>
  );
}
