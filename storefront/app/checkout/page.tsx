"use client";

import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import { useCart } from "@/lib/cart";
import { useMember } from "@/lib/member";
import { checkout, getSuggestions, memberCheck, validateAffiliate, validateCoupon } from "@/lib/api";
import { getAffiliateRef } from "@/lib/affiliate";
import { fbqTrack } from "@/lib/fbq";
import { money, imageSrc } from "@/lib/format";
import type { CouponPreview, Product } from "@/lib/types";
import styles from "./checkout.module.css";

export default function CheckoutPage() {
  const { lines, total, clear, add, setOpen } = useCart();
  const { member } = useMember();
  const [form, setForm] = useState({
    name: "",
    phone: "",
    email: "",
    address: "",
    notes: "",
  });
  const [slip, setSlip] = useState<File | null>(null);
  const [slipPreview, setSlipPreview] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [orderId, setOrderId] = useState<number | null>(null);
  const fileInput = useRef<HTMLInputElement>(null);

  // Coupon — previewed against the cart subtotal; the backend revalidates
  // (and enforces quotas atomically) when the order is placed.
  const [couponCode, setCouponCode] = useState("");
  const [coupon, setCoupon] = useState<CouponPreview | null>(null);
  const [couponError, setCouponError] = useState<string | null>(null);
  const [couponChecking, setCouponChecking] = useState(false);

  // Affiliate referral — prefilled from a remembered ?ref link (30-day,
  // last-click); typing a code overrides it. Validation is instant feedback
  // only — the backend re-resolves the code and never fails the sale on it.
  const [refCode, setRefCode] = useState("");
  const [refApplied, setRefApplied] = useState<string | null>(null);
  const [refError, setRefError] = useState<string | null>(null);
  useEffect(() => {
    const saved = getAffiliateRef();
    if (saved) {
      setRefCode(saved);
      setRefApplied(saved);
    }
  }, []);
  async function onApplyRef() {
    const code = refCode.trim();
    if (!code) return;
    setRefError(null);
    const res = await validateAffiliate(code);
    if (res.ok && res.code) {
      setRefApplied(res.code);
    } else {
      setRefApplied(null);
      setRefError(res.error || null);
    }
  }

  // Member discount — logged-in members get it from their profile; guests get
  // it when their phone matches a member/returning customer. Display only:
  // the backend recomputes and applies the real discount at order time.
  const [memberPct, setMemberPct] = useState(0);
  const prefilled = useRef(false);

  useEffect(() => {
    if (!member) return;
    setMemberPct(member.discount_percent);
    if (!prefilled.current) {
      prefilled.current = true;
      setForm((f) => ({
        ...f,
        name: f.name || member.name,
        phone: f.phone || member.phone,
        email: f.email || member.email,
        address: f.address || member.address,
      }));
    }
  }, [member]);

  // Guest phone check (debounced) — returning customers qualify by phone.
  useEffect(() => {
    if (member) return;
    const phone = form.phone.trim();
    if (phone.length < 9) {
      setMemberPct(0);
      return;
    }
    const timer = setTimeout(async () => {
      const res = await memberCheck(phone);
      setMemberPct(res.is_member ? res.discount_percent : 0);
    }, 600);
    return () => clearTimeout(timer);
  }, [member, form.phone]);

  // Cross-sell: "bought together" suggestions for the current bag. Keyed on
  // the product-id set so adding a suggested item refreshes the list (the
  // backend excludes items already in the bag).
  const [suggestions, setSuggestions] = useState<Product[]>([]);
  const cartIds = lines.map((l) => l.product.id).sort().join(",");
  useEffect(() => {
    if (!cartIds) {
      setSuggestions([]);
      return;
    }
    let cancelled = false;
    getSuggestions(cartIds.split(",").map(Number)).then((res) => {
      if (!cancelled) setSuggestions(res.slice(0, 3));
    });
    return () => {
      cancelled = true;
    };
  }, [cartIds]);

  function addSuggestion(p: Product) {
    add(p, null);
    setOpen(false); // stay on the checkout page — no bag drawer popover
  }

  const memberDiscount = memberPct > 0 ? Math.round(total * memberPct) / 100 : 0;
  const discount = coupon
    ? Math.min(coupon.discount, total - memberDiscount)
    : 0;
  const payable = total - memberDiscount - discount;

  // Pixel: InitiateCheckout once per visit to this page with items in the bag.
  const checkoutTracked = useRef(false);
  useEffect(() => {
    if (checkoutTracked.current || lines.length === 0) return;
    checkoutTracked.current = true;
    fbqTrack("InitiateCheckout", {
      content_ids: lines.map((l) => String(l.product.id)),
      content_type: "product",
      num_items: lines.reduce((n, l) => n + l.quantity, 0),
      value: total,
      currency: "THB",
    });
  }, [lines, total]);

  async function onApplyCoupon() {
    const code = couponCode.trim();
    if (!code || couponChecking) return;
    setCouponError(null);
    setCouponChecking(true);
    const res = await validateCoupon(code, total, form.phone.trim());
    setCouponChecking(false);
    if (res.ok && res.coupon) {
      setCoupon(res.coupon);
    } else {
      setCoupon(null);
      setCouponError(res.error || null);
    }
  }

  function onRemoveCoupon() {
    setCoupon(null);
    setCouponCode("");
    setCouponError(null);
  }

  function update(field: keyof typeof form) {
    return (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) =>
      setForm((f) => ({ ...f, [field]: e.target.value }));
  }

  function onSlipChange(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0] ?? null;
    setError(null);
    if (file && !file.type.startsWith("image/")) {
      setError("Please upload an image of your payment slip.");
      setSlip(null);
      setSlipPreview(null);
      return;
    }
    setSlip(file);
    setSlipPreview(file ? URL.createObjectURL(file) : null);
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    if (!slip) {
      setError("Please attach your payment slip to place the order.");
      return;
    }
    setSubmitting(true);
    const res = await checkout(
      {
        name: form.name.trim(),
        phone: form.phone.trim(),
        email: form.email.trim() || undefined,
        address: form.address.trim(),
        notes: form.notes.trim() || undefined,
        coupon_code: coupon ? coupon.code : undefined,
        affiliate_code: refApplied || getAffiliateRef() || undefined,
        items: lines.map((l) => ({
          product_id: l.product.id,
          variant_id: l.variant ? l.variant.id : null,
          quantity: l.quantity,
        })),
      },
      slip
    );
    setSubmitting(false);
    if (res.ok) {
      fbqTrack("Purchase", {
        content_ids: lines.map((l) => String(l.product.id)),
        content_type: "product",
        num_items: lines.reduce((n, l) => n + l.quantity, 0),
        value: payable,
        currency: "THB",
      });
      setOrderId(res.orderId ?? 0);
      clear();
    } else {
      setError(res.error || "Something went wrong. Please try again.");
    }
  }

  // ---- Confirmation ----
  if (orderId !== null) {
    return (
      <main className={styles.page}>
        <div className={styles.confirm}>
          <span className="kicker">Order Received</span>
          <h1 className={`display ${styles.confirmTitle}`}>
            Thank you. <em>It is reserved.</em>
          </h1>
          <p className={styles.confirmCopy}>
            Your order{orderId ? ` (N° ${orderId})` : ""} has been placed and your
            pieces are reserved from atelier stock. We will be in touch shortly to
            arrange payment and delivery.
          </p>
          <Link href="/shop" className="qlink">
            Continue Shopping <span className="arrow">→</span>
          </Link>
        </div>
      </main>
    );
  }

  // ---- Empty bag guard ----
  if (lines.length === 0) {
    return (
      <main className={styles.page}>
        <div className={styles.confirm}>
          <span className="kicker">Checkout</span>
          <h1 className={`display ${styles.confirmTitle}`}>Your bag is empty.</h1>
          <Link href="/shop" className="qlink">
            Explore the Collection <span className="arrow">→</span>
          </Link>
        </div>
      </main>
    );
  }

  return (
    <main className={styles.page}>
      <header className={styles.head}>
        <span className="kicker">Checkout</span>
        <h1 className={`display ${styles.title}`}>Shipping &amp; Details</h1>
      </header>

      <div className={styles.layout}>
        <form className={styles.form} onSubmit={onSubmit}>
          <label className={styles.field}>
            <span>Full Name *</span>
            <input value={form.name} onChange={update("name")} required />
          </label>
          <label className={styles.field}>
            <span>Phone *</span>
            <input value={form.phone} onChange={update("phone")} required inputMode="tel" />
          </label>
          <label className={styles.field}>
            <span>Email</span>
            <input value={form.email} onChange={update("email")} type="email" />
          </label>
          <label className={styles.field}>
            <span>Shipping Address *</span>
            <textarea value={form.address} onChange={update("address")} required rows={3} />
          </label>
          <label className={styles.field}>
            <span>Notes</span>
            <textarea value={form.notes} onChange={update("notes")} rows={2} />
          </label>

          <div className={styles.payment}>
            <h2 className={styles.payTitle}>Payment · ชำระเงิน</h2>
            <p className={styles.payCopy}>
              Please transfer the total below, then attach your payment slip to
              confirm the order. โอนยอดด้านล่าง แล้วแนบสลิปเพื่อยืนยันคำสั่งซื้อ
            </p>
            <dl className={styles.bank}>
              <div>
                <dt>Bank · ธนาคาร</dt>
                <dd>ธนาคารกสิกรไทย (KBank)</dd>
              </div>
              <div>
                <dt>Account · เลขบัญชี</dt>
                <dd>231-1421-053</dd>
              </div>
              <div>
                <dt>Name · ชื่อบัญชี</dt>
                <dd>บจก. บรูโน่ คอลเลคทีฟ</dd>
              </div>
            </dl>
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img
              className={styles.bankImage}
              src="/payment/promptpay-qr.jpg"
              alt="Thai QR Payment (พร้อมเพย์) — สแกนเพื่อชำระเงิน บจก. บรูโน่ คอลเลคทีฟ"
            />
          </div>

          <div className={styles.field}>
            <span>Payment Slip *</span>
            <input
              ref={fileInput}
              type="file"
              accept="image/*"
              onChange={onSlipChange}
              className={styles.fileInput}
            />
            <button
              type="button"
              className={styles.slipBtn}
              onClick={() => fileInput.current?.click()}
            >
              {slip ? "Change Slip" : "Upload Slip"}
            </button>
            {slipPreview && (
              <div className={styles.slipPreview}>
                {/* eslint-disable-next-line @next/next/no-img-element */}
                <img src={slipPreview} alt="Payment slip preview" />
                <span className={styles.slipName}>{slip?.name}</span>
              </div>
            )}
          </div>

          {error && <p className={styles.error}>{error}</p>}

          <button type="submit" className={styles.place} disabled={submitting}>
            {submitting ? "Placing Order…" : "Place Order"} <span className="arrow">→</span>
          </button>
          <p className={styles.fine}>
            Your order is reserved once we receive your slip. We will confirm
            delivery with you directly.
          </p>
        </form>

        <aside className={styles.summary}>
          <h2 className={styles.sumTitle}>Your Bag</h2>
          <div className={styles.items}>
            {lines.map((l) => {
              const variantLabel = l.variant
                ? [l.variant.size, l.variant.color].filter(Boolean).join(" / ")
                : "";
              return (
                <div key={`${l.product.id}:${l.variant ? l.variant.id : 0}`} className={styles.item}>
                  <div
                    className={styles.thumb}
                    style={{
                      backgroundImage: l.product.image_url
                        ? `url('${imageSrc(l.product.image_url)}')`
                        : undefined,
                    }}
                  />
                  <div className={styles.itemBody}>
                    <div className={styles.itemName}>{l.product.name}</div>
                    <div className={styles.itemMeta}>
                      {variantLabel ? `${variantLabel} · ` : ""}Qty {l.quantity}
                    </div>
                  </div>
                  <div className={styles.itemPrice}>{money(l.product.price * l.quantity)}</div>
                </div>
              );
            })}
          </div>
          {suggestions.length > 0 && (
            <div className={styles.crosssell}>
              <div className={styles.crosssellTitle}>ซื้อคู่กันบ่อย · Often bought together</div>
              {suggestions.map((p) => (
                <div key={p.id} className={styles.crosssellItem}>
                  <div
                    className={styles.thumb}
                    style={{
                      backgroundImage: p.image_url
                        ? `url('${imageSrc(p.image_url)}')`
                        : undefined,
                    }}
                  />
                  <div className={styles.itemBody}>
                    <div className={styles.itemName}>{p.name}</div>
                    <div className={styles.itemMeta}>{money(p.price)}</div>
                  </div>
                  {p.variants && p.variants.length > 0 ? (
                    <Link href={`/product/${p.id}`} className={styles.crosssellBtn}>
                      เลือกไซส์
                    </Link>
                  ) : (
                    <button
                      type="button"
                      className={styles.crosssellBtn}
                      onClick={() => addSuggestion(p)}
                    >
                      + เพิ่ม
                    </button>
                  )}
                </div>
              ))}
            </div>
          )}

          <div className={styles.coupon}>
            {coupon ? (
              <div className={styles.couponApplied}>
                <span>
                  {coupon.code} · ส่วนลด −{money(discount)}
                </span>
                <button type="button" onClick={onRemoveCoupon}>
                  Remove
                </button>
              </div>
            ) : (
              <div className={styles.couponRow}>
                <input
                  value={couponCode}
                  onChange={(e) => {
                    setCouponCode(e.target.value.toUpperCase());
                    setCouponError(null);
                  }}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") {
                      e.preventDefault();
                      onApplyCoupon();
                    }
                  }}
                  placeholder="Coupon Code · โค้ดส่วนลด"
                  aria-label="Coupon code"
                />
                <button
                  type="button"
                  onClick={onApplyCoupon}
                  disabled={couponChecking || !couponCode.trim()}
                >
                  {couponChecking ? "…" : "Apply"}
                </button>
              </div>
            )}
            {couponError && <p className={styles.couponError}>{couponError}</p>}
          </div>

          {/* Referral code — reuses the coupon field styling */}
          <div className={styles.coupon}>
            {refApplied ? (
              <div className={styles.couponApplied}>
                <span>รหัสผู้แนะนำ · {refApplied}</span>
                <button
                  type="button"
                  onClick={() => {
                    setRefApplied(null);
                    setRefCode("");
                  }}
                >
                  Remove
                </button>
              </div>
            ) : (
              <div className={styles.couponRow}>
                <input
                  value={refCode}
                  onChange={(e) => {
                    setRefCode(e.target.value.toUpperCase());
                    setRefError(null);
                  }}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") {
                      e.preventDefault();
                      onApplyRef();
                    }
                  }}
                  placeholder="Referral Code · รหัสผู้แนะนำ (ถ้ามี)"
                  aria-label="Referral code"
                />
                <button type="button" onClick={onApplyRef} disabled={!refCode.trim()}>
                  Apply
                </button>
              </div>
            )}
            {refError && <p className={styles.couponError}>{refError}</p>}
          </div>

          <p className={styles.memberNote}>
            {memberDiscount > 0 ? (
              <>✦ ส่วนลดสมาชิก {memberPct}% ถูกนำมาคำนวณแล้ว</>
            ) : (
              <>
                สมาชิกลด 5% ทุกออเดอร์ —{" "}
                <Link href="/member">เข้าสู่ระบบ / สมัครสมาชิก</Link>
              </>
            )}
          </p>

          {(coupon || memberDiscount > 0) && (
            <div className={styles.sumRow}>
              <span>Subtotal</span>
              <span>{money(total)}</span>
            </div>
          )}
          {memberDiscount > 0 && (
            <div className={`${styles.sumRow} ${styles.sumDiscount}`}>
              <span>Member −{memberPct}%</span>
              <span>−{money(memberDiscount)}</span>
            </div>
          )}
          {coupon && (
            <div className={`${styles.sumRow} ${styles.sumDiscount}`}>
              <span>Discount ({coupon.code})</span>
              <span>−{money(discount)}</span>
            </div>
          )}
          <div className={`${styles.sumRow} ${styles.sumTotal}`}>
            <span>Total</span>
            <span>{money(payable)}</span>
          </div>
        </aside>
      </div>
    </main>
  );
}
