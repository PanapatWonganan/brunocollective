"use client";

import { useState, useRef, useCallback, useEffect } from "react";
import { imageSrc } from "@/lib/format";
import styles from "./ProductGallery.module.css";

interface Props {
  images: string[];
  alt: string;
}

export default function ProductGallery({ images, alt }: Props) {
  const [active, setActive] = useState(0);
  const [lightbox, setLightbox] = useState(false);
  const [zoom, setZoom] = useState(false);
  const [origin, setOrigin] = useState({ x: 50, y: 50 });
  const frameRef = useRef<HTMLDivElement>(null);

  // Clamp the active index if the image list changes.
  const safeActive = Math.min(active, Math.max(images.length - 1, 0));
  const current = images[safeActive];

  const go = useCallback(
    (dir: number) => {
      setActive((i) => {
        const n = images.length;
        if (n === 0) return 0;
        return (i + dir + n) % n;
      });
    },
    [images.length]
  );

  // Hover-zoom: track the cursor over the main frame and shift the transform
  // origin so the image magnifies under the pointer (desktop / fine pointers).
  const onMove = useCallback((e: React.MouseEvent<HTMLDivElement>) => {
    const el = frameRef.current;
    if (!el) return;
    const r = el.getBoundingClientRect();
    const x = ((e.clientX - r.left) / r.width) * 100;
    const y = ((e.clientY - r.top) / r.height) * 100;
    setOrigin({ x, y });
  }, []);

  // Keyboard support for the lightbox: arrows navigate, Escape closes.
  useEffect(() => {
    if (!lightbox) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setLightbox(false);
      else if (e.key === "ArrowRight") go(1);
      else if (e.key === "ArrowLeft") go(-1);
    };
    window.addEventListener("keydown", onKey);
    // Prevent background scroll while the lightbox is open.
    const prev = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      window.removeEventListener("keydown", onKey);
      document.body.style.overflow = prev;
    };
  }, [lightbox, go]);

  if (images.length === 0) {
    return (
      <div className={styles.box}>
        <div className={styles.empty} />
      </div>
    );
  }

  return (
    <div className={styles.wrap}>
      <div
        ref={frameRef}
        className={styles.box}
        onMouseEnter={() => setZoom(true)}
        onMouseLeave={() => setZoom(false)}
        onMouseMove={onMove}
        onClick={() => setLightbox(true)}
        role="button"
        aria-label="Open image viewer"
        tabIndex={0}
        onKeyDown={(e) => {
          if (e.key === "Enter" || e.key === " ") {
            e.preventDefault();
            setLightbox(true);
          }
        }}
      >
        <div
          className={styles.img}
          style={{
            backgroundImage: `url('${imageSrc(current)}')`,
            transform: zoom ? "scale(1.8)" : "scale(1)",
            transformOrigin: `${origin.x}% ${origin.y}%`,
          }}
        />
        <span className={styles.hint}>Click to zoom</span>
      </div>

      {images.length > 1 && (
        <div className={styles.thumbs}>
          {images.map((img, i) => (
            <button
              key={img}
              className={`${styles.thumb} ${i === safeActive ? styles.thumbActive : ""}`}
              onClick={() => setActive(i)}
              aria-label={`View image ${i + 1}`}
            >
              <span
                className={styles.thumbImg}
                style={{ backgroundImage: `url('${imageSrc(img)}')` }}
              />
            </button>
          ))}
        </div>
      )}

      {lightbox && (
        <div className={styles.lightbox} onClick={() => setLightbox(false)}>
          <button
            className={styles.close}
            onClick={() => setLightbox(false)}
            aria-label="Close"
          >
            ×
          </button>
          {images.length > 1 && (
            <button
              className={`${styles.nav} ${styles.prev}`}
              onClick={(e) => {
                e.stopPropagation();
                go(-1);
              }}
              aria-label="Previous image"
            >
              ‹
            </button>
          )}
          <img
            className={styles.lightImg}
            src={imageSrc(current)}
            alt={alt}
            onClick={(e) => e.stopPropagation()}
          />
          {images.length > 1 && (
            <button
              className={`${styles.nav} ${styles.next}`}
              onClick={(e) => {
                e.stopPropagation();
                go(1);
              }}
              aria-label="Next image"
            >
              ›
            </button>
          )}
          {images.length > 1 && (
            <div className={styles.counter}>
              {safeActive + 1} / {images.length}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
