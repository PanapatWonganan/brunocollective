import styles from "./MarqueeStrip.module.css";

const PHRASE = (
  <>
    Linen of Solbiate Olona <i className={styles.dot} /> Cashmere from Inner
    Mongolia <i className={styles.dot} /> Hand-finished in Solomeo{" "}
    <i className={styles.dot} /> Sea-island cotton from Barbados{" "}
    <i className={styles.dot} /> Italian box-calf leather{" "}
    <i className={styles.dot} /> Buttons in Corozo <i className={styles.dot} />{" "}
    One hundred and seventeen hours per coat <i className={styles.dot} />
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
