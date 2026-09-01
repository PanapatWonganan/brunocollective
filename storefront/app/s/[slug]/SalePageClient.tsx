"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import Link from "next/link";
import Reveal from "@/components/Reveal";
import type { SalePage, SalePageSection, ProductVariant } from "@/lib/types";
import type { CouponPreview } from "@/lib/types";
import { salePageOrder, validateCoupon } from "@/lib/api";
import { getAffiliateRef } from "@/lib/affiliate";
import { money, imageSrc } from "@/lib/format";
import styles from "./salepage.module.css";

interface Props {
  page: SalePage;
  isPreview: boolean;
  sizeChartUrl?: string;
}

// Live countdown to a deadline. Returns null once passed (or when no deadline).
function useCountdown(endsAt: string | null) {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    if (!endsAt) return;
    const t = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(t);
  }, [endsAt]);
  if (!endsAt) return { active: false, ended: false, parts: null as null | { d: number; h: number; m: number; s: number } };
  const diff = new Date(endsAt).getTime() - now;
  if (diff <= 0) return { active: false, ended: true, parts: null };
  const d = Math.floor(diff / 86400000);
  const h = Math.floor((diff % 86400000) / 3600000);
  const m = Math.floor((diff % 3600000) / 60000);
  const s = Math.floor((diff % 60000) / 1000);
  return { active: true, ended: false, parts: { d, h, m, s } };
}

const pad2 = (n: number) => String(n).padStart(2, "0");

// Italicise the final word of a headline — the site's signature serif-em
// accent, applied automatically so the owner just types plain text.
function emphasize(text: string) {
  const words = text.trim().split(/\s+/);
  if (words.length < 2) return <em>{text}</em>;
  const last = words.pop();
  return (
    <>
      {words.join(" ")} <em>{last}</em>
    </>
  );
}

// English kickers per section — the small editorial label above each title.
const SECTION_KICKERS: Record<string, string> = {
  pain: "The Problem",
  story: "Our Story",
  showcase: "The Details",
  offer: "The Offer",
  testimonials: "Loved by Customers",
  guarantee: "Our Promise",
  faq: "Questions",
};

export default function SalePageClient({ page, isPreview, sizeChartUrl }: Props) {
  const product = page.product;
  const variants = (product.variants || []).filter(Boolean);
  const hasVariants = variants.length > 0;
  // Inline size chart in the order form — only when the product actually comes
  // in sizes and the admin uploaded a chart (Site Images → size_chart slot).
  const showSizeChart =
    !!sizeChartUrl &&
    (variants.some((v) => v.size) || (!hasVariants && !!product.size));

  const productImages: string[] = useMemo(
    () =>
      [
        ...(product.image_url ? [product.image_url] : []),
        ...(product.images || []),
      ].filter((v, i, a) => v && a.indexOf(v) === i),
    [product]
  );

  const [variantId, setVariantId] = useState<number | null>(null);
  const [quantity, setQuantity] = useState(1);
  const [bump, setBump] = useState(false);
  const [bumpVariantId, setBumpVariantId] = useState<number | null>(null);
  const [step, setStep] = useState<1 | 2>(1);
  const [form, setForm] = useState({ name: "", phone: "", email: "", address: "", notes: "" });
  const [slip, setSlip] = useState<File | null>(null);
  const [slipPreview, setSlipPreview] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [orderId, setOrderId] = useState<number | null>(null);
  const fileInput = useRef<HTMLInputElement>(null);
  const orderRef = useRef<HTMLDivElement>(null);

  // Sticky buy bar: shown once the visitor scrolls past the hero, hidden again
  // while the order form itself is on screen (no duplicate CTA).
  const [pastHero, setPastHero] = useState(false);
  const [orderVisible, setOrderVisible] = useState(false);
  useEffect(() => {
    const onScroll = () => setPastHero(window.scrollY > window.innerHeight * 0.6);
    onScroll();
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => window.removeEventListener("scroll", onScroll);
  }, []);
  useEffect(() => {
    const el = orderRef.current;
    if (!el) return;
    const io = new IntersectionObserver(
      (entries) => setOrderVisible(entries[0]?.isIntersecting ?? false),
      { threshold: 0.05 }
    );
    io.observe(el);
    return () => io.disconnect();
  }, [orderId]);

  // Coupon
  const [couponCode, setCouponCode] = useState("");
  const [coupon, setCoupon] = useState<CouponPreview | null>(null);
  const [couponError, setCouponError] = useState<string | null>(null);
  const [couponChecking, setCouponChecking] = useState(false);

  const countdown = useCountdown(page.countdown_ends_at);
  const offerEnded = countdown.ended;

  const unitPrice = page.offer_price ?? product.price;
  const catalogPrice = product.price;
  const hasDiscountedOffer = page.offer_price != null && page.offer_price < catalogPrice;

  const selectedVariant: ProductVariant | null =
    variants.find((v) => v.id === variantId) || null;
  const stockLeft = selectedVariant
    ? selectedVariant.stock
    : product.total_stock || product.stock;

  const bumpProduct = page.bump_product;
  const bumpVariants = (bumpProduct?.variants || []).filter((v) => v.stock > 0);

  const mainTotal = unitPrice * quantity;
  const bumpTotal = bump && page.bump_enabled ? page.bump_price : 0;
  const subtotal = mainTotal + bumpTotal;
  const discount = coupon ? Math.min(coupon.discount, subtotal) : 0;
  const payable = subtotal - discount;

  // Re-check an applied coupon whenever the amount changes (qty/bump edits) —
  // a percent coupon's discount and min-order eligibility both depend on it.
  useEffect(() => {
    if (!coupon) return;
    let stale = false;
    validateCoupon(coupon.code, subtotal, form.phone.trim()).then((res) => {
      if (stale) return;
      if (res.ok && res.coupon) {
        setCoupon(res.coupon);
      } else {
        setCoupon(null);
        setCouponError(res.error || null);
      }
    });
    return () => {
      stale = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [subtotal]);

  async function onApplyCoupon() {
    const code = couponCode.trim();
    if (!code || couponChecking) return;
    setCouponError(null);
    setCouponChecking(true);
    const res = await validateCoupon(code, subtotal, form.phone.trim());
    setCouponChecking(false);
    if (res.ok && res.coupon) {
      setCoupon(res.coupon);
    } else {
      setCoupon(null);
      setCouponError(res.error || null);
    }
  }

  function update(field: keyof typeof form) {
    return (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) =>
      setForm((f) => ({ ...f, [field]: e.target.value }));
  }

  function onSlipChange(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0] ?? null;
    setError(null);
    if (file && !file.type.startsWith("image/")) {
      setError("กรุณาแนบรูปสลิปการโอนเงิน");
      setSlip(null);
      setSlipPreview(null);
      return;
    }
    setSlip(file);
    setSlipPreview(file ? URL.createObjectURL(file) : null);
  }

  function scrollToOrder() {
    orderRef.current?.scrollIntoView({ behavior: "smooth", block: "start" });
  }

  function goToStep2() {
    setError(null);
    if (hasVariants && !variantId) {
      setError("กรุณาเลือกไซซ์/สีก่อน");
      return;
    }
    if (!form.name.trim() || !form.phone.trim() || !form.address.trim()) {
      setError("กรุณากรอกชื่อ เบอร์โทร และที่อยู่จัดส่ง");
      return;
    }
    setStep(2);
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    if (!slip) {
      setError("กรุณาแนบสลิปการโอนเงินเพื่อยืนยันคำสั่งซื้อ");
      return;
    }
    if (bump && bumpVariants.length > 0 && !bumpVariantId) {
      setError("กรุณาเลือกไซซ์/สีของสินค้าแถมพิเศษ");
      return;
    }
    setSubmitting(true);
    const res = await salePageOrder(
      page.slug,
      {
        name: form.name.trim(),
        phone: form.phone.trim(),
        email: form.email.trim() || undefined,
        address: form.address.trim(),
        notes: form.notes.trim() || undefined,
        quantity,
        variantId,
        bump: bump && page.bump_enabled,
        bumpVariantId: bump ? bumpVariantId : null,
        couponCode: coupon?.code,
        // Silent ?ref attribution — no extra field in the funnel form.
        affiliateCode: getAffiliateRef() ?? undefined,
      },
      slip
    );
    setSubmitting(false);
    if (res.ok) {
      setOrderId(res.orderId ?? 0);
      window.scrollTo({ top: 0, behavior: "smooth" });
    } else {
      setError(res.error || "สั่งซื้อไม่สำเร็จ กรุณาลองใหม่");
    }
  }

  // ---- Success state ----
  if (orderId !== null) {
    return (
      <main className={styles.page}>
        <div className={`wrap ${styles.confirm}`}>
          <span className="kicker">Order Received</span>
          <h1 className={`display ${styles.confirmTitle}`}>
            ขอบคุณครับ <em>ออเดอร์ของคุณถูกจองแล้ว</em>
          </h1>
          <p className={styles.confirmCopy}>
            คำสั่งซื้อ (N° {orderId}) ได้รับการบันทึกเรียบร้อย เราจะตรวจสอบสลิป
            และติดต่อกลับเพื่อยืนยันการจัดส่งโดยเร็วที่สุด
          </p>
          <Link href="/shop" className="qlink">
            Explore the Collection <span className="arrow">→</span>
          </Link>
        </div>
      </main>
    );
  }

  const enabledSections = (page.sections || []).filter((s) => s.enabled);
  const heroSection = enabledSections.find((s) => s.type === "hero");
  const contentSections = enabledSections.filter((s) => s.type !== "hero");
  const hasTestimonials = contentSections.some(
    (s) => s.type === "testimonials" && (s.data?.items || []).some((x: any) => x?.quote?.trim())
  );

  const heroData = heroSection?.data || {};
  const heroImage = heroData.image_url || productImages[0] || "";
  const marqueeText = `${product.name} · Limited Release · Quietly Made in Thailand · `;

  return (
    <main className={styles.page}>
      {isPreview && (
        <div className={styles.previewBar}>
          Preview Mode — {page.status === "draft" ? "หน้านี้ยังไม่เผยแพร่" : "views ไม่ถูกนับ"}
        </div>
      )}

      {/* ============ HERO — full-bleed editorial ============ */}
      {heroSection && (
        <section className={`${styles.hero} ${heroImage ? "" : styles.heroPlain}`}>
          {heroImage && (
            <div
              className={styles.heroBg}
              style={{ backgroundImage: `url('${imageSrc(heroImage)}')` }}
              aria-hidden
            />
          )}
          {heroImage && <div className={styles.heroScrim} aria-hidden />}
          <div className={styles.heroInner}>
            <Reveal as="span" className={`kicker ${styles.heroKicker}`}>
              {heroData.kicker || "Limited Release"}
            </Reveal>
            <Reveal delay={2}>
              <h1 className={`display ${styles.heroTitle}`}>
                {emphasize(heroData.headline || product.name)}
              </h1>
            </Reveal>
            {heroData.subheadline && (
              <Reveal delay={3}>
                <p className={styles.heroSub}>{heroData.subheadline}</p>
              </Reveal>
            )}
            <Reveal delay={3}>
              <div className={styles.heroPriceRow}>
                {hasDiscountedOffer && (
                  <span className={styles.heroPriceWas}>{money(catalogPrice)}</span>
                )}
                <span className={styles.heroPrice}>{money(unitPrice)}</span>
                {hasDiscountedOffer && (
                  <span className={styles.saveTag}>
                    ประหยัด {money(catalogPrice - unitPrice)}
                  </span>
                )}
              </div>
            </Reveal>
            {countdown.active && countdown.parts && (
              <Reveal delay={4}>
                <div className={styles.heroCountdown}>
                  <span className={styles.countdownLabel}>ข้อเสนอสิ้นสุดใน</span>
                  <span className={styles.countdownDigits}>
                    {countdown.parts.d > 0 && `${countdown.parts.d} วัน `}
                    {pad2(countdown.parts.h)}:{pad2(countdown.parts.m)}:{pad2(countdown.parts.s)}
                  </span>
                </div>
              </Reveal>
            )}
            <Reveal delay={4}>
              <button type="button" className={styles.heroCta} onClick={scrollToOrder}>
                {heroData.cta_text || "สั่งซื้อตอนนี้"} <span className="arrow">→</span>
              </button>
            </Reveal>
          </div>
          <div className={styles.heroMeta}>
            <span className="label">{heroData.kicker || "Limited Release"}</span>
            <span className={styles.heroMetaBrand}>
              <span className={`serif ${styles.heroMetaNum}`}>Bruno</span>
              <span className="label">Collective</span>
            </span>
            <span className="label">Made in Thailand · งานตัดเย็บมือ</span>
          </div>
        </section>
      )}

      {/* Marquee — the site's signature moving strip */}
      <div className={styles.marquee} aria-hidden>
        <div className={styles.marqueeTrack}>
          <span>{marqueeText.repeat(4)}</span>
          <span>{marqueeText.repeat(4)}</span>
        </div>
      </div>

      {contentSections.map((section, i) => (
        <Section
          key={`${section.type}-${i}`}
          section={section}
          page={page}
          index={i + 1}
          unitPrice={unitPrice}
          productImages={productImages}
          onCta={scrollToOrder}
        />
      ))}

      {/* ============ ORDER FORM ============ */}
      <section className={styles.orderSection} ref={orderRef} id="order">
        <div className={styles.orderInner}>
          <div className={styles.orderHead}>
            <Reveal as="span" className="kicker">Reserve Yours</Reveal>
            <Reveal delay={2}>
              <h2 className={`display ${styles.orderTitle}`}>{emphasize(`สั่งซื้อ ${product.name}`)}</h2>
            </Reveal>
            {hasTestimonials && <div className={styles.stars} aria-label="5 stars">★★★★★</div>}
            <div className={styles.orderPrice}>
              {hasDiscountedOffer && (
                <span className={styles.priceWas}>{money(catalogPrice)}</span>
              )}
              <span className={styles.priceNow}>{money(unitPrice)}</span>
            </div>
            {page.show_stock && stockLeft > 0 && stockLeft <= 20 && (
              <div className={styles.stockNote}>เหลือเพียง {stockLeft} ชิ้น</div>
            )}
            {countdown.active && countdown.parts && (
              <div className={styles.countdownRow}>
                <span className={styles.countdownLabel}>ข้อเสนอสิ้นสุดใน</span>
                <span className={styles.countdownDigits}>
                  {countdown.parts.d > 0 && `${countdown.parts.d} วัน `}
                  {pad2(countdown.parts.h)}:{pad2(countdown.parts.m)}:{pad2(countdown.parts.s)}
                </span>
              </div>
            )}
          </div>

          {offerEnded ? (
            <div className={styles.ended}>
              <h3 className={styles.endedTitle}>ข้อเสนอนี้สิ้นสุดแล้ว</h3>
              <p>ขอบคุณที่สนใจ — ชมคอลเลกชันปัจจุบันของเราได้ที่ร้านค้า</p>
              <Link href="/shop" className="qlink">
                Explore the Collection <span className="arrow">→</span>
              </Link>
            </div>
          ) : (
            <form className={styles.form} onSubmit={onSubmit}>
              {/* Step indicator */}
              <div className={styles.steps}>
                <button
                  type="button"
                  className={`${styles.stepTab} ${step === 1 ? styles.stepActive : styles.stepDone}`}
                  onClick={() => setStep(1)}
                >
                  <span className={styles.stepNo}>01</span> ข้อมูลจัดส่ง
                </button>
                <span className={styles.stepDividerLine} />
                <button
                  type="button"
                  className={`${styles.stepTab} ${step === 2 ? styles.stepActive : ""}`}
                  onClick={goToStep2}
                >
                  <span className={styles.stepNo}>02</span> ชำระเงิน
                </button>
              </div>

              {/* ---- STEP 1 ---- */}
              {step === 1 && (
                <div className={styles.stepBody}>
                  {hasVariants && (
                    <div className={styles.field}>
                      <span>Size / Color *</span>
                      <div className={styles.variantGrid}>
                        {variants.map((v) => {
                          const label = [v.size, v.color].filter(Boolean).join(" / ") || "One size";
                          const out = v.stock <= 0;
                          return (
                            <button
                              key={v.id}
                              type="button"
                              disabled={out}
                              className={`${styles.variantBtn} ${variantId === v.id ? styles.variantOn : ""} ${out ? styles.variantOut : ""}`}
                              onClick={() => setVariantId(v.id)}
                            >
                              {label}
                            </button>
                          );
                        })}
                      </div>
                    </div>
                  )}

                  {showSizeChart && (
                    <div className={styles.sizeChart}>
                      {/* eslint-disable-next-line @next/next/no-img-element */}
                      <img src={imageSrc(sizeChartUrl!)} alt="ตารางไซส์เสื้อ" loading="lazy" />
                    </div>
                  )}

                  <div className={styles.field}>
                    <span>จำนวน</span>
                    <div className={styles.qtyRow}>
                      <button type="button" className={styles.qtyBtn}
                        onClick={() => setQuantity((q) => Math.max(1, q - 1))}>−</button>
                      <span className={styles.qtyNum}>{quantity}</span>
                      <button type="button" className={styles.qtyBtn}
                        onClick={() => setQuantity((q) => Math.min(Math.max(stockLeft, 1), q + 1))}>+</button>
                    </div>
                  </div>

                  <label className={styles.field}>
                    <span>ชื่อ-นามสกุล *</span>
                    <input value={form.name} onChange={update("name")} required />
                  </label>
                  <label className={styles.field}>
                    <span>เบอร์โทร *</span>
                    <input value={form.phone} onChange={update("phone")} required inputMode="tel" />
                  </label>
                  <label className={styles.field}>
                    <span>อีเมล</span>
                    <input value={form.email} onChange={update("email")} type="email" />
                  </label>
                  <label className={styles.field}>
                    <span>ที่อยู่จัดส่ง *</span>
                    <textarea value={form.address} onChange={update("address")} rows={3} required />
                  </label>

                  {error && <p className={styles.error}>{error}</p>}

                  <button type="button" className={styles.next} onClick={goToStep2}>
                    ถัดไป — ชำระเงิน <span className="arrow">→</span>
                  </button>
                </div>
              )}

              {/* ---- STEP 2 ---- */}
              {step === 2 && (
                <div className={styles.stepBody}>
                  {/* Order bump */}
                  {page.bump_enabled && bumpProduct && (
                    <div
                      className={`${styles.bump} ${bump ? styles.bumpOn : ""}`}
                      onClick={() => setBump(!bump)}
                    >
                      <div className={styles.bumpTag}>ข้อเสนอพิเศษ — ครั้งเดียวเท่านั้น</div>
                      <div className={styles.bumpCheckRow}>
                        <span className={`${styles.bumpBox} ${bump ? styles.bumpBoxOn : ""}`}>
                          {bump ? "✓" : ""}
                        </span>
                        {bumpProduct.image_url && (
                          // eslint-disable-next-line @next/next/no-img-element
                          <img
                            className={styles.bumpThumb}
                            src={imageSrc(bumpProduct.image_url)}
                            alt={bumpProduct.name}
                          />
                        )}
                        <div className={styles.bumpText}>
                          <div className={styles.bumpHead}>
                            ✦ {page.bump_headline || `เพิ่ม ${bumpProduct.name} ในราคาพิเศษ`}
                          </div>
                          {page.bump_description && (
                            <div className={styles.bumpDesc}>{page.bump_description}</div>
                          )}
                          <div className={styles.bumpPrice}>
                            {bumpProduct.price > page.bump_price && (
                              <span className={styles.priceWas}>{money(bumpProduct.price)}</span>
                            )}
                            <span>{money(page.bump_price)}</span>
                          </div>
                        </div>
                      </div>
                      {bump && bumpVariants.length > 0 && (
                        <div className={styles.bumpVariants} onClick={(e) => e.stopPropagation()}>
                          {bumpVariants.map((v) => {
                            const label = [v.size, v.color].filter(Boolean).join(" / ") || "One size";
                            return (
                              <button
                                key={v.id}
                                type="button"
                                className={`${styles.variantBtn} ${bumpVariantId === v.id ? styles.variantOn : ""}`}
                                onClick={() => setBumpVariantId(v.id)}
                              >
                                {label}
                              </button>
                            );
                          })}
                        </div>
                      )}
                    </div>
                  )}

                  {/* Coupon */}
                  {page.allow_coupon && (
                    <div className={styles.coupon}>
                      {coupon ? (
                        <div className={styles.couponApplied}>
                          <span>{coupon.code} · ส่วนลด −{money(discount)}</span>
                          <button type="button" onClick={() => { setCoupon(null); setCouponCode(""); }}>
                            Remove
                          </button>
                        </div>
                      ) : (
                        <div className={styles.couponRow}>
                          <input
                            value={couponCode}
                            onChange={(e) => { setCouponCode(e.target.value.toUpperCase()); setCouponError(null); }}
                            onKeyDown={(e) => { if (e.key === "Enter") { e.preventDefault(); onApplyCoupon(); } }}
                            placeholder="Coupon Code · โค้ดส่วนลด"
                          />
                          <button type="button" onClick={onApplyCoupon}
                            disabled={couponChecking || !couponCode.trim()}>
                            {couponChecking ? "…" : "Apply"}
                          </button>
                        </div>
                      )}
                      {couponError && <p className={styles.error}>{couponError}</p>}
                    </div>
                  )}

                  {/* Summary */}
                  <div className={styles.summaryBox}>
                    <div className={styles.sumItemRow}>
                      {productImages[0] && (
                        // eslint-disable-next-line @next/next/no-img-element
                        <img className={styles.sumThumb} src={imageSrc(productImages[0])} alt={product.name} />
                      )}
                      <div className={styles.sumItemBody}>
                        <div className={styles.sumItemName}>{product.name}</div>
                        <div className={styles.sumItemMeta}>
                          {selectedVariant &&
                            `${[selectedVariant.size, selectedVariant.color].filter(Boolean).join(" / ")} · `}
                          จำนวน {quantity}
                        </div>
                      </div>
                      <span className={styles.sumItemPrice}>{money(mainTotal)}</span>
                    </div>
                    {bump && page.bump_enabled && bumpProduct && (
                      <div className={styles.sumRow}>
                        <span>✦ {bumpProduct.name}</span>
                        <span>{money(page.bump_price)}</span>
                      </div>
                    )}
                    {coupon && (
                      <div className={`${styles.sumRow} ${styles.sumDiscount}`}>
                        <span>ส่วนลด ({coupon.code})</span>
                        <span>−{money(discount)}</span>
                      </div>
                    )}
                    <div className={`${styles.sumRow} ${styles.sumTotal}`}>
                      <span>ยอดโอนทั้งสิ้น</span>
                      <span>{money(payable)}</span>
                    </div>
                  </div>

                  {/* Payment */}
                  <div className={styles.payment}>
                    <h3 className={styles.payTitle}>โอนเงินเพื่อยืนยันคำสั่งซื้อ</h3>
                    <dl className={styles.bank}>
                      <div><dt>ธนาคาร</dt><dd>ธนาคารกสิกรไทย (KBank)</dd></div>
                      <div><dt>เลขบัญชี</dt><dd>231-1421-053</dd></div>
                      <div><dt>ชื่อบัญชี</dt><dd>บจก. บรูโน่ คอลเลคทีฟ</dd></div>
                    </dl>
                  </div>

                  <div className={styles.field}>
                    <span>แนบสลิปการโอน *</span>
                    <input ref={fileInput} type="file" accept="image/*" onChange={onSlipChange}
                      className={styles.fileInput} />
                    <button type="button" className={styles.slipBtn} onClick={() => fileInput.current?.click()}>
                      {slip ? "เปลี่ยนสลิป" : "อัปโหลดสลิป"}
                    </button>
                    {slipPreview && (
                      <div className={styles.slipPreview}>
                        {/* eslint-disable-next-line @next/next/no-img-element */}
                        <img src={slipPreview} alt="Payment slip preview" />
                        <span className={styles.slipName}>{slip?.name}</span>
                      </div>
                    )}
                  </div>

                  <label className={styles.field}>
                    <span>หมายเหตุ</span>
                    <textarea value={form.notes} onChange={update("notes")} rows={2} />
                  </label>

                  {error && <p className={styles.error}>{error}</p>}

                  <button type="submit" className={styles.place} disabled={submitting}>
                    {submitting ? "กำลังส่งคำสั่งซื้อ…" : `ยืนยันสั่งซื้อ — ${money(payable)}`}
                    <span className="arrow">→</span>
                  </button>
                  <p className={styles.fine}>
                    ออเดอร์ของคุณจะถูกจองทันทีที่เราได้รับสลิป
                  </p>
                </div>
              )}

              {/* Trust strip — quiet reassurance under the form */}
              <div className={styles.trust}>
                <span>✦ ชำระตรงเข้าบัญชีบริษัท</span>
                <span>✦ ทีมงานยืนยันออเดอร์ทุกวัน</span>
                <span>✦ จัดส่งทั่วประเทศ</span>
              </div>
            </form>
          )}
        </div>
      </section>

      {/* ============ STICKY BUY BAR ============ */}
      {!offerEnded && (
        <div
          className={`${styles.stickyBar} ${pastHero && !orderVisible ? styles.stickyBarOn : ""}`}
          aria-hidden={!(pastHero && !orderVisible)}
        >
          {productImages[0] && (
            // eslint-disable-next-line @next/next/no-img-element
            <img className={styles.stickyThumb} src={imageSrc(productImages[0])} alt="" />
          )}
          <div className={styles.stickyInfo}>
            <span className={styles.stickyName}>{product.name}</span>
            <span className={styles.stickyPrice}>
              {hasDiscountedOffer && <s>{money(catalogPrice)}</s>} {money(unitPrice)}
              {countdown.active && countdown.parts && (
                <span className={styles.stickyTimer}>
                  · เหลือ {countdown.parts.d > 0 ? `${countdown.parts.d} วัน ` : ""}
                  {pad2(countdown.parts.h)}:{pad2(countdown.parts.m)}:{pad2(countdown.parts.s)}
                </span>
              )}
            </span>
          </div>
          <button type="button" className={styles.stickyCta} onClick={scrollToOrder}>
            สั่งซื้อ <span className="arrow">→</span>
          </button>
        </div>
      )}
    </main>
  );
}

// ---- Section renderer (numbered editorial blocks) ----

interface SectionProps {
  section: SalePageSection;
  page: SalePage;
  index: number;
  unitPrice: number;
  productImages: string[];
  onCta: () => void;
}

// Shared numbered section header — mirrors the landing page's `.sec-head`.
function SecHead({ index, kicker, title }: { index: number; kicker: string; title: string }) {
  return (
    <div className="sec-head">
      <Reveal as="span" className="num">{pad2(index)}</Reveal>
      <div className="right">
        <Reveal>
          <span className="kicker">{kicker}</span>
          <h2>{emphasize(title)}</h2>
        </Reveal>
      </div>
    </div>
  );
}

function Section({ section, page, index, unitPrice, productImages, onCta }: SectionProps) {
  const d = section.data || {};
  const kicker = SECTION_KICKERS[section.type] || "";

  switch (section.type) {
    case "pain": {
      const items: string[] = (d.items || []).filter((x: string) => x && x.trim());
      if (!items.length) return null;
      return (
        <section className={styles.block}>
          <div className="wrap">
            <SecHead index={index} kicker={kicker} title={d.title || "คุณเคยเจอแบบนี้ไหม?"} />
            <div className={styles.painGrid}>
              {items.map((item, i) => (
                <Reveal key={i} delay={(Math.min(i, 2) + 2) as 2 | 3 | 4} className={styles.painCard}>
                  <span className={styles.painNum}>{pad2(i + 1)}</span>
                  <p>{item}</p>
                </Reveal>
              ))}
            </div>
          </div>
        </section>
      );
    }

    case "story": {
      const paragraphs = String(d.body || "").split(/\n\s*\n/).filter((p: string) => p.trim());
      const img = d.image_url || productImages[1] || productImages[0];
      if (!paragraphs.length && !img) return null;
      return (
        <section className={`${styles.block} ${styles.blockIvory}`}>
          <div className="wrap">
            <SecHead index={index} kicker={kicker} title={d.title || "เรื่องราวของเรา"} />
            <div className={styles.storyGrid}>
              {img && (
                <Reveal as="figure" className={styles.storyFigure}>
                  {/* eslint-disable-next-line @next/next/no-img-element */}
                  <img className={styles.storyImage} src={imageSrc(img)} alt={d.title || "Story"} />
                  <figcaption className="label label--quiet">Bruno Collective — Atelier</figcaption>
                </Reveal>
              )}
              <div className={styles.storyText}>
                {paragraphs.map((p: string, i: number) =>
                  i === 0 ? (
                    <Reveal as="p" key={i} className={styles.storyLead}>{p}</Reveal>
                  ) : (
                    <Reveal as="p" key={i} delay={2}>{p}</Reveal>
                  )
                )}
              </div>
            </div>
          </div>
        </section>
      );
    }

    case "showcase": {
      let images: string[] = (d.images || []).filter((x: string) => x && x.trim());
      if (!images.length) images = productImages;
      if (!images.length) return null;
      return (
        <section className={styles.block}>
          <div className="wrap">
            <SecHead index={index} kicker={kicker} title={d.title || "รายละเอียดที่มองเห็นได้"} />
            <div className={styles.showcaseGrid}>
              {images.map((img, i) => (
                <Reveal
                  key={i}
                  as="figure"
                  delay={((i % 3) + 2) as 2 | 3 | 4}
                  className={`${styles.showcaseItem} ${i % 3 === 0 ? styles.showcaseTall : ""}`}
                >
                  {/* eslint-disable-next-line @next/next/no-img-element */}
                  <img className={styles.showcaseImage} src={imageSrc(img)} alt={`${page.product.name} ${i + 1}`} />
                </Reveal>
              ))}
            </div>
            {d.caption && <p className={styles.showcaseCaption}>{d.caption}</p>}
          </div>
        </section>
      );
    }

    case "offer": {
      const items: { name: string; value: number }[] = (d.items || []).filter(
        (x: any) => x && x.name && x.name.trim()
      );
      if (!items.length) return null;
      const totalValue = items.reduce((sum, x) => sum + (Number(x.value) || 0), 0);
      return (
        <section className={`${styles.block} ${styles.blockIvory}`}>
          <div className="wrap">
            <SecHead index={index} kicker={kicker} title={d.title || "สิ่งที่คุณจะได้รับ"} />
            <div className={styles.offerWrap}>
              <Reveal className={styles.offerBox}>
                <div className={styles.offerSeal} aria-hidden>✦</div>
                {items.map((item, i) => (
                  <div key={i} className={styles.offerRow}>
                    <span className={styles.offerCheck}>✓</span>
                    <span className={styles.offerName}>{item.name}</span>
                    {Number(item.value) > 0 && (
                      <span className={styles.offerValue}>มูลค่า {money(Number(item.value))}</span>
                    )}
                  </div>
                ))}
                <div className={styles.offerTotalRow}>
                  {totalValue > unitPrice && (
                    <span className={styles.offerTotalWas}>มูลค่ารวม {money(totalValue)}</span>
                  )}
                  <span className={styles.offerTotalNow}>
                    วันนี้เพียง <b>{money(unitPrice)}</b>
                  </span>
                </div>
                {d.note && <p className={styles.offerNote}>{d.note}</p>}
                <button type="button" className={styles.heroCta} onClick={onCta}>
                  รับข้อเสนอนี้ <span className="arrow">→</span>
                </button>
              </Reveal>
            </div>
          </div>
        </section>
      );
    }

    case "testimonials": {
      const items: { quote: string; name: string }[] = (d.items || []).filter(
        (x: any) => x && x.quote && x.quote.trim()
      );
      if (!items.length) return null;
      return (
        <section className={styles.block}>
          <div className="wrap">
            <SecHead index={index} kicker={kicker} title={d.title || "เสียงจากลูกค้า"} />
            <div className={styles.quoteGrid}>
              {items.map((item, i) => (
                <Reveal
                  key={i}
                  as="figure"
                  delay={((i % 3) + 2) as 2 | 3 | 4}
                  className={`${styles.quote} ${i % 2 === 1 ? styles.quoteOffset : ""}`}
                >
                  <span className={styles.quoteMark} aria-hidden>“</span>
                  <div className={styles.stars} aria-label="5 stars">★★★★★</div>
                  <blockquote>{item.quote}</blockquote>
                  {item.name && <figcaption>— {item.name}</figcaption>}
                </Reveal>
              ))}
            </div>
            <div className={styles.blockCtaRow}>
              <button type="button" className="qlink" onClick={onCta}>
                ร่วมเป็นหนึ่งในนั้น — สั่งซื้อเลย <span className="arrow">→</span>
              </button>
            </div>
          </div>
        </section>
      );
    }

    case "guarantee":
      if (!d.body) return null;
      return (
        <section className={`${styles.block} ${styles.blockIvory}`}>
          <div className={`wrap ${styles.guaranteeWrap}`}>
            <Reveal>
              <div className={styles.guaranteeSeal}>✦</div>
              <span className="kicker">{kicker}</span>
              <h2 className={`display ${styles.guaranteeTitle}`}>{emphasize(d.title || "การรับประกันจากเรา")}</h2>
              <p className={styles.guaranteeBody}>{d.body}</p>
            </Reveal>
          </div>
        </section>
      );

    case "faq": {
      const items: { q: string; a: string }[] = (d.items || []).filter(
        (x: any) => x && x.q && x.q.trim()
      );
      if (!items.length) return null;
      return (
        <section className={styles.block}>
          <div className="wrap">
            <SecHead index={index} kicker={kicker} title={d.title || "คำถามที่พบบ่อย"} />
            <div className={styles.faqList}>
              {items.map((item, i) => (
                <Reveal key={i}>
                  <details className={styles.faqItem}>
                    <summary>{item.q}</summary>
                    <p>{item.a}</p>
                  </details>
                </Reveal>
              ))}
            </div>
          </div>
        </section>
      );
    }

    default:
      return null;
  }
}
