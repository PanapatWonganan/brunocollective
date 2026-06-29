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
              Made well, <em>made in Thailand.</em>
            </h2>
          </div>
        </Reveal>

        <div className={s.philGrid}>
          <Reveal as="p" className={s.lead}>
            <span className={s.drop}>W</span>e do not chase the season. We attend
            to it — slowly, patiently, in the belief that a well-made garment is
            a quiet kind of luxury, worn for years rather than a single summer.
          </Reveal>
          <Reveal delay={2} className={s.body}>
            <p>
              Bruno Collective began with a simple obsession — fine cloth, clean
              lines, and clothes that feel as good as they look. We started in a
              small studio in Khon Kaen, cutting and finishing each piece by hand,
              and we have kept it that way.
            </p>
            <p>
              We choose fabric the way some choose coffee — for where it comes
              from, for how it breathes, for the way it softens with wear. We
              believe a garment should outlast its purchase, and that getting
              dressed well need not be loud to be luxurious.
            </p>
            <p>
              There is little here that is loud. Only the steady practice of
              making things well, in Thailand. ตั้งใจทำให้ดี ในทุกชิ้น.
            </p>
            <div className={s.sig}>
              <div>
                <div className={s.sigName}>Bruno Collective</div>
                <div className={s.sigRole}>
                  Designed &amp; finished by hand — Khon Kaen, Thailand
                </div>
              </div>
            </div>
          </Reveal>
        </div>
      </div>
    </section>
  );
}
