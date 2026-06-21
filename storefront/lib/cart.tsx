"use client";

import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import type { CartLine, Product } from "./types";

const STORAGE_KEY = "bc_cart";

interface CartContextValue {
  lines: CartLine[];
  count: number;
  total: number;
  add: (product: Product, quantity?: number) => void;
  setQuantity: (productId: number, quantity: number) => void;
  remove: (productId: number) => void;
  clear: () => void;
  open: boolean;
  setOpen: (open: boolean) => void;
}

const CartContext = createContext<CartContextValue | null>(null);

export function CartProvider({ children }: { children: ReactNode }) {
  const [lines, setLines] = useState<CartLine[]>([]);
  const [open, setOpen] = useState(false);
  const [hydrated, setHydrated] = useState(false);

  // Load from localStorage once on mount.
  useEffect(() => {
    try {
      const raw = localStorage.getItem(STORAGE_KEY);
      if (raw) setLines(JSON.parse(raw));
    } catch {
      /* ignore malformed cart */
    }
    setHydrated(true);
  }, []);

  // Persist on change (after initial hydration).
  useEffect(() => {
    if (!hydrated) return;
    localStorage.setItem(STORAGE_KEY, JSON.stringify(lines));
  }, [lines, hydrated]);

  function add(product: Product, quantity = 1) {
    setLines((prev) => {
      const existing = prev.find((l) => l.product.id === product.id);
      const cap = Math.max(product.stock, 0);
      if (existing) {
        const next = Math.min(existing.quantity + quantity, cap);
        return prev.map((l) =>
          l.product.id === product.id ? { ...l, quantity: next } : l
        );
      }
      return [...prev, { product, quantity: Math.min(quantity, cap) }];
    });
    setOpen(true);
  }

  function setQuantity(productId: number, quantity: number) {
    setLines((prev) =>
      prev
        .map((l) =>
          l.product.id === productId
            ? {
                ...l,
                quantity: Math.max(0, Math.min(quantity, l.product.stock)),
              }
            : l
        )
        .filter((l) => l.quantity > 0)
    );
  }

  function remove(productId: number) {
    setLines((prev) => prev.filter((l) => l.product.id !== productId));
  }

  function clear() {
    setLines([]);
  }

  const value = useMemo<CartContextValue>(() => {
    const count = lines.reduce((n, l) => n + l.quantity, 0);
    const total = lines.reduce((n, l) => n + l.quantity * l.product.price, 0);
    return { lines, count, total, add, setQuantity, remove, clear, open, setOpen };
  }, [lines, open]);

  return <CartContext.Provider value={value}>{children}</CartContext.Provider>;
}

export function useCart(): CartContextValue {
  const ctx = useContext(CartContext);
  if (!ctx) throw new Error("useCart must be used within CartProvider");
  return ctx;
}
