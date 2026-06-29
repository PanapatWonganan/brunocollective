"use client";

import { useRef, useState } from "react";
import Link from "next/link";
import { useCart } from "@/lib/cart";
import { checkout } from "@/lib/api";
import { money, imageSrc } from "@/lib/format";
import styles from "./checkout.module.css";

export default function CheckoutPage() {
  const { lines, total, clear } = useCart();
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
        items: lines.map((l) => ({ product_id: l.product.id, quantity: l.quantity })),
      },
      slip
    );
    setSubmitting(false);
    if (res.ok) {
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
              src="/payment/bank-kbank.jpg"
              alt="ช่องทางการชำระเงิน — ธนาคารกสิกรไทย เลขบัญชี 231-1421-053 บจก. บรูโน่ คอลเลคทีฟ"
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
            {lines.map((l) => (
              <div key={l.product.id} className={styles.item}>
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
                  <div className={styles.itemMeta}>Qty {l.quantity}</div>
                </div>
                <div className={styles.itemPrice}>{money(l.product.price * l.quantity)}</div>
              </div>
            ))}
          </div>
          <div className={`${styles.sumRow} ${styles.sumTotal}`}>
            <span>Total</span>
            <span>{money(total)}</span>
          </div>
        </aside>
      </div>
    </main>
  );
}
