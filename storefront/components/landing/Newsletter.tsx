"use client";

import { useState } from "react";
import styles from "./Newsletter.module.css";

export default function Newsletter() {
  const [email, setEmail] = useState("");
  const [sent, setSent] = useState(false);

  function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setEmail("");
    setSent(true);
    setTimeout(() => setSent(false), 5200);
  }

  return (
    <section className={`${styles.news} section--tight`} id="newsletter">
      <div className={styles.inner}>
        <span className="kicker">Correspondence</span>
        <h2 className={`display ${styles.h2}`}>
          <em>Letters from</em> the house.
        </h2>
        <p className={styles.copy}>
          An occasional note from the atelier — new pieces, restocks, and the
          occasional essay. No noise, just the good stuff. ไม่บ่อย แต่คัดมาแล้ว.
        </p>

        <form className={styles.form} onSubmit={onSubmit} autoComplete="off">
          <input
            type="email"
            name="email"
            placeholder="Your address"
            required
            aria-label="Email address"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
          />
          <button type="submit" className={styles.submit}>
            Subscribe <span className="arrow">→</span>
          </button>
        </form>
        <div className={styles.fine}>
          No marketing. Unsubscribe with a single line.
        </div>
        <div className={`${styles.toast} ${sent ? styles.toastOn : ""}`}>
          Thank you — a note will be in your post by week&apos;s end.
        </div>
      </div>
    </section>
  );
}
