import Link from "next/link";
import Reveal from "@/components/Reveal";
import { imageSrc } from "@/lib/format";
import type { SiteImage } from "@/lib/api";
import styles from "./Hero.module.css";

export default function Hero({ site }: { site?: Record<string, SiteImage> }) {
  // Custom hero image overrides the CSS-module default background when set.
  const custom = site?.hero?.image_url;

  return (
    <section className={styles.hero}>
      <div
        className={styles.img}
        aria-hidden
        style={custom ? { backgroundImage: `url('${imageSrc(custom)}')` } : undefined}
      />
      <div className={styles.scrim} aria-hidden />

      <div className={styles.inner}>
        <Reveal as="div" className="kicker" >
          <span style={{ color: "var(--champagne-2)" }}>
            Quietly Made in Thailand — เสื้อผ้าตัดเย็บในไทย
          </span>
        </Reveal>
        <Reveal delay={2}>
          <h1 className={`display ${styles.h1}`}>
            Quiet<br />
            <em>Luxury</em>
          </h1>
        </Reveal>
        <Reveal delay={3}>
          <p className={styles.tagline}>
            Born from a love of fine cloth and understated elegance —
            designed and finished by hand in Khon Kaen.
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
          <span className="label">Designed &amp; Made — Khon Kaen, Thailand</span>
        </div>
        <div className={styles.center}>
          <span className={`serif ${styles.num}`}>Bruno</span>
          <span className="label">Collective</span>
        </div>
        <div className={styles.right}>
          <span className="label">Finished by hand · งานทำมือ</span>
        </div>
      </div>

      <div className={styles.scrollCue} aria-hidden>
        Scroll
        <span className={styles.line} />
      </div>
    </section>
  );
}
