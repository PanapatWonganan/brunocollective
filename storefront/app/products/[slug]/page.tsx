import type { Metadata } from "next";
import { notFound, permanentRedirect } from "next/navigation";
import Link from "next/link";
import { getProduct, getRelated, getSiteImages, sizeChartFor } from "@/lib/api";
import { money, imageSrc } from "@/lib/format";
import AddToBag from "@/components/AddToBag";
import ProductGallery from "@/components/ProductGallery";
import Accordion from "@/components/Accordion";
import Rating from "@/components/Rating";
import ProductRow from "@/components/home/ProductRow";
import JsonLd from "@/components/JsonLd";
import { absoluteUrl, SITE_NAME } from "@/lib/site";
import { collectionPath, productPath } from "@/lib/paths";
import styles from "./product.module.css";

// Build the gallery list: prefer the multi-image array, fall back to the
// legacy single image_url, de-duplicated and stripped of blanks.
function galleryImages(product: { images?: string[]; image_url?: string }): string[] {
  const list = [...(product.images || [])];
  if (product.image_url && !list.includes(product.image_url)) {
    list.unshift(product.image_url);
  }
  return list.filter(Boolean);
}

interface Params {
  params: Promise<{ slug: string }>;
}

export async function generateMetadata({ params }: Params): Promise<Metadata> {
  const { slug } = await params;
  const product = await getProduct(slug).catch(() => null);
  if (!product) return { title: "Not found" };
  const canonical = productPath(product);
  return {
    title: product.name,
    description:
      product.description ||
      `${product.name} — cut and finished by hand at the Bruno Collective atelier in Thailand.`,
    alternates: { canonical },
    openGraph: {
      type: "website",
      url: canonical,
      title: product.name,
      description: product.description || product.name,
      images: product.image_url ? [imageSrc(product.image_url)] : undefined,
    },
  };
}

export default async function ProductPage({ params }: Params) {
  const { slug } = await params;
  const [product, siteImages] = await Promise.all([
    getProduct(slug).catch(() => null),
    getSiteImages(),
  ]);
  if (!product) notFound();
  // Someone hit the page by id or an old slug — send them to the canonical URL.
  if (product.slug && product.slug !== slug) permanentRedirect(productPath(product));
  const [related] = await Promise.all([getRelated(product.id, 4)]);
  const sizeChartUrl = sizeChartFor(product.category, siteImages);

  const stock = product.variants?.length ? product.total_stock : product.stock;

  // Structured data for rich results + AI answer engines: Product (price,
  // availability, brand) and the breadcrumb trail. No AggregateRating —
  // Google requires those to come from real customer reviews.
  const url = absoluteUrl(productPath(product));
  const images = galleryImages(product).map((i) => absoluteUrl(imageSrc(i)));
  const productLd = {
    "@context": "https://schema.org",
    "@type": "Product",
    name: product.name,
    description: product.description || undefined,
    sku: product.sku || undefined,
    image: images.length ? images : undefined,
    url,
    brand: { "@type": "Brand", name: SITE_NAME },
    category: product.category || undefined,
    offers: {
      "@type": "Offer",
      url,
      priceCurrency: "THB",
      price: product.price,
      availability:
        stock > 0 ? "https://schema.org/InStock" : "https://schema.org/OutOfStock",
      itemCondition: "https://schema.org/NewCondition",
      seller: { "@type": "Organization", name: SITE_NAME },
    },
  };
  const crumbs = [
    { name: "Home", item: absoluteUrl("/") },
    { name: "Shop", item: absoluteUrl("/shop") },
    ...(product.category
      ? [{ name: product.category, item: absoluteUrl(collectionPath(product.category)) }]
      : []),
    { name: product.name, item: url },
  ];
  const breadcrumbLd = {
    "@context": "https://schema.org",
    "@type": "BreadcrumbList",
    itemListElement: crumbs.map((c, i) => ({
      "@type": "ListItem",
      position: i + 1,
      name: c.name,
      item: c.item,
    })),
  };

  return (
    <main className={styles.page}>
      <JsonLd data={productLd} />
      <JsonLd data={breadcrumbLd} />
      <div className={styles.pdp}>
        <div className={styles.galCol}>
          <ProductGallery images={galleryImages(product)} alt={product.name} />
        </div>

        <div className={styles.detail}>
          <div className={styles.crumbs}>
            <Link href="/">Home</Link>
            <span>/</span>
            <Link href="/shop">Shop</Link>
            <span>/</span>
            {product.category ? (
              <Link href={collectionPath(product.category)}>
                {product.category}
              </Link>
            ) : (
              <span className={styles.crumbHere}>{product.name}</span>
            )}
          </div>

          <h1 className={styles.name}>{product.name}</h1>
          <div className={styles.sub}>
            {product.sku ? `${product.sku} — ` : ""}Bruno Collective · Made in Thailand
          </div>
          <div className={styles.priceRow}>
            <div className={styles.price}>{money(product.price)}</div>
            <Rating value={product.rating} count={product.rating_count} size="md" />
          </div>
          <div className={styles.tax}>ราคารวมทุกอย่างแล้ว — ไม่มีบวกเพิ่มหน้างาน</div>

          <AddToBag product={product} sizeChartUrl={sizeChartUrl} />

          <div className={styles.note}>
            Finished by hand in Thailand — แพ็คอย่างดี ส่งไวทั่วไทย
          </div>

          <Accordion
            items={[
              {
                title: "The Piece — รายละเอียด",
                open: true,
                content: (
                  <>
                    {product.description ? (
                      <p>{product.description}</p>
                    ) : (
                      <p>
                        Cut and finished by hand at our studio in Thailand —
                        considered fabric, clean lines, made in a limited run.
                      </p>
                    )}
                    <ul>
                      {product.sku && <li>Reference — {product.sku}</li>}
                      {product.size && !(product.variants?.length ?? 0) && (
                        <li>Size — {product.size}</li>
                      )}
                      <li>
                        {stock > 0
                          ? `In stock — เหลือ ${stock} ชิ้นในรอบนี้`
                          : "Sold out — หมดรอบนี้แล้ว"}
                      </li>
                      <li>Made &amp; finished by hand in Thailand</li>
                    </ul>
                  </>
                ),
              },
              {
                title: "Care — การดูแลรักษา",
                content: (
                  <p>
                    ซักเบา ๆ ตากในที่ร่ม รีดไฟอ่อนด้านใน —
                    treated gently, this piece will keep its shape and colour
                    for years of wear.
                  </p>
                ),
              },
              {
                title: "Shipping & Exchange — จัดส่ง/เปลี่ยนไซส์",
                content: (
                  <p>
                    จัดส่งทั่วไทยพร้อมเลขติดตามทุกออเดอร์ —
                    หากไซส์ไม่พอดี ทักแชทหาเราทาง LINE / Facebook / Instagram
                    เพื่อเปลี่ยนไซส์ได้เลย เราตอบเองทุกแชท
                  </p>
                ),
              },
            ]}
          />

          <div className={styles.vicons}>
            <span className={styles.vi}>Hand-finished</span>
            <span className={styles.vi}>Limited run</span>
            <span className={styles.vi}>Easy exchange</span>
          </div>
        </div>
      </div>

      {related.length > 0 && (
        <div className={styles.also}>
          <ProductRow
            bare
            title={
              <>
                You may also <em>consider.</em>
              </>
            }
            products={related}
          />
        </div>
      )}
    </main>
  );
}
