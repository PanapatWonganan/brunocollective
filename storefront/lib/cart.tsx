"use client";

import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import type { CartLine, Product, ProductVariant } from "./types";

const STORAGE_KEY = "bc_cart";

// A cart line is uniquely identified by its product AND chosen variant, so two
// sizes/colors of the same garment are distinct lines.
export function lineKey(productId: number, variantId: number | null): string {
  return `${productId}:${variantId ?? 0}`;
}

// Stock available for a line: the variant's stock when a variant is chosen,
// else the product-level (legacy) stock.
function availableStock(product: Product, variant: ProductVariant | null): number {
  return Math.max(variant ? variant.stock : product.stock, 0);
}

interface CartContextValue {
  lines: CartLine[];
  count: number;
  total: number;
  add: (product: Product, variant: ProductVariant | null, quantity?: number) => void;
  setQuantity: (key: string, quantity: number) => void;
  remove: (key: string) => void;
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
      if (raw) {
        // Tolerate carts saved before variants existed (no `variant` field).
        const parsed: CartLine[] = JSON.parse(raw).map((l: CartLine) => ({
          ...l,
          variant: l.variant ?? null,
        }));
        setLines(parsed);
      }
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

  function add(product: Product, variant: ProductVariant | null, quantity = 1) {
    const key = lineKey(product.id, variant ? variant.id : null);
    const cap = availableStock(product, variant);
    setLines((prev) => {
      const existing = prev.find(
        (l) => lineKey(l.product.id, l.variant ? l.variant.id : null) === key
      );
      if (existing) {
        const next = Math.min(existing.quantity + quantity, cap);
        return prev.map((l) =>
          lineKey(l.product.id, l.variant ? l.variant.id : null) === key
            ? { ...l, quantity: next }
            : l
        );
      }
      return [...prev, { product, variant, quantity: Math.min(quantity, cap) }];
    });
    setOpen(true);
  }

  function setQuantity(key: string, quantity: number) {
    setLines((prev) =>
      prev
        .map((l) =>
          lineKey(l.product.id, l.variant ? l.variant.id : null) === key
            ? {
                ...l,
                quantity: Math.max(
                  0,
                  Math.min(quantity, availableStock(l.product, l.variant))
                ),
              }
            : l
        )
        .filter((l) => l.quantity > 0)
    );
  }

  function remove(key: string) {
    setLines((prev) =>
      prev.filter(
        (l) => lineKey(l.product.id, l.variant ? l.variant.id : null) !== key
      )
    );
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
