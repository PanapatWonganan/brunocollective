"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  createLiveTryOnToken,
  endLiveTryOnSession,
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

// Person presence (MediaPipe pose, same approach as Decart's example): poll
// the local camera once a second; after NO_PERSON_MISSES misses in a row
// the session stops so an empty frame never burns streaming time.
const DETECT_INTERVAL_MS = 1000;
const NO_PERSON_MISSES = 3;
interface PoseDetector {
  detectForVideo(video: HTMLVideoElement, ts: number): { landmarks: unknown[] };
  close(): void;
}
async function loadPoseDetector(): Promise<PoseDetector> {
  const { PoseLandmarker, FilesetResolver } = await import("@mediapipe/tasks-vision");
  const vision = await FilesetResolver.forVisionTasks(
    "https://cdn.jsdelivr.net/npm/@mediapipe/tasks-vision@0.10.35/wasm"
  );
  return PoseLandmarker.createFromOptions(vision, {
    baseOptions: {
      modelAssetPath:
        "https://storage.googleapis.com/mediapipe-models/pose_landmarker/pose_landmarker_lite/float16/1/pose_landmarker_lite.task",
      delegate: "GPU",
    },
    runningMode: "VIDEO",
    numPoses: 1,
  });
}

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
  const detector = useRef<PoseDetector | null>(null);
  const detectTimer = useRef<number | null>(null);
  const sessionId = useRef<number | null>(null);
  const billedSec = useRef(0);
  const pathname = usePathname();
  const memberHref = `/member?next=${encodeURIComponent(pathname || "/")}`;

  useEffect(() => {
    if (!isGarment(product.category)) return;
    getLiveTryOnStatus().then(setStatus);
  }, [product.category]);

  const stopAll = useCallback((next: Phase, reason = "closed") => {
    if (timer.current) {
      window.clearInterval(timer.current);
      timer.current = null;
    }
    if (detectTimer.current) {
      window.clearInterval(detectTimer.current);
      detectTimer.current = null;
    }
    const client = rt.current;
    rt.current = null;
    // Cost tracking: report what the SDK says was generated (once).
    if (sessionId.current != null) {
      endLiveTryOnSession(sessionId.current, billedSec.current, reason);
      sessionId.current = null;
    }
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

  // Never leave a stream running when the page unmounts, and stop the
  // moment the tab is hidden (phone locked, app switched) — billing is per
  // second whether or not anyone is watching.
  useEffect(() => {
    const onVis = () => {
      if (document.visibilityState === "hidden" && rt.current) {
        stopAll("ended", "hidden");
        setNote("หยุดเพราะสลับหน้าจอ — กดลองอีกครั้งได้");
      }
    };
    document.addEventListener("visibilitychange", onVis);
    return () => {
      document.removeEventListener("visibilitychange", onVis);
      stopAll("idle", "unmount");
      detector.current?.close();
      detector.current = null;
    };
  }, [stopAll]);

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
    billedSec.current = 0;
    try {
      // 0. Warm the person detector in parallel (non-fatal if it fails).
      const detectorReady = detector.current
        ? Promise.resolve(detector.current)
        : loadPoseDetector().then((d) => (detector.current = d)).catch(() => null);

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
      const tok = await createLiveTryOnToken(product.id);
      if (!tok.ok || !tok.api_key) {
        if (tok.remaining != null) setStatus((s) => (s ? { ...s, remaining: tok.remaining } : s));
        throw new Error(tok.error || "ระบบลองใส่สดขัดข้อง");
      }
      setStatus((s) => (s ? { ...s, remaining: tok.remaining ?? s.remaining } : s));
      sessionId.current = tok.session_id ?? null;
      const cap = tok.session_seconds || status?.session_seconds || 20;

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
      // Billed time as the SDK sees it (used for the cost log).
      rtc.on("generationTick", (t) => {
        const s = (t as { seconds?: number })?.seconds;
        if (typeof s === "number") billedSec.current = s;
      });
      rtc.on("generationEnded", (t) => {
        const s = (t as { seconds?: number })?.seconds;
        if (typeof s === "number") billedSec.current = s;
      });
      // Server-side end (session cap reached, quota) — terminal, no reconnect.
      rtc.on("sessionEnded", () => {
        if (rt.current) stopAll("ended", "server");
      });
      rtc.on("error", () => setNote("สัญญาณสะดุด กำลังเชื่อมต่อใหม่…"));

      // Person watchdog: stop when nobody has been in frame for a few seconds.
      detectorReady.then((d) => {
        if (!d || !rt.current || !localVideo.current) return;
        let misses = 0;
        detectTimer.current = window.setInterval(() => {
          const v = localVideo.current;
          if (!v || v.readyState < 2 || !rt.current) return;
          try {
            const res = d.detectForVideo(v, performance.now());
            if (res.landmarks.length > 0) {
              misses = 0;
            } else if (++misses >= NO_PERSON_MISSES) {
              stopAll("ended", "no_person");
              setNote("ไม่เห็นคนในเฟรม เลยหยุดให้เพื่อไม่ให้เสียเวลาลอง — กดลองอีกครั้งได้");
            }
          } catch {
            /* detector hiccup — ignore this tick */
          }
        }, DETECT_INTERVAL_MS);
      });

      // 4. Countdown mirrors the server-side cap Decart enforces.
      setSecondsLeft(cap);
      setPhase("live");
      const startedAt = Date.now();
      timer.current = window.setInterval(() => {
        const left = Math.max(0, cap - Math.round((Date.now() - startedAt) / 1000));
        setSecondsLeft(left);
        if (left <= 0) stopAll("ended", "timeout");
      }, 500);
    } catch (e) {
      const msg = e instanceof Error ? e.message : "";
      stopAll("error", "error");
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
  const sessionSec = status.session_seconds ?? 20;
  const needsMember = !!status.members_only && !status.member;

  return (
    <>
      <button type="button" className={styles.trigger} onClick={() => setOpen(true)}>
        <span className={styles.triggerIcon} aria-hidden>◉</span>
        <span>
          <strong>ลองใส่สดผ่านกล้อง — Live Try-On</strong>
          <small>
            เปิดกล้องหน้า แล้วเห็นตัวเองใส่ชิ้นนี้ทันที ({sessionSec} วินาทีต่อครั้ง
            {needsMember ? " · สำหรับสมาชิก" : ""})
          </small>
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
                      {needsMember
                        ? "สมัครสมาชิกฟรี แล้วเปิดกล้องลองใส่ได้ทันที"
                        : "กด \"เริ่ม\" แล้วอนุญาตให้ใช้กล้อง — ยืนให้เห็นลำตัว แสงสว่าง"}
                      <br />
                      ภาพจากกล้องจะถูกส่งไปประมวลผลสด ไม่มีการบันทึกไว้ และจะหยุดเองเมื่อไม่เห็นคนในเฟรม
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
                {needsMember ? (
                  <>
                    ลองใส่สดเปิดให้สมาชิกเท่านั้น — สมัครฟรี ได้ส่วนลด 5% ทุกออเดอร์ด้วย
                  </>
                ) : exhausted ? (
                  "วันนี้ครบจำนวนแล้ว พรุ่งนี้ลองใหม่ได้"
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
              ) : needsMember ? (
                <Link href={memberHref} className={styles.primary}>
                  สมัคร / เข้าสู่ระบบสมาชิก <span className="arrow">→</span>
                </Link>
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
