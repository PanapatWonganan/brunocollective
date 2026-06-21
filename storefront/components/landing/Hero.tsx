import Link from "next/link";
import Reveal from "@/components/Reveal";
import styles from "./Hero.module.css";

export default function Hero() {
  return (
    <section className={styles.hero}>
      <div className={styles.img} aria-hidden />
      <div className={styles.scrim} aria-hidden />

      <div className={styles.inner}>
        <Reveal as="div" className="kicker" >
          <span style={{ color: "var(--champagne-2)" }}>
            Spring — Summer Collection — MMXXVI
          </span>
        </Reveal>
        <Reveal delay={2}>
          <h1 className={`display ${styles.h1}`}>
            A Quiet<br />
            <em>Inheritance</em>
          </h1>
        </Reveal>
        <Reveal delay={3}>
          <p className={styles.tagline}>
            Garments cut for the long quiet of a life lived deliberately —
            linen, cashmere, and time.
          </p>
        </Reveal>
        <Reveal delay={4}>
          <Link href="/shop" className="qlink qlink--light">
            Explore the Collection <span className="arrow">→</span>
          </Link>
        </Reveal>
      </div>

      <div className={styles.metaRow}>
        <div className={styles.left}>
          <span className="label">Est. 1908 — Bellagio, Lombardia</span>
        </div>
        <div className={styles.center}>
          <span className={`serif ${styles.num}`}>N°&nbsp;XVII</span>
          <span className="label">The Spring Edit</span>
        </div>
        <div className={styles.right}>
          <span className="label">Photographed at Villa Serbelloni</span>
        </div>
      </div>

      <div className={styles.scrollCue} aria-hidden>
        Scroll
        <span className={styles.line} />
      </div>
    </section>
  );
}
