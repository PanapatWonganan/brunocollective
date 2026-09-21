import type { Metadata } from "next";

// Transactional / account page — keep out of search indexes.
export const metadata: Metadata = { robots: { index: false, follow: false } };

export default function Layout({ children }: { children: React.ReactNode }) {
  return children;
}
