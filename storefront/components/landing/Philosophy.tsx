import Reveal from "@/components/Reveal";
import s from "./sections.module.css";

export default function Philosophy() {
  return (
    <section className={`${s.philosophy} section`} id="philosophy">
      <div className="wrap">
        <Reveal className="sec-head">
          <div className="num">I.</div>
          <div className="right">
            <span className="kicker">Manifesto</span>
            <h2>
              Heritage in <em>every thread.</em>
            </h2>
          </div>
        </Reveal>

        <div className={s.philGrid}>
          <Reveal as="p" className={s.lead}>
            <span className={s.drop}>W</span>e do not chase the season. We attend
            to it — slowly, patiently, with the conviction that a well-made
            garment is a quiet inheritance, passed from one shoulder to the next.
          </Reveal>
          <Reveal delay={2} className={s.body}>
            <p>
              Bruno Collective was begun in a single room above a tailor&apos;s
              shop on the lake at Bellagio, in the autumn of 1908. Four
              generations on, our atelier remains rooted in the same valley,
              drawing on the same mills, the same patient hands.
            </p>
            <p>
              We believe in fabric chosen the way fine wine is chosen — for
              provenance, for breath, for the way it ages. We believe a coat
              should outlast its purchase, and that a wardrobe is a kind of
              memoir, written in linen and wool.
            </p>
            <p>
              There is little here that is new. Only the steady practice of
              making things well.
            </p>
            <div className={s.sig}>
              <div>
                <div className={s.sigName}>Tommaso Bruno</div>
                <div className={s.sigRole}>
                  Fourth Generation, Maestro d&apos;Atelier
                </div>
              </div>
            </div>
          </Reveal>
        </div>
      </div>
    </section>
  );
}
