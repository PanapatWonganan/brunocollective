"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import Link from "next/link";
import {
  createLiveTryOnToken,
  getLiveTryOnStatus,
  type LiveTryOnStatus,
} from "@/lib/api";
import { imageSrc } from "@/lib/format";
import type { Product } from "@/lib/types";
import styles from "./LiveTryOn.module.css";

// Live try-on is a garment model: it dresses the upper/lower body from a
// webcam feed. It is poor at jewellery, bags, hats and shoes (per Decart), so
// those categories only get the photo try-on.
function isGarment(category: string): boolean {
  const c = (category || "").toLowerCase();
  const blocked = [
    "แหวน", "ring", "เครื่องประดับ", "สร้อย", "jewel", "bracelet", "necklace",
    "กระเป๋า", "bag", "หมวก", "hat", "cap", "รองเท้า", "shoe",
    "poster", "โปสเตอร์", "booklet", "book", "หนังสือ", "gift", "บัตร", "voucher",
  ];
  return !blocked.some((k) => c.includes(k));
}

// Decart's VTON prompting guide: one action + the body region + let the
// reference image lead. We name the region from the category.
function garmentPrompt(product: Product): string {
  const c = (product.category || "").toLowerCase();
  let region = "upper body garment";
  if (["กางเกง", "กระโปรง", "pant", "trouser", "short", "skirt", "jean"].some((k) => c.includes(k))) {
    region = "lower body garment";
  } else if (["เดรส", "dress", "ชุด", "jumpsuit"].some((k) => c.includes(k))) {
    region = "outfit";
  }
  return `Substitute the ${region} with the garment shown in the reference image (${product.name}), keeping its exact colour, fabric, fit, sleeve length, neckline and any visible logo.`;
}

// Minimal view of the Decart realtime client we use (the SDK is loaded
// lazily so it stays out of the main bundle).
interface RtClient {
  set(input: { prompt: string; image: Blob | null; enhance: boolean }): Promise<void>;
  disconnect(): void;
  on(event: string, fn: (payload: unknown) => void): void;
}

type RealtimeModelId = "lucy-vton-latest" | "lucy-vton-3.5";
function realtimeModelId(id: string | undefined): RealtimeModelId {
  return id === "lucy-vton-3.5" ? id : "lucy-vton-latest";
}

type Phase = "idle" | "starting" | "live" | "ended" | "error";

export default function LiveTryOn({ product }: { product: Product }) {
  const [status, setStatus] = useState<LiveTryOnStatus | null>(null);
  const [open, setOpen] = useState(false);
  const [phase, setPhase] = useState<Phase>("idle");
  const [error, setError] = useState<string | null>(null);
  const [note, setNote] = useState<string>("");
  const [secondsLeft, setSecondsLeft] = useState(0);
  const [applying, setApplying] = useState(false);

  const gallery = Array.from(
    new Set([product.image_url, ...(product.images || [])].filter(Boolean))
  );
  const [garment, setGarment] = useState<string>(gallery[0] || "");

  const outVideo = useRef<HTMLVideoElement>(null);
  const localVideo = useRef<HTMLVideoElement>(null);
  const rt = useRef<RtClient | null>(null);
  const cam = useRef<MediaStream | null>(null);
  const timer = useRef<number | null>(null);

  useEffect(() => {
    if (!isGarment(product.category)) return;
    getLiveTryOnStatus().then(setStatus);
  }, [product.category]);

  const stopAll = useCallback((next: Phase) => {
    if (timer.current) {
      window.clearInterval(timer.current);
      timer.current = null;
    }
    const client = rt.current;
    rt.current = null;
    try {
      client?.disconnect();
    } catch {
      /* already closed */
    }
    cam.current?.getTracks().forEach((t) => t.stop());
    cam.current = null;
    if (outVideo.current) outVideo.current.srcObject = null;
    if (localVideo.current) localVideo.current.srcObject = null;
    setPhase(next);
  }, []);

  // Close = hard stop (streaming is billed per second).
  function close() {
    stopAll("idle");
    setOpen(false);
    setError(null);
    setNote("");
    getLiveTryOnStatus().then(setStatus);
  }

  useEffect(() => {
    if (!open) return;
    const prev = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && close();
    window.addEventListener("keydown", onKey);
    return () => {
      document.body.style.overflow = prev;
      window.removeEventListener("keydown", onKey);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  // Never leave a stream running when the page unmounts.
  useEffect(() => () => stopAll("idle"), [stopAll]);

  // Clean "item only" reference (backend cuts the garment out of the product
  // photo and caches it) — Decart wants the garment alone on a plain
  // background; a photo of a model wearing it confuses the try-on.
  async function garmentBlob(url: string): Promise<Blob> {
    const res = await fetch(
      `/api/shop/products/${product.id}/garment?image=${encodeURIComponent(url)}`
    );
    if (!res.ok) throw new Error("garment");
    return res.blob();
  }

  async function start() {
    setError(null);
    setNote("");
    setPhase("starting");
    try {
      // 1. Camera first — if the shopper declines, no session is spent.
      const { createDecartClient, models } = await import("@decartai/sdk");
      const model = models.realtime("lucy-vton-latest");
      const fps = typeof model.fps === "number" ? model.fps : 25;
      const stream = await navigator.mediaDevices.getUserMedia({
        audio: false,
        video: { facingMode: "user", width: model.width, height: model.height, frameRate: fps },
      });
      cam.current = stream;
      if (localVideo.current) {
        localVideo.current.srcObject = stream;
        localVideo.current.play().catch(() => {});
      }

      // 2. Reference image first (the first request per image can take a
      //    while when the cutout is generated), then the session token.
      const blob = await garmentBlob(garment);
      const tok = await createLiveTryOnToken();
      if (!tok.ok || !tok.api_key) {
        if (tok.remaining != null) setStatus((s) => (s ? { ...s, remaining: tok.remaining } : s));
        throw new Error(tok.error || "ระบบลองใส่สดขัดข้อง");
      }
      setStatus((s) => (s ? { ...s, remaining: tok.remaining ?? s.remaining } : s));
      const cap = tok.session_seconds || status?.session_seconds || 45;

      // 3. Connect; the garment goes in as the initial state so the first
      //    frames already show it.
      const client = createDecartClient({ apiKey: tok.api_key });
      const rtc = (await client.realtime.connect(stream, {
        model: models.realtime(realtimeModelId(tok.model)),
        mirror: "auto",
        onRemoteStream: (remote: MediaStream) => {
          if (outVideo.current) {
            outVideo.current.srcObject = remote;
            outVideo.current.play().catch(() => {});
          }
        },
        onConnectionChange: (state: string) => {
          if (state === "reconnecting") setNote("สัญญาณสะดุด กำลังเชื่อมต่อใหม่…");
          else if (state === "connected" || state === "generating") setNote("");
          else if (state === "disconnected" && rt.current) stopAll("ended");
        },
        initialState: { prompt: { text: garmentPrompt(product), enhance: false }, image: blob },
      })) as unknown as RtClient;
      rt.current = rtc;
      // Server-side end (session cap reached, quota) — terminal, no reconnect.
      rtc.on("sessionEnded", () => {
        if (rt.current) stopAll("ended");
      });
      rtc.on("error", () => setNote("สัญญาณสะดุด กำลังเชื่อมต่อใหม่…"));

      // 4. Countdown mirrors the server-side cap Decart enforces.
      setSecondsLeft(cap);
      setPhase("live");
      const startedAt = Date.now();
      timer.current = window.setInterval(() => {
        const left = Math.max(0, cap - Math.round((Date.now() - startedAt) / 1000));
        setSecondsLeft(left);
        if (left <= 0) stopAll("ended");
      }, 500);
    } catch (e) {
      const msg = e instanceof Error ? e.message : "";
      stopAll("error");
      if (/NotAllowed|Permission|denied/i.test(msg)) {
        setError("ไม่ได้รับอนุญาตให้ใช้กล้อง — เปิดสิทธิ์กล้องให้เว็บนี้แล้วลองใหม่");
      } else if (/NotFound|Devices/i.test(msg)) {
        setError("ไม่พบกล้องในอุปกรณ์นี้");
      } else {
        setError(msg && !/^[A-Za-z]/.test(msg) ? msg : "เชื่อมต่อระบบลองใส่สดไม่สำเร็จ กรุณาลองใหม่");
      }
    }
  }

  async function switchGarment(url: string) {
    setGarment(url);
    if (!rt.current) return;
    setApplying(true);
    try {
      await rt.current.set({ prompt: garmentPrompt(product), image: await garmentBlob(url), enhance: false });
    } catch {
      setNote("เปลี่ยนรูปสินค้าไม่สำเร็จ");
    } finally {
      setApplying(false);
    }
  }

  function snapshot() {
    const v = outVideo.current;
    if (!v || !v.videoWidth) return;
    const canvas = document.createElement("canvas");
    canvas.width = v.videoWidth;
    canvas.height = v.videoHeight;
    canvas.getContext("2d")?.drawImage(v, 0, 0);
    const a = document.createElement("a");
    a.href = canvas.toDataURL("image/jpeg", 0.9);
    a.download = `bruno-live-${product.slug || product.id}.jpg`;
    a.click();
  }

  if (!status?.enabled || !gallery.length) return null;

  const remaining = status.remaining ?? 0;
  const exhausted = remaining <= 0;
  const sessionSec = status.session_seconds ?? 45;

  return (
    <>
      <button type="button" className={styles.trigger} onClick={() => setOpen(true)}>
        <span className={styles.triggerIcon} aria-hidden>◉</span>
        <span>
          <strong>ลองใส่สดผ่านกล้อง — Live Try-On</strong>
          <small>เปิดกล้องหน้า แล้วเห็นตัวเองใส่ชิ้นนี้ทันที ({sessionSec} วินาทีต่อครั้ง)</small>
        </span>
        <span className="arrow">→</span>
      </button>

      {open && (
        <div className={styles.scrim} onClick={close}>
          <div
            className={styles.dialog}
            role="dialog"
            aria-modal="true"
            aria-label="Live try-on"
            onClick={(e) => e.stopPropagation()}
          >
            <div className={styles.head}>
              <div>
                <div className={styles.kicker}>Live Try-On</div>
                <div className={styles.title}>{product.name}</div>
              </div>
              <button type="button" className={styles.close} onClick={close} aria-label="Close">
                ✕
              </button>
            </div>

            <div className={styles.body}>
              <div className={styles.stage}>
                {/* Output from the model; the local camera shows underneath until the first frame arrives. */}
                <video ref={localVideo} className={styles.localVideo} playsInline muted autoPlay />
                <video
                  ref={outVideo}
                  className={`${styles.outVideo} ${phase === "live" ? styles.outVideoOn : ""}`}
                  playsInline
                  muted
                  autoPlay
                />
                {phase === "idle" && (
                  <div className={styles.overlay}>
                    <p>
                      กด &quot;เริ่ม&quot; แล้วอนุญาตให้ใช้กล้อง — ยืนให้เห็นลำตัว แสงสว่าง
                      <br />
                      ภาพจากกล้องจะถูกส่งไปประมวลผลสด ไม่มีการบันทึกไว้
                    </p>
                  </div>
                )}
                {phase === "starting" && (
                  <div className={styles.overlay}>
                    <span className={styles.spinner} aria-hidden />
                    <p>กำลังเชื่อมต่อ… ภาพแรกใช้เวลา 5–10 วินาที</p>
                  </div>
                )}
                {phase === "ended" && (
                  <div className={styles.overlay}>
                    <p>หมดเวลาสำหรับรอบนี้แล้ว</p>
                  </div>
                )}
                {phase === "live" && (
                  <div className={styles.hud}>
                    <span className={styles.liveDot} aria-hidden />
                    LIVE · เหลือ {secondsLeft} วิ
                    {applying && " · กำลังเปลี่ยนชิ้น…"}
                  </div>
                )}
              </div>

              {gallery.length > 1 && (
                <div className={styles.thumbs}>
                  {gallery.map((g) => (
                    <button
                      key={g}
                      type="button"
                      className={`${styles.thumb} ${g === garment ? styles.thumbOn : ""}`}
                      onClick={() => switchGarment(g)}
                      disabled={phase === "starting" || applying}
                      aria-label="เลือกรูปสินค้า"
                    >
                      {/* eslint-disable-next-line @next/next/no-img-element */}
                      <img src={imageSrc(g)} alt="" />
                    </button>
                  ))}
                </div>
              )}

              {note && <p className={styles.note}>{note}</p>}
              {error && <p className={styles.error}>{error}</p>}
              <p className={styles.fine}>
                ภาพจำลองด้วย AI แบบเรียลไทม์ — สี ทรง และความพอดีตัวอาจต่างจากของจริง
              </p>
            </div>

            <div className={styles.foot}>
              <div className={styles.quota}>
                {exhausted ? (
                  status.member ? (
                    "วันนี้ครบจำนวนแล้ว พรุ่งนี้ลองใหม่ได้"
                  ) : (
                    <>
                      วันนี้ครบจำนวนแล้ว — <Link href="/member">สมัครสมาชิก</Link> เพื่อลองได้นานขึ้น
                    </>
                  )
                ) : (
                  `ลองได้อีก ${remaining} ครั้งวันนี้ · ครั้งละ ${sessionSec} วินาที`
                )}
                <span className={styles.privacy}>ภาพกล้องประมวลผลสดโดยผู้ให้บริการ AI (Decart) เราไม่บันทึกวิดีโอ</span>
              </div>
              {phase === "live" ? (
                <div className={styles.actions}>
                  <button type="button" className={styles.ghost} onClick={snapshot}>
                    ถ่ายภาพ
                  </button>
                  <button type="button" className={styles.primary} onClick={() => stopAll("ended")}>
                    หยุด
                  </button>
                </div>
              ) : (
                <button
                  type="button"
                  className={styles.primary}
                  onClick={start}
                  disabled={phase === "starting" || exhausted}
                >
                  {phase === "starting" ? "กำลังเชื่อมต่อ…" : phase === "idle" ? "เริ่ม" : "ลองอีกครั้ง"}{" "}
                  <span className="arrow">→</span>
                </button>
              )}
            </div>
          </div>
        </div>
      )}
    </>
  );
}
