import styles from "./Rating.module.css";

// Star rating shown on product cards and the product page. The value comes
// from the backend (derived from real sales — see handlers/rating.go) and
// is hidden entirely when the product has no sales yet.
export default function Rating({
  value,
  count,
  size = "sm",
}: {
  value: number;
  count: number;
  size?: "sm" | "md";
}) {
  // value is 0 when the product has no sales and no admin override → hidden.
  // An admin-pinned rating shows even before the first sale (count omitted).
  if (!value) return null;
  const pct = Math.max(0, Math.min(100, (value / 5) * 100));
  const label =
    count > 0
      ? `${value.toFixed(1)} จาก 5 ดาว จากยอดขาย ${count} ชิ้น`
      : `${value.toFixed(1)} จาก 5 ดาว`;

  return (
    <span className={`${styles.rating} ${size === "md" ? styles.md : ""}`} aria-label={label} title={label}>
      <span className={styles.stars} aria-hidden="true">
        <span className={styles.base}>★★★★★</span>
        <span className={styles.fill} style={{ width: `${pct}%` }}>★★★★★</span>
      </span>
      <span className={styles.text} aria-hidden="true">
        {value.toFixed(1)}
        {count > 0 && <span className={styles.count}> ({count})</span>}
      </span>
    </span>
  );
}
