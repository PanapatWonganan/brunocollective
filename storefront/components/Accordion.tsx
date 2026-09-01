"use client";

import { useState, type ReactNode } from "react";
import styles from "./Accordion.module.css";

export interface AccordionItem {
  title: string;
  content: ReactNode;
  open?: boolean;
}

// Product-page accordion (design: .acc) — hairline rows with a rotating "+".
export default function Accordion({ items }: { items: AccordionItem[] }) {
  const [open, setOpen] = useState<boolean[]>(items.map((i) => !!i.open));

  return (
    <div className={styles.acc}>
      {items.map((item, i) => (
        <div key={item.title} className={styles.item}>
          <button
            type="button"
            className={styles.ah}
            aria-expanded={open[i]}
            onClick={() =>
              setOpen((prev) => prev.map((v, j) => (j === i ? !v : v)))
            }
          >
            {item.title}
            <span className={`${styles.car} ${open[i] ? styles.carOpen : ""}`}>+</span>
          </button>
          <div className={styles.ab} hidden={!open[i]}>
            {item.content}
          </div>
        </div>
      ))}
    </div>
  );
}
