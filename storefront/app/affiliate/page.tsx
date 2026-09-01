"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { affiliateLogin } from "@/lib/api";
import { useAffiliate } from "@/lib/affiliate";
import styles from "./affiliate.module.css";

// Affiliate portal sign-in. No self-registration — accounts are created by
// the shop; partners sign in with the phone + password they were given.
export default function AffiliateLoginPage() {
  const router = useRouter();
  const { affiliate, ready, signIn } = useAffiliate();
  const [phone, setPhone] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (ready && affiliate) router.replace("/affiliate/dashboard");
  }, [ready, affiliate, router]);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setSubmitting(true);
    const res = await affiliateLogin(phone.trim(), password);
    setSubmitting(false);
    if (res.ok && res.token && res.affiliate) {
      signIn(res.token, res.affiliate);
      router.replace("/affiliate/dashboard");
    } else {
      setError(res.error || "เข้าสู่ระบบไม่สำเร็จ กรุณาลองใหม่");
    }
  }

  return (
    <main className={styles.page}>
      <header className={styles.head}>
        <span className="kicker">Affiliate Program</span>
        <h1 className={`display ${styles.title}`}>
          Partner <em>Portal.</em>
        </h1>
        <p className={styles.lede}>
          พื้นที่สำหรับพาร์ทเนอร์ของ Bruno Collective — เช็คยอดคลิก ยอดขาย
          และค่าคอมมิชชั่นของคุณแบบเรียลไทม์
          ยังไม่มีบัญชี? ทักแชทหาร้านเพื่อสมัครเป็นผู้แนะนำได้เลย
        </p>
      </header>

      <div className={styles.card}>
        <form className={styles.form} onSubmit={onSubmit}>
          <label className={styles.field}>
            <span>เบอร์โทรศัพท์ · Phone</span>
            <input
              value={phone}
              onChange={(e) => setPhone(e.target.value)}
              required
              inputMode="tel"
              autoComplete="tel"
            />
          </label>
          <label className={styles.field}>
            <span>รหัสผ่าน · Password</span>
            <input
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              type="password"
              autoComplete="current-password"
            />
          </label>
          {error && <p className={styles.error}>{error}</p>}
          <button className={styles.submit} type="submit" disabled={submitting}>
            {submitting ? "กำลังเข้าสู่ระบบ…" : "เข้าสู่ระบบ · Sign In"}
          </button>
        </form>
      </div>
    </main>
  );
}
