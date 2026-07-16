"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { memberLogin, memberRegister } from "@/lib/api";
import { useMember } from "@/lib/member";
import styles from "./member.module.css";

type Mode = "login" | "register";

export default function MemberPage() {
  const router = useRouter();
  const { member, ready, signIn } = useMember();
  const [mode, setMode] = useState<Mode>("login");
  const [form, setForm] = useState({
    name: "",
    phone: "",
    email: "",
    password: "",
    confirm: "",
  });
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  // Already signed in — go straight to the account page.
  useEffect(() => {
    if (ready && member) router.replace("/member/account");
  }, [ready, member, router]);

  function update(field: keyof typeof form) {
    return (e: React.ChangeEvent<HTMLInputElement>) =>
      setForm((f) => ({ ...f, [field]: e.target.value }));
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);

    if (mode === "register" && form.password !== form.confirm) {
      setError("รหัสผ่านทั้งสองช่องไม่ตรงกัน");
      return;
    }

    setSubmitting(true);
    const res =
      mode === "login"
        ? await memberLogin(form.phone.trim(), form.password)
        : await memberRegister({
            name: form.name.trim(),
            phone: form.phone.trim(),
            email: form.email.trim() || undefined,
            password: form.password,
          });
    setSubmitting(false);

    if (res.ok && res.token && res.member) {
      signIn(res.token, res.member);
      router.replace("/member/account");
    } else {
      setError(res.error || "ไม่สำเร็จ กรุณาลองใหม่");
    }
  }

  return (
    <main className={styles.page}>
      <header className={styles.head}>
        <span className="kicker">Membership</span>
        <h1 className={`display ${styles.title}`}>
          The <em>Collective</em> Circle
        </h1>
        <p className={styles.lede}>
          สมาชิก Bruno Collective รับส่วนลด 5% ทุกคำสั่งซื้อ —
          และยังใช้โค้ดส่วนลดเพิ่มได้อีกต่อหนึ่ง
          Members enjoy 5% off every order, on top of any coupon.
        </p>
      </header>

      <div className={styles.card}>
        <div className={styles.perk}>
          <span className={styles.perkBadge}>−5%</span>
          <span className={styles.perkCopy}>
            ส่วนลดสมาชิกทุกออเดอร์ · ใช้ร่วมกับคูปองได้ · ดูประวัติคำสั่งซื้อย้อนหลัง
          </span>
        </div>

        <div className={styles.tabs} role="tablist">
          <button
            role="tab"
            aria-selected={mode === "login"}
            className={`${styles.tab} ${mode === "login" ? styles.tabActive : ""}`}
            onClick={() => {
              setMode("login");
              setError(null);
            }}
          >
            เข้าสู่ระบบ · Sign In
          </button>
          <button
            role="tab"
            aria-selected={mode === "register"}
            className={`${styles.tab} ${mode === "register" ? styles.tabActive : ""}`}
            onClick={() => {
              setMode("register");
              setError(null);
            }}
          >
            สมัครสมาชิก · Join
          </button>
        </div>

        <form className={styles.form} onSubmit={onSubmit}>
          {mode === "register" && (
            <label className={styles.field}>
              <span>ชื่อ-นามสกุล · Full Name *</span>
              <input value={form.name} onChange={update("name")} required />
            </label>
          )}
          <label className={styles.field}>
            <span>เบอร์โทรศัพท์ · Phone *</span>
            <input
              value={form.phone}
              onChange={update("phone")}
              required
              inputMode="tel"
              autoComplete="tel"
            />
          </label>
          {mode === "register" && (
            <label className={styles.field}>
              <span>อีเมล · Email</span>
              <input value={form.email} onChange={update("email")} type="email" />
            </label>
          )}
          <label className={styles.field}>
            <span>รหัสผ่าน · Password *</span>
            <input
              value={form.password}
              onChange={update("password")}
              type="password"
              required
              minLength={6}
              autoComplete={mode === "login" ? "current-password" : "new-password"}
            />
          </label>
          {mode === "register" && (
            <label className={styles.field}>
              <span>ยืนยันรหัสผ่าน · Confirm Password *</span>
              <input
                value={form.confirm}
                onChange={update("confirm")}
                type="password"
                required
                minLength={6}
                autoComplete="new-password"
              />
            </label>
          )}

          {error && <p className={styles.error}>{error}</p>}

          <button type="submit" className={styles.submit} disabled={submitting}>
            {submitting
              ? "…"
              : mode === "login"
                ? "เข้าสู่ระบบ"
                : "สมัครสมาชิก — รับส่วนลด 5%"}{" "}
            <span className="arrow">→</span>
          </button>
        </form>
      </div>
    </main>
  );
}
