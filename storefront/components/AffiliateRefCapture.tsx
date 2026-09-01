"use client";

import { useEffect } from "react";
import { useSearchParams } from "next/navigation";
import { saveAffiliateRef } from "@/lib/affiliate";

// Renders nothing — watches every navigation for ?ref=CODE, remembers the
// code for 30 days (last click wins) and fires a click-track ping. Mounted in
// the root layout inside <Suspense> (useSearchParams requires a boundary).
export default function AffiliateRefCapture() {
  const params = useSearchParams();
  const ref = params.get("ref");

  useEffect(() => {
    if (!ref || !ref.trim()) return;
    saveAffiliateRef(ref);
    // Fire-and-forget click counter — errors are irrelevant to the shopper.
    fetch("/api/shop/affiliates/track", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ code: ref.trim().toUpperCase() }),
    }).catch(() => {});
  }, [ref]);

  return null;
}
