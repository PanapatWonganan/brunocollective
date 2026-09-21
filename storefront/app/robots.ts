import type { MetadataRoute } from "next";
import { SITE_URL } from "@/lib/site";

// Crawl rules. Transactional and account pages carry no search value and
// can hold personal data, so they are kept out of the index.
export default function robots(): MetadataRoute.Robots {
  return {
    rules: [
      {
        userAgent: "*",
        allow: "/",
        disallow: [
          "/cart",
          "/checkout",
          "/pay/",
          "/member",
          "/affiliate",
          "/admin",
          "/api/",
          "/data-deletion",
        ],
      },
    ],
    sitemap: `${SITE_URL}/sitemap.xml`,
    host: SITE_URL,
  };
}
