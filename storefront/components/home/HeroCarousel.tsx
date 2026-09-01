"use client";

import Link from "next/link";
import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { imageSrc } from "@/lib/format";
import type { SiteImage } from "@/lib/api";
import s from "./home.module.css";

// Built-in slides — each can be overridden (image + captions) from the admin
// "Site Images" page via the home_hero_1..3 slots. Slide 3 only appears when
// its slot has an image.
const DEFAULTS = [
  {
    img: "/design/hero-polo-dunes.jpg",
    kicker: "Bruno Collective — Khon Kaen, Thailand",
    headline: "The quiet *uniform.*",
    cta: "Shop the Collection",
    href: "/shop",
  },
  {
    img: "/design/hero-black-tee-villa.jpg",
    kicker: "Everyday Essentials",
    headline: "One tee, *worn for years.*",
    cta: "Discover",
    href: "/shop",
  },
  { img: "", kicker: "", headline: "", cta: "Explore", href: "/shop" },
];

// Renders "*words*" in a headline as an italic champagne segment, mirroring
// the design's <em> spans.
function renderHeadline(text: string): ReactNode[] {
  return text.split(/\*([^*]+)\*/g).map((part, i) =>
    i % 2 === 1 ? <em key={i}>{part}</em> : <span key={i}>{part}</span>
  );
}

export default function HeroCarousel({ site }: { site?: Record<string, SiteImage> }) {
  const slides = DEFAULTS.map((d, i) => {
    const slot = site?.[`home_hero_${i + 1}`];
    return {
      ...d,
      img: slot?.image_url ? imageSrc(slot.image_url) : d.img,
      kicker: slot?.caption_a || d.kicker,
      headline: slot?.caption_b || d.headline,
    };
  }).filter((sl) => sl.img);

  const [active, setActive] = useState(0);
  // Slides after the first only get their (large) background image once the
  // page is interactive, so the first paint downloads a single hero image.
  const [ready, setReady] = useState(false);
  useEffect(() => setReady(true), []);
  const timer = useRef<ReturnType<typeof setInterval> | null>(null);

  const restart = useCallback(() => {
    if (timer.current) clearInterval(timer.current);
    if (slides.length < 2) return;
    timer.current = setInterval(
      () => setActive((i) => (i + 1) % slides.length),
      6000
    );
  }, [slides.length]);

  useEffect(() => {
    restart();
    return () => {
      if (timer.current) clearInterval(timer.current);
    };
  }, [restart]);

  if (slides.length === 0) return null;
  const safe = Math.min(active, slides.length - 1);

  return (
    <section className={s.carousel} aria-label="Featured">
      {slides.map((sl, i) => (
        <div key={i} className={`${s.slide} ${i === safe ? s.slideOn : ""}`}>
          <div
            className={s.slideBg}
            style={
              i === 0 || ready
                ? { backgroundImage: `url('${sl.img}')` }
                : undefined
            }
            aria-hidden
          />
          <div className={s.slideScrim} aria-hidden />
          <div className={s.slideTxt}>
            {sl.kicker && <span className={`kicker ${s.slideKicker}`}>{sl.kicker}</span>}
            {sl.headline && (
              <h2 className={s.slideH}>{renderHeadline(sl.headline)}</h2>
            )}
            <Link href={sl.href} className={`qlink ${s.slideLink}`}>
              {sl.cta} <span className="arrow">→</span>
            </Link>
          </div>
        </div>
      ))}
      {slides.length > 1 && (
        <div className={s.dots}>
          {slides.map((_, i) => (
            <button
              key={i}
              className={i === safe ? s.dotOn : ""}
              aria-label={`Slide ${i + 1}`}
              onClick={() => {
                setActive(i);
                restart();
              }}
            />
          ))}
        </div>
      )}
    </section>
  );
}
