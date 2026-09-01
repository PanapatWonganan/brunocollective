import type { Metadata } from "next";
import { notFound } from "next/navigation";
import Link from "next/link";
import { getProduct, getRelated, getSiteImages, sizeChartFor } from "@/lib/api";
import { money, imageSrc } from "@/lib/format";
import AddToBag from "@/components/AddToBag";
import ProductGallery from "@/components/ProductGallery";
import Accordion from "@/components/Accordion";
import ProductRow from "@/components/home/ProductRow";
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
  params: Promise<{ id: string }>;
}

export async function generateMetadata({ params }: Params): Promise<Metadata> {
  const { id } = await params;
  const product = await getProduct(id).catch(() => null);
  if (!product) return { title: "Not found" };
  return {
    title: product.name,
    description:
      product.description ||
      `${product.name} — cut and finished by hand at the Bruno Collective atelier in Thailand.`,
    openGraph: {
      title: product.name,
      description: product.description || product.name,
      images: product.image_url ? [imageSrc(product.image_url)] : undefined,
    },
  };
}

export default async function ProductPage({ params }: Params) {
  const { id } = await params;
  const [product, siteImages] = await Promise.all([
    getProduct(id).catch(() => null),
    getSiteImages(),
  ]);
  if (!product) notFound();
  const [related] = await Promise.all([getRelated(product.id, 4)]);
  const sizeChartUrl = sizeChartFor(product.category, siteImages);

  const stock = product.variants?.length ? product.total_stock : product.stock;

  return (
    <main className={styles.page}>
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
              <Link href={`/shop?cat=${encodeURIComponent(product.category)}`}>
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
          <div className={styles.price}>{money(product.price)}</div>
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
