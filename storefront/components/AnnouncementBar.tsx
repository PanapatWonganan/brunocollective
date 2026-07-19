"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useMember } from "@/lib/member";
import styles from "./AnnouncementBar.module.css";

// Bump the key to relaunch the campaign for visitors who dismissed an
// earlier one (dismissal is remembered per browser).
const DISMISS_KEY = "bc_promo_dismissed_member5";
const BAR_HEIGHT = "38px";

// Membership campaign strip fixed above the top bar. Hidden for logged-in
// members (they already have the discount) and for visitors who closed it.
export default function AnnouncementBar() {
  const { member, ready } = useMember();
  const [dismissed, setDismissed] = useState(true); // true until checked — no flash

  useEffect(() => {
    setDismissed(localStorage.getItem(DISMISS_KEY) === "1");
  }, []);

  const visible = ready && !member && !dismissed;

  // The top bar and page content offset themselves by --promo-h, so the strip
  // pushes the layout down instead of covering it.
  useEffect(() => {
    document.documentElement.style.setProperty("--promo-h", visible ? BAR_HEIGHT : "0px");
    return () => {
      document.documentElement.style.setProperty("--promo-h", "0px");
    };
  }, [visible]);

  if (!visible) return null;

  return (
    <div className={styles.bar} role="region" aria-label="Promotion">
      <Link href="/member" className={styles.msg}>
        สมัครสมาชิกวันนี้ — รับส่วนลด <b>5%</b> ทุกออเดอร์
        <span className={styles.cta}>
          สมัครเลย <span className="arrow">→</span>
        </span>
      </Link>
      <button
        className={styles.close}
        aria-label="Close promotion"
        onClick={() => {
          localStorage.setItem(DISMISS_KEY, "1");
          setDismissed(true);
        }}
      >
        ✕
      </button>
    </div>
  );
}
