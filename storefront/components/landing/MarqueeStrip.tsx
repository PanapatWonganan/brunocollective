import styles from "./MarqueeStrip.module.css";

const PHRASE = (
  <>
    Designed in Khon Kaen <i className={styles.dot} /> Cut &amp; finished by hand{" "}
    <i className={styles.dot} /> Considered fabrics, chosen with care{" "}
    <i className={styles.dot} /> Limited runs <i className={styles.dot} /> Quiet
    luxury, made in Thailand <i className={styles.dot} /> เสื้อผ้าคุณภาพ ตัดเย็บในไทย{" "}
    <i className={styles.dot} /> Made to be worn for years <i className={styles.dot} />
  </>
);

export default function MarqueeStrip() {
  return (
    <div className={styles.strip} aria-hidden>
      <div className={styles.inner}>
        <span className={styles.run}>{PHRASE}</span>
        <span className={styles.run}>{PHRASE}</span>
      </div>
    </div>
  );
}
