"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { memberOrders, memberUpdate } from "@/lib/api";
import { useMember } from "@/lib/member";
import { money } from "@/lib/format";
import type { MemberOrder } from "@/lib/types";
import styles from "../member.module.css";

const STATUS_TH: Record<string, string> = {
  pending: "รอตรวจสอบ",
  confirmed: "ยืนยันแล้ว",
  shipped: "จัดส่งแล้ว",
  delivered: "ได้รับสินค้าแล้ว",
  cancelled: "ยกเลิก",
};

function orderDate(iso: string): string {
  return new Date(iso).toLocaleDateString("th-TH", {
    day: "numeric",
    month: "short",
    year: "numeric",
  });
}

export default function MemberAccountPage() {
  const router = useRouter();
  const { member, ready, signOut, setMember } = useMember();
  const [orders, setOrders] = useState<MemberOrder[]>([]);
  const [form, setForm] = useState({ name: "", email: "", address: "" });
  const [saving, setSaving] = useState(false);
  const [saveMsg, setSaveMsg] = useState<string | null>(null);
  const [saveErr, setSaveErr] = useState<string | null>(null);

  // Not signed in — send to the login page (once the token check finishes).
  useEffect(() => {
    if (ready && !member) router.replace("/member");
  }, [ready, member, router]);

  useEffect(() => {
    if (!member) return;
    setForm({ name: member.name, email: member.email, address: member.address });
    memberOrders().then(setOrders);
  }, [member?.id]); // eslint-disable-line react-hooks/exhaustive-deps

  if (!ready || !member) return null;

  function update(field: keyof typeof form) {
    return (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) =>
      setForm((f) => ({ ...f, [field]: e.target.value }));
  }

  async function onSave(e: React.FormEvent) {
    e.preventDefault();
    setSaveMsg(null);
    setSaveErr(null);
    setSaving(true);
    const res = await memberUpdate({
      name: form.name.trim(),
      email: form.email.trim(),
      address: form.address.trim(),
    });
    setSaving(false);
    if (res.ok && res.member) {
      setMember(res.member);
      setSaveMsg("บันทึกข้อมูลแล้ว");
    } else {
      setSaveErr(res.error || "บันทึกไม่สำเร็จ กรุณาลองใหม่");
    }
  }

  function onSignOut() {
    signOut();
    router.replace("/member");
  }

  return (
    <main className={styles.page}>
      <header className={styles.head}>
        <span className="kicker">Membership</span>
        <h1 className={`display ${styles.title}`}>
          Welcome back, <em>{member.name.split(" ")[0]}</em>
        </h1>
        <p className={styles.lede}>
          สถานะสมาชิก: ส่วนลด {member.discount_percent}% ทุกคำสั่งซื้อ
          (ระบบหักให้อัตโนมัติตอน checkout — ใช้คูปองซ้อนได้)
        </p>
      </header>

      <div className={styles.accountGrid}>
        <section className={styles.panel}>
          <h2 className={styles.panelTitle}>ข้อมูลของฉัน</h2>
          <p className={styles.memberMeta}>
            Member
            {member.member_since
              ? ` · since ${orderDate(member.member_since)}`
              : ""}{" "}
            · {member.phone}
          </p>
          <form className={styles.form} onSubmit={onSave}>
            <label className={styles.field}>
              <span>ชื่อ-นามสกุล · Full Name</span>
              <input value={form.name} onChange={update("name")} required />
            </label>
            <label className={styles.field}>
              <span>อีเมล · Email</span>
              <input value={form.email} onChange={update("email")} type="email" />
            </label>
            <label className={styles.field}>
              <span>ที่อยู่จัดส่ง · Shipping Address</span>
              <textarea value={form.address} onChange={update("address")} rows={3} />
            </label>
            {saveErr && <p className={styles.error}>{saveErr}</p>}
            {saveMsg && <p className={styles.success}>{saveMsg}</p>}
            <button type="submit" className={styles.submit} disabled={saving}>
              {saving ? "…" : "บันทึก"} <span className="arrow">→</span>
            </button>
          </form>
          <button type="button" className={styles.signout} onClick={onSignOut}>
            ออกจากระบบ · Sign Out
          </button>
        </section>

        <section>
          <h2 className={styles.panelTitle}>ประวัติคำสั่งซื้อ</h2>
          <p className={styles.memberMeta}>{orders.length} orders</p>
          {orders.length === 0 ? (
            <p className={styles.empty}>
              ยังไม่มีคำสั่งซื้อ —{" "}
              <Link href="/shop" className="qlink">
                เลือกชมสินค้า <span className="arrow">→</span>
              </Link>
            </p>
          ) : (
            <div className={styles.orders}>
              {orders.map((o) => (
                <article key={o.id} className={styles.order}>
                  <div className={styles.orderHead}>
                    <span className={styles.orderNo}>N° {o.id}</span>
                    <span className={styles.orderDate}>{orderDate(o.created_at)}</span>
                    <span className={styles.orderStatus}>
                      {STATUS_TH[o.status] || o.status}
                    </span>
                  </div>
                  <p className={styles.orderItems}>
                    {o.items
                      .map((it) => {
                        const variant = [it.size, it.color].filter(Boolean).join("/");
                        return `${it.product.name}${variant ? ` (${variant})` : ""} ×${it.quantity}`;
                      })
                      .join(" · ")}
                  </p>
                  <div className={styles.orderTotals}>
                    {(o.member_discount > 0 || o.discount_amount > 0) && (
                      <div className={styles.orderRow}>
                        <span>ยอดสินค้า</span>
                        <span>{money(o.subtotal || o.total_amount)}</span>
                      </div>
                    )}
                    {o.member_discount > 0 && (
                      <div className={`${styles.orderRow} ${styles.orderDiscount}`}>
                        <span>ส่วนลดสมาชิก</span>
                        <span>−{money(o.member_discount)}</span>
                      </div>
                    )}
                    {o.discount_amount > 0 && (
                      <div className={`${styles.orderRow} ${styles.orderDiscount}`}>
                        <span>คูปอง{o.coupon_code ? ` (${o.coupon_code})` : ""}</span>
                        <span>−{money(o.discount_amount)}</span>
                      </div>
                    )}
                    <div className={`${styles.orderRow} ${styles.orderGrand}`}>
                      <span>ยอดชำระ</span>
                      <span>{money(o.total_amount)}</span>
                    </div>
                  </div>
                </article>
              ))}
            </div>
          )}
        </section>
      </div>
    </main>
  );
}
