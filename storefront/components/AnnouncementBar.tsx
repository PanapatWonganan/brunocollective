"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useMember } from "@/lib/member";
import styles from "./AnnouncementBar.module.css";

// Bump the key to relaunch the campaign for visitors who dismissed an
// earlier one (dismissal is remembered per browser).
const DISMISS_KEY = "bc_promo_dismissed_member5";
const BAR_HEIGHT = "38px";
const ROTATE_MS = 4200;

// Rotating announcement strip (design: .annc) fixed above the top bar.
// Hidden for logged-in members (the lead message is the member campaign)
// and for visitors who closed it.
export default function AnnouncementBar() {
  const { member, ready } = useMember();
  const [dismissed, setDismissed] = useState(true); // true until checked — no flash
  const [active, setActive] = useState(0);

  const messages = [
    <Link key="member" href="/member" className={styles.msgLink}>
      สมัครสมาชิกวันนี้ — รับส่วนลด <em>5%</em> ทุกออเดอร์{" "}
      <span className={styles.cta}>สมัครเลย →</span>
    </Link>,
    <span key="made">
      Cut &amp; finished by hand in <em>Thailand</em> — ตัดเย็บประณีตทุกชิ้น
    </span>,
    <span key="runs">
      Limited runs — <em>ผลิตจำนวนจำกัด</em> หมดแล้วหมดเลย
    </span>,
  ];

  useEffect(() => {
    setDismissed(localStorage.getItem(DISMISS_KEY) === "1");
  }, []);

  const visible = ready && !member && !dismissed;

  // Rotate through the messages while visible.
  useEffect(() => {
    if (!visible) return;
    const t = setInterval(
      () => setActive((i) => (i + 1) % messages.length),
      ROTATE_MS
    );
    return () => clearInterval(t);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [visible]);

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
    <div className={styles.bar} role="region" aria-label="Announcements">
      {messages.map((m, i) => (
        <div key={i} className={`${styles.msg} ${i === active ? styles.msgOn : ""}`}>
          {m}
        </div>
      ))}
      <button
        className={styles.close}
        aria-label="Close announcements"
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
