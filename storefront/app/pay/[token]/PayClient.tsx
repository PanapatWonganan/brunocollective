"use client";

import { useRef, useState } from "react";
import Link from "next/link";
import { uploadPaySlip, type PayOrder } from "@/lib/api";
import { fbqTrack } from "@/lib/fbq";
import { money } from "@/lib/format";
import ThankYou from "@/components/ThankYou";
import styles from "./pay.module.css";

// Order status → customer-facing Thai label. "pending" splits on whether a
// slip has already been received.
function statusView(order: PayOrder): { label: string; tone: "wait" | "ok" | "bad" } {
  switch (order.status) {
    case "pending":
      return order.has_slip
        ? { label: "ได้รับสลิปแล้ว · กำลังตรวจสอบ", tone: "wait" }
        : { label: "รอชำระเงิน", tone: "wait" };
    case "confirmed":
      return { label: "ยืนยันการชำระเงินแล้ว", tone: "ok" };
    case "shipped":
      return { label: "จัดส่งแล้ว", tone: "ok" };
    case "delivered":
      return { label: "จัดส่งสำเร็จ", tone: "ok" };
    case "cancelled":
      return { label: "ออเดอร์ถูกยกเลิก", tone: "bad" };
    default:
      return { label: order.status, tone: "wait" };
  }
}

export default function PayClient({
  token,
  initialOrder,
}: {
  token: string;
  initialOrder: PayOrder;
}) {
  const [order, setOrder] = useState(initialOrder);
  const [slip, setSlip] = useState<File | null>(null);
  const [slipPreview, setSlipPreview] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [justUploaded, setJustUploaded] = useState(false);
  const fileInput = useRef<HTMLInputElement>(null);

  const status = statusView(order);
  const showPayment = order.status === "pending";

  function onSlipChange(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0] ?? null;
    setError(null);
    if (file && !file.type.startsWith("image/")) {
      setError("กรุณาแนบสลิปเป็นไฟล์รูปภาพ");
      setSlip(null);
      setSlipPreview(null);
      return;
    }
    setSlip(file);
    setSlipPreview(file ? URL.createObjectURL(file) : null);
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!slip || submitting) return;
    setError(null);
    setSubmitting(true);
    const firstSlip = !order.has_slip; // re-uploads must not double-count Purchase
    const res = await uploadPaySlip(token, slip);
    setSubmitting(false);
    if (res.ok && res.order) {
      if (firstSlip) {
        fbqTrack("Purchase", {
          content_ids: res.order.items.map((it) => String(it.product_id)),
          content_type: "product",
          num_items: res.order.items.reduce((n, it) => n + it.quantity, 0),
          value: res.order.total_amount,
          currency: "THB",
        });
      }
      setOrder(res.order);
      setSlip(null);
      setSlipPreview(null);
      setJustUploaded(true);
    } else {
      setError(res.error || "อัปโหลดสลิปไม่สำเร็จ กรุณาลองใหม่");
    }
  }

  // ---- Thank-you screen after the slip lands ----
  if (justUploaded) {
    return (
      <ThankYou
        orderNo={order.order_no}
        title="ขอบคุณค่ะ"
        titleEm="ได้รับสลิปของคุณแล้ว"
        copy="ร้านจะตรวจสอบยอดโอนและยืนยันคำสั่งซื้อให้เร็วที่สุด หากมีข้อสงสัยทักแชทหาร้านได้เลยค่ะ"
        productIds={order.items.map((it) => it.product_id)}
      >
        <button
          type="button"
          className={styles.backLink}
          onClick={() => setJustUploaded(false)}
        >
          ดูรายละเอียดออเดอร์ / แก้ไขสลิป
        </button>
      </ThankYou>
    );
  }

  return (
    <main className={styles.page}>
      <header className={styles.head}>
        <span className="kicker">Payment · ชำระเงิน</span>
        <h1 className={`display ${styles.title}`}>
          Order <em>N° {order.order_no}</em>
        </h1>
        <p className={styles.greeting}>
          คุณ{order.customer_name} — ขอบคุณที่สั่งซื้อกับ Bruno Collective ค่ะ
        </p>
        <span className={`${styles.status} ${styles[`status_${status.tone}`]}`}>
          {status.label}
        </span>
      </header>

      <div className={styles.layout}>
        <section className={styles.summary}>
          <h2 className={styles.sumTitle}>รายการสินค้า</h2>
          <div className={styles.items}>
            {order.items.map((it, i) => {
              const variant = [it.size, it.color].filter(Boolean).join(" / ");
              return (
                <div key={i} className={styles.item}>
                  <div className={styles.itemBody}>
                    <div className={styles.itemName}>{it.name}</div>
                    <div className={styles.itemMeta}>
                      {variant ? `${variant} · ` : ""}จำนวน {it.quantity}
                    </div>
                  </div>
                  <div className={styles.itemPrice}>{money(it.price * it.quantity)}</div>
                </div>
              );
            })}
          </div>
          {(order.member_discount > 0 || order.discount_amount > 0) && (
            <div className={styles.sumRow}>
              <span>ยอดรวม</span>
              <span>{money(order.subtotal)}</span>
            </div>
          )}
          {order.member_discount > 0 && (
            <div className={`${styles.sumRow} ${styles.sumDiscount}`}>
              <span>ส่วนลดสมาชิก</span>
              <span>−{money(order.member_discount)}</span>
            </div>
          )}
          {order.discount_amount > 0 && (
            <div className={`${styles.sumRow} ${styles.sumDiscount}`}>
              <span>คูปอง {order.coupon_code}</span>
              <span>−{money(order.discount_amount)}</span>
            </div>
          )}
          <div className={`${styles.sumRow} ${styles.sumTotal}`}>
            <span>ยอดชำระ</span>
            <span>{money(order.total_amount)}</span>
          </div>
        </section>

        {showPayment ? (
          <form className={styles.paybox} onSubmit={onSubmit}>
            <h2 className={styles.payTitle}>ชำระเงิน {money(order.total_amount)}</h2>
            <p className={styles.payCopy}>
              โอนยอดด้านบนแล้วแนบสลิปในหน้านี้ได้เลย ระบบจะแจ้งร้านให้ตรวจสอบทันที
            </p>
            <dl className={styles.bank}>
              <div>
                <dt>ธนาคาร</dt>
                <dd>ธนาคารกสิกรไทย (KBank)</dd>
              </div>
              <div>
                <dt>เลขบัญชี</dt>
                <dd>231-1421-053</dd>
              </div>
              <div>
                <dt>ชื่อบัญชี</dt>
                <dd>บจก. บรูโน่ คอลเลคทีฟ</dd>
              </div>
            </dl>
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img
              className={styles.bankImage}
              src="/payment/promptpay-qr.jpg"
              alt="Thai QR Payment (พร้อมเพย์) — สแกนเพื่อชำระเงิน บจก. บรูโน่ คอลเลคทีฟ"
            />

            {order.has_slip && (
              <p className={styles.received}>
                {justUploaded
                  ? "✓ ได้รับสลิปแล้ว ขอบคุณค่ะ ร้านจะตรวจสอบและยืนยันให้เร็วที่สุด"
                  : "เราได้รับสลิปของคุณแล้ว หากต้องการแก้ไข อัปโหลดใหม่ได้ด้านล่าง"}
              </p>
            )}

            <div className={styles.slipField}>
              <span>แนบสลิปการโอนเงิน{order.has_slip ? " (อัปโหลดใหม่)" : " *"}</span>
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
                {slip ? "เปลี่ยนรูปสลิป" : "เลือกรูปสลิป"}
              </button>
              {slipPreview && (
                <div className={styles.slipPreview}>
                  {/* eslint-disable-next-line @next/next/no-img-element */}
                  <img src={slipPreview} alt="ตัวอย่างสลิป" />
                  <span className={styles.slipName}>{slip?.name}</span>
                </div>
              )}
            </div>

            {error && <p className={styles.error}>{error}</p>}

            <button type="submit" className={styles.submit} disabled={!slip || submitting}>
              {submitting ? "กำลังส่งสลิป…" : "ยืนยันการชำระเงิน"} <span className="arrow">→</span>
            </button>
          </form>
        ) : (
          <div className={styles.paybox}>
            <h2 className={styles.payTitle}>{status.label}</h2>
            <p className={styles.payCopy}>
              {order.status === "cancelled"
                ? "ออเดอร์นี้ถูกยกเลิกแล้ว หากมีข้อสงสัยทักแชทหาร้านได้เลยค่ะ"
                : "ขอบคุณที่สั่งซื้อกับเรา หากมีข้อสงสัยทักแชทหาร้านได้เลยค่ะ"}
            </p>
            <Link href="/shop" className="qlink">
              เลือกชมสินค้าต่อ <span className="arrow">→</span>
            </Link>
          </div>
        )}
      </div>
    </main>
  );
}
