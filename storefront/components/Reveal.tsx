"use client";

import { useEffect, useRef, useState, type ReactNode } from "react";

// Mirrors the design's scroll-reveal: fade + rise once in view.
export default function Reveal({
  children,
  delay,
  className = "",
  as: Tag = "div",
}: {
  children: ReactNode;
  delay?: 2 | 3 | 4;
  className?: string;
  as?: "div" | "figure" | "article" | "p" | "span";
}) {
  const ref = useRef<HTMLElement | null>(null);
  const [shown, setShown] = useState(false);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    const io = new IntersectionObserver(
      (entries) => {
        for (const e of entries) {
          if (e.isIntersecting) {
            setShown(true);
            io.unobserve(e.target);
          }
        }
      },
      { rootMargin: "-8% 0px -8% 0px", threshold: 0.05 }
    );
    io.observe(el);
    return () => io.disconnect();
  }, []);

  const cls = [
    "reveal",
    delay ? `reveal-d${delay}` : "",
    shown ? "is-in" : "",
    className,
  ]
    .filter(Boolean)
    .join(" ");

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const Comp = Tag as any;
  return (
    <Comp ref={ref} className={cls}>
      {children}
    </Comp>
  );
}
