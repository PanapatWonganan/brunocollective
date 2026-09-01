"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useState } from "react";
import { useCart } from "@/lib/cart";
import { imageSrc } from "@/lib/format";
import styles from "./TopBar.module.css";

// Featured product for the mega-menu image tile (admin-uploaded photo).
export interface NavFeatured {
  id: number;
  name: string;
  image: string;
}

const NAV_LINKS = [
  { href: "/shop", label: "Shop" },
  { href: "/shop?sort=new", label: "New In" },
  { href: "/story", label: "Editorial" },
  { href: "/member", label: "Membership" },
  { href: "/service", label: "Client Services" },
];

export default function TopBar({
  categories = [],
  featured = null,
}: {
  categories?: string[];
  featured?: NavFeatured | null;
}) {
  const pathname = usePathname();
  const { count, setOpen } = useCart();
  const [menuOpen, setMenuOpen] = useState(false);
  // Pages with a dark full-bleed hero behind the bar start transparent.
  const overHero = pathname === "/" || pathname === "/story";
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

  // Close the mobile menu whenever the route changes.
  useEffect(() => {
    setMenuOpen(false);
  }, [pathname]);

  return (
    <>
      <header
        className={`${styles.topbar} ${scrolled ? styles.scrolled : styles.transparent}`}
      >
        <nav className={styles.left} aria-label="Primary">
          <div className={styles.navItem}>
            <Link href="/shop">Shop</Link>
            {/* Mega menu — categories from the live catalogue */}
            <div className={styles.mega}>
              <div>
                <h5 className={styles.megaH}>Categories</h5>
                <ul className={styles.megaUl}>
                  <li>
                    <Link href="/shop">All Pieces</Link>
                  </li>
                  {categories.slice(0, 6).map((c) => (
                    <li key={c}>
                      <Link href={`/shop?cat=${encodeURIComponent(c)}`}>{c}</Link>
                    </li>
                  ))}
                </ul>
              </div>
              <div>
                <h5 className={styles.megaH}>This Season</h5>
                <ul className={styles.megaUl}>
                  <li>
                    <Link href="/shop?sort=new">New Arrivals</Link>
                  </li>
                  <li>
                    <Link href="/shop">Best Sellers</Link>
                  </li>
                  <li>
                    <Link href="/member">Member — ส่วนลด 5%</Link>
                  </li>
                </ul>
              </div>
              {featured && featured.image && (
                <div className={styles.megaFeat}>
                  <Link href={`/product/${featured.id}`}>
                    <div
                      className={styles.megaImg}
                      style={{ backgroundImage: `url('${imageSrc(featured.image)}')` }}
                    />
                    <div className={styles.megaFt}>{featured.name}</div>
                  </Link>
                </div>
              )}
            </div>
          </div>
          <div className={styles.navItem}>
            <Link href="/shop?sort=new">New In</Link>
          </div>
          <div className={styles.navItem}>
            <Link href="/story">Editorial</Link>
          </div>
        </nav>

        <button
          className={styles.menubtn}
          aria-label="Open menu"
          aria-expanded={menuOpen}
          onClick={() => setMenuOpen(true)}
        >
          <span className={styles.bar} />
          <span className={styles.bar} />
          <span className={styles.bar} />
          Menu
        </button>

        <Link className={styles.brand} href="/" aria-label="Bruno Collective home">
          <span className={styles.monogram}>BC</span>
          <span>Bruno&nbsp;Collective</span>
        </Link>

        <nav className={styles.right} aria-label="Secondary">
          <Link href="/service#contact">Contact</Link>
          <Link href="/member" className={styles.memberBtn} aria-label="Member account">
            <svg viewBox="0 0 24 24">
              <circle cx="12" cy="8" r="3.6" />
              <path d="M4.5 20c1.2-3.6 4.2-5.6 7.5-5.6S18.3 16.4 19.5 20" />
            </svg>
            Member
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
          className={styles.bagbtn}
          aria-label="Bag"
          onClick={() => setOpen(true)}
        >
          <svg viewBox="0 0 24 24">
            <path d="M5 8h14l-1.2 12H6.2L5 8z" />
            <path d="M9 8V6.5a3 3 0 0 1 6 0V8" />
          </svg>
          {count > 0 && <span className={styles.badge}>{count}</span>}
        </button>
      </header>

      {/* Mobile navigation menu */}
      <div
        className={`${styles.menuScrim} ${menuOpen ? styles.menuScrimOn : ""}`}
        onClick={() => setMenuOpen(false)}
        aria-hidden={!menuOpen}
      />
      <aside
        className={`${styles.menuPanel} ${menuOpen ? styles.menuPanelOn : ""}`}
        aria-label="Menu"
        aria-hidden={!menuOpen}
      >
        <div className={styles.menuHead}>
          <span className={styles.menuBrand}>Bruno&nbsp;Collective</span>
          <button
            className={styles.menuClose}
            onClick={() => setMenuOpen(false)}
            aria-label="Close menu"
          >
            ✕
          </button>
        </div>
        <nav className={styles.menuLinks} aria-label="Mobile">
          {NAV_LINKS.map((l) => (
            <Link key={l.href} href={l.href} onClick={() => setMenuOpen(false)}>
              {l.label}
            </Link>
          ))}
          {categories.length > 0 && (
            <div className={styles.menuCats}>
              {categories.slice(0, 6).map((c) => (
                <Link
                  key={c}
                  href={`/shop?cat=${encodeURIComponent(c)}`}
                  onClick={() => setMenuOpen(false)}
                >
                  {c}
                </Link>
              ))}
            </div>
          )}
        </nav>
      </aside>
    </>
  );
}
