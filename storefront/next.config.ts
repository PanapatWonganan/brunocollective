import type { NextConfig } from "next";

// Backend origin for server-side fetches and dev rewrites.
const BACKEND = process.env.BACKEND_URL || "http://localhost:8080";

const nextConfig: NextConfig = {
  async rewrites() {
    // Proxy API and uploaded images to the Go backend so the storefront can be
    // served from a single origin (mirrors the Vite dev proxy used by the admin app).
    // In production Nginx routes /api and /uploads to the backend before they
    // ever reach Next, so these rewrites effectively only matter in dev.
    return [
      { source: "/api/:path*", destination: `${BACKEND}/api/:path*` },
      { source: "/uploads/:path*", destination: `${BACKEND}/uploads/:path*` },
    ];
  },
  images: {
    remotePatterns: [
      { protocol: "https", hostname: "images.unsplash.com" },
      { protocol: "https", hostname: "images.pexels.com" },
    ],
  },
};

export default nextConfig;
