"use client";

import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { AFFILIATE_TOKEN_KEY, affiliateMe, type AffiliateProfile } from "./api";

// Affiliate portal auth context — clone of the member provider, scoped to the
// /affiliate pages only (mounted in app/affiliate/layout.tsx).

interface AffiliateContextValue {
  affiliate: AffiliateProfile | null;
  ready: boolean;
  signIn: (token: string, affiliate: AffiliateProfile) => void;
  signOut: () => void;
}

const AffiliateContext = createContext<AffiliateContextValue | null>(null);

export function AffiliateProvider({ children }: { children: ReactNode }) {
  const [affiliate, setAffiliate] = useState<AffiliateProfile | null>(null);
  const [ready, setReady] = useState(false);

  // Validate the stored token once on mount; a rejected token is stale.
  useEffect(() => {
    let cancelled = false;
    (async () => {
      if (!localStorage.getItem(AFFILIATE_TOKEN_KEY)) {
        setReady(true);
        return;
      }
      const stats = await affiliateMe();
      if (cancelled) return;
      if (stats?.affiliate) {
        setAffiliate(stats.affiliate);
      } else {
        localStorage.removeItem(AFFILIATE_TOKEN_KEY);
      }
      setReady(true);
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const value = useMemo<AffiliateContextValue>(
    () => ({
      affiliate,
      ready,
      signIn: (token, profile) => {
        localStorage.setItem(AFFILIATE_TOKEN_KEY, token);
        setAffiliate(profile);
      },
      signOut: () => {
        localStorage.removeItem(AFFILIATE_TOKEN_KEY);
        setAffiliate(null);
      },
    }),
    [affiliate, ready]
  );

  return <AffiliateContext.Provider value={value}>{children}</AffiliateContext.Provider>;
}

export function useAffiliate(): AffiliateContextValue {
  const ctx = useContext(AffiliateContext);
  if (!ctx) throw new Error("useAffiliate must be used within AffiliateProvider");
  return ctx;
}

// ---- ?ref attribution (30-day, last-click) ----
// Plain helpers usable outside the provider (checkout, sale pages, capture).
// Storage errors are swallowed so attribution never breaks a page.

const AFF_REF_KEY = "bc_aff_ref";
const REF_TTL_MS = 30 * 24 * 60 * 60 * 1000;

export function saveAffiliateRef(code: string) {
  try {
    localStorage.setItem(
      AFF_REF_KEY,
      JSON.stringify({ code: code.trim().toUpperCase(), exp: Date.now() + REF_TTL_MS })
    );
  } catch {
    /* ignore */
  }
}

export function getAffiliateRef(): string | null {
  try {
    const raw = localStorage.getItem(AFF_REF_KEY);
    if (!raw) return null;
    const { code, exp } = JSON.parse(raw);
    if (!code || !exp || Date.now() > exp) {
      localStorage.removeItem(AFF_REF_KEY);
      return null;
    }
    return code;
  } catch {
    return null;
  }
}
