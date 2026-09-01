"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import {
  affiliateChangePassword,
  affiliateMe,
  affiliateOrders,
  type AffiliateOrderRow,
  type AffiliateStats,
} from "@/lib/api";
import { useAffiliate } from "@/lib/affiliate";
import { money } from "@/lib/format";
import styles from "../affiliate.module.css";

const STATUS_LABEL: Record<string, { label: string; cls: string }> = {
  pending: { label: "รอส่งมอบ", cls: "badgePending" },
  confirmed: { label: "ยืนยันแล้ว", cls: "badgeConfirmed" },
  paid: { label: "จ่ายแล้ว", cls: "badgePaid" },
  cancelled: { label: "ยกเลิก", cls: "badgeCancelled" },
};

export default function AffiliateDashboard() {
  const router = useRouter();
  const { affiliate, ready, signOut } = useAffiliate();
  const [stats, setStats] = useState<AffiliateStats | null>(null);
  const [orders, setOrders] = useState<AffiliateOrderRow[]>([]);
  const [copied, setCopied] = useState<"code" | "link" | null>(null);

  // Password change
  const [curPw, setCurPw] = useState("");
  const [newPw, setNewPw] = useState("");
  const [pwMsg, setPwMsg] = useState<{ ok: boolean; text: string } | null>(null);
  const [pwSaving, setPwSaving] = useState(false);

  useEffect(() => {
    if (ready && !affiliate) router.replace("/affiliate");
  }, [ready, affiliate, router]);

  const refresh = useCallback(async () => {
    const [s, o] = await Promise.all([affiliateMe(), affiliateOrders()]);
    if (s) setStats(s);
    setOrders(o);
  }, []);

  useEffect(() => {
    if (!affiliate) return;
    refresh();
    // Refetch when the tab regains focus — "real time" enough for payouts.
    const onFocus = () => refresh();
    window.addEventListener("focus", onFocus);
    return () => window.removeEventListener("focus", onFocus);
  }, [affiliate, refresh]);

  if (!ready || !affiliate) return null;

  const link = `${typeof window !== "undefined" ? window.location.origin : ""}/?ref=${affiliate.code}`;

  async function copy(text: string, which: "code" | "link") {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(which);
      setTimeout(() => setCopied(null), 1800);
    } catch {
      /* ignore */
    }
  }

  async function onChangePassword(e: React.FormEvent) {
    e.preventDefault();
    setPwMsg(null);
    setPwSaving(true);
    const res = await affiliateChangePassword(curPw, newPw);
    setPwSaving(false);
    if (res.ok) {
      setPwMsg({ ok: true, text: "เปลี่ยนรหัสผ่านเรียบร้อย" });
      setCurPw("");
      setNewPw("");
    } else {
      setPwMsg({ ok: false, text: res.error || "บันทึกไม่สำเร็จ" });
    }
  }

  const cards = [
    { label: "คลิกลิงก์", value: String(stats?.clicks ?? 0), hint: "จากลิงก์ ?ref ของคุณ" },
    { label: "ออเดอร์", value: String(stats?.orders_count ?? 0), hint: "ที่มาจากการแนะนำ" },
    { label: "รอส่งมอบ", value: money(stats?.pending_amount ?? 0), hint: "ยืนยันเมื่อส่งสำเร็จ" },
    { label: "รอรับเงิน", value: money(stats?.confirmed_amount ?? 0), hint: "ยืนยันแล้ว รอร้านโอน", hi: true },
    { label: "รับแล้ว", value: money(stats?.paid_amount ?? 0), hint: "จ่ายให้คุณแล้วทั้งหมด" },
  ];

  return (
    <main className={styles.page}>
      <div className={styles.dashHead}>
        <div>
          <span className="kicker">Affiliate Program</span>
          <h1 className={`display ${styles.title}`}>
            สวัสดี, <em>{affiliate.name}</em>
          </h1>
          <div className={styles.who}>
            โค้ดของคุณ {affiliate.code} · ค่าคอมมาตรฐาน {affiliate.commission_percent}%
          </div>
        </div>
        <button className={styles.signout} onClick={() => { signOut(); router.replace("/affiliate"); }}>
          ออกจากระบบ
        </button>
      </div>

      <div className={styles.statGrid}>
        {cards.map((c) => (
          <div key={c.label} className={`${styles.stat} ${c.hi ? styles.statHi : ""}`}>
            <div className={styles.statLabel}>{c.label}</div>
            <div className={styles.statValue}>{c.value}</div>
            <div className={styles.statHint}>{c.hint}</div>
          </div>
        ))}
      </div>

      <div className={styles.share}>
        <span className={styles.shareTitle}>แชร์แล้วรับค่าคอม</span>
        <div className={styles.shareRow}>
          <span className={styles.shareCode}>{affiliate.code}</span>
          <button className={styles.copyBtn} onClick={() => copy(affiliate.code, "code")}>
            {copied === "code" ? "Copied!" : "Copy Code"}
          </button>
        </div>
        <div className={styles.shareRow}>
          <span className={styles.shareLink}>{link}</span>
          <button className={styles.copyBtn} onClick={() => copy(link, "link")}>
            {copied === "link" ? "Copied!" : "Copy Link"}
          </button>
        </div>
      </div>

      <h2 className={styles.sectionTitle}>ออเดอร์จากการแนะนำ</h2>
      {orders.length === 0 ? (
        <p className={styles.empty}>
          ยังไม่มีออเดอร์ — แชร์ลิงก์ของคุณได้เลย ระบบจำผู้กดลิงก์ให้ 30 วัน
        </p>
      ) : (
        <div className={styles.tableWrap}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th>ออเดอร์</th>
                <th>วันที่</th>
                <th className={styles.num}>ยอดออเดอร์</th>
                <th className={styles.num}>ค่าคอมของคุณ</th>
                <th>สถานะ</th>
              </tr>
            </thead>
            <tbody>
              {orders.map((o) => {
                const st = STATUS_LABEL[o.commission_status] || { label: o.commission_status, cls: "" };
                return (
                  <tr key={`${o.order_id}-${o.commission_status}`}>
                    <td>#{o.order_id}</td>
                    <td>
                      {new Date(o.created_at).toLocaleDateString("th-TH", {
                        year: "2-digit", month: "short", day: "numeric",
                      })}
                    </td>
                    <td className={styles.num}>{money(o.order_total)}</td>
                    <td className={styles.num}>{money(o.commission)}</td>
                    <td>
                      <span className={`${styles.badge} ${styles[st.cls] || ""}`}>{st.label}</span>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      <div className={styles.panelRow}>
        <div className={styles.panel}>
          <h3 className={styles.panelTitle}>เปลี่ยนรหัสผ่าน</h3>
          <form className={styles.form} onSubmit={onChangePassword}>
            <label className={styles.field}>
              <span>รหัสผ่านปัจจุบัน</span>
              <input type="password" value={curPw} onChange={(e) => setCurPw(e.target.value)} required autoComplete="current-password" />
            </label>
            <label className={styles.field}>
              <span>รหัสผ่านใหม่ (อย่างน้อย 6 ตัวอักษร)</span>
              <input type="password" value={newPw} onChange={(e) => setNewPw(e.target.value)} required minLength={6} autoComplete="new-password" />
            </label>
            {pwMsg && <p className={pwMsg.ok ? styles.success : styles.error}>{pwMsg.text}</p>}
            <button className={styles.submit} type="submit" disabled={pwSaving}>
              {pwSaving ? "กำลังบันทึก…" : "บันทึก"}
            </button>
          </form>
        </div>
        <div className={styles.panel}>
          <h3 className={styles.panelTitle}>เงื่อนไขค่าคอมมิชชั่น</h3>
          <p className={styles.empty} style={{ padding: 0 }}>
            ค่าคอมคิดจากยอดที่ลูกค้าจ่ายจริง (หลังหักส่วนลด) และจะ
            <strong> ยืนยันเมื่อออเดอร์จัดส่งสำเร็จ</strong> —
            ออเดอร์ที่ยกเลิกจะไม่ถูกนับ ร้านจะสรุปและโอนค่าคอมที่ยืนยันแล้วให้เป็นรอบ ๆ
            มีคำถามทักแชทหาร้านได้เลย
          </p>
        </div>
      </div>
    </main>
  );
}
