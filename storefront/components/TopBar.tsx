"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useState } from "react";
import { useCart } from "@/lib/cart";
import styles from "./TopBar.module.css";

export default function TopBar() {
  const pathname = usePathname();
  const { count, setOpen } = useCart();
  // Only the landing page has the dark full-bleed hero behind the bar.
  const overHero = pathname === "/";
  const [scrolled, setScrolled] = useState(!overHero);

  useEffect(() => {
    if (!overHero) {
      setScrolled(true);
      return;
    }
    const onScroll = () => setScrolled(window.scrollY > 40);
    window.addEventListener("scroll", onScroll, { passive: true });
    onScroll();
    return () => window.removeEventListener("scroll", onScroll);
  }, [overHero]);

  return (
    <header
      className={`${styles.topbar} ${scrolled ? styles.scrolled : styles.transparent}`}
    >
      <nav className={styles.left} aria-label="Primary">
        <Link href="/shop">Collection</Link>
        <Link href="/#atelier">Atelier</Link>
        <Link href="/#lookbook">Lookbook</Link>
        <Link href="/#journal">Journal</Link>
      </nav>

      <Link className={styles.brand} href="/" aria-label="Bruno Collective home">
        <span className={styles.monogram}>BC</span>
        <span>Bruno&nbsp;Collective</span>
      </Link>

      <nav className={styles.right} aria-label="Secondary">
        <Link href="/#boutiques">Visit</Link>
        <Link href="/#contact">Contact</Link>
        <Link href="/shop" className={styles.icons} aria-label="Account">
          <svg viewBox="0 0 24 24">
            <circle cx="12" cy="8" r="3.6" />
            <path d="M4.5 20c1.2-3.6 4.2-5.6 7.5-5.6S18.3 16.4 19.5 20" />
          </svg>
        </Link>
        <button
          className={styles.icons}
          aria-label="Bag"
          onClick={() => setOpen(true)}
        >
          <svg viewBox="0 0 24 24">
            <path d="M5 8h14l-1.2 12H6.2L5 8z" />
            <path d="M9 8V6.5a3 3 0 0 1 6 0V8" />
          </svg>
          {count > 0 && <span className={styles.badge}>{count}</span>}
        </button>
      </nav>

      <button
        className={styles.menubtn}
        aria-label="Bag"
        onClick={() => setOpen(true)}
      >
        <span className={styles.bar} />
        <span className={styles.bar} />
        <span className={styles.bar} />
        Bag{count > 0 ? ` (${count})` : ""}
      </button>
    </header>
  );
}
