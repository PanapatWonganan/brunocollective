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
            <span className="kicker">The Atelier — Khon Kaen, Thailand</span>
            <h2>
              Finished
              <br />
              by <em>hand.</em>
            </h2>
            <p>
              Each piece begins with a single length of cloth and the steady
              attention of a small team of makers who care more about how a seam
              sits than how fast it is sewn.
            </p>
            <p>
              Seams are pressed open. Buttons are sewn on by hand, not fired from
              a machine. Hems are checked twice before a piece ever leaves the
              studio — because quiet luxury is really just care you can feel.
            </p>

            <div className={s.specs}>
              <div>
                <div className={s.k}>Made in</div>
                <div className={s.v}>Khon Kaen, TH</div>
              </div>
              <div>
                <div className={s.k}>Finishing</div>
                <div className={s.v}>By hand</div>
              </div>
              <div>
                <div className={s.k}>Runs</div>
                <div className={s.v}>Limited</div>
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
