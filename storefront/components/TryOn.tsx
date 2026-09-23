"use client";

import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import { generateTryOn, getTryOnStatus, type TryOnStatus } from "@/lib/api";
import { imageSrc } from "@/lib/format";
import type { Product } from "@/lib/types";
import styles from "./TryOn.module.css";

// Product types the try-on makes no sense for (printed matter, gift cards).
function isWearable(category: string): boolean {
  const c = (category || "").toLowerCase();
  return !["poster", "โปสเตอร์", "booklet", "book", "หนังสือ", "gift", "บัตร", "voucher"].some((k) =>
    c.includes(k)
  );
}

// Downscale + re-encode the chosen photo in the browser so uploads from
// phones (3–8MB HEIC/JPEG) become a ~200KB JPEG. Drawing through an <img>
// also bakes in the EXIF orientation, so the model never sees a sideways
// photo.
async function shrinkPhoto(file: File, maxDim = 1280): Promise<Blob> {
  const url = URL.createObjectURL(file);
  try {
    const img = await new Promise<HTMLImageElement>((resolve, reject) => {
      const el = new Image();
      el.onload = () => resolve(el);
      el.onerror = () => reject(new Error("decode"));
      el.src = url;
    });
    const scale = Math.min(1, maxDim / Math.max(img.naturalWidth, img.naturalHeight));
    const w = Math.round(img.naturalWidth * scale);
    const h = Math.round(img.naturalHeight * scale);
    const canvas = document.createElement("canvas");
    canvas.width = w;
    canvas.height = h;
    const ctx = canvas.getContext("2d");
    if (!ctx) throw new Error("canvas");
    ctx.fillStyle = "#fff";
    ctx.fillRect(0, 0, w, h);
    ctx.drawImage(img, 0, 0, w, h);
    const blob = await new Promise<Blob | null>((r) => canvas.toBlob(r, "image/jpeg", 0.86));
    if (!blob) throw new Error("encode");
    return blob;
  } finally {
    URL.revokeObjectURL(url);
  }
}

type Step = "pick" | "working" | "result";

// What kind of photo works best, by product category.
function photoHint(category: string): string {
  const c = (category || "").toLowerCase();
  if (c.includes("แหวน") || c.includes("ring")) return "ถ่ายมือใกล้ ๆ ให้เห็นนิ้วชัด แสงสว่าง";
  if (c.includes("สร้อย") || c.includes("เครื่องประดับ") || c.includes("jewel") || c.includes("bracelet") || c.includes("necklace"))
    return "ถ่ายใกล้ ๆ บริเวณที่จะใส่ (มือ ข้อมือ หรือคอ) แสงสว่าง";
  if (c.includes("รองเท้า") || c.includes("shoe")) return "ยืนเต็มตัว เห็นเท้าชัด แสงสว่าง";
  return "ยืนตรง เห็นตัวชัด แสงสว่าง ไม่มีคนอื่นในรูป";
}

export default function TryOn({ product }: { product: Product }) {
  const [status, setStatus] = useState<TryOnStatus | null>(null);
  const [open, setOpen] = useState(false);
  const [step, setStep] = useState<Step>("pick");
  const [photo, setPhoto] = useState<Blob | null>(null);
  const [photoUrl, setPhotoUrl] = useState<string | null>(null);
  const [result, setResult] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [elapsed, setElapsed] = useState(0);
  const fileInput = useRef<HTMLInputElement>(null);

  // Product images the shopper can try (first = primary).
  const gallery = Array.from(
    new Set([product.image_url, ...(product.images || [])].filter(Boolean))
  );
  const [garment, setGarment] = useState<string>(gallery[0] || "");

  useEffect(() => {
    if (!isWearable(product.category)) return;
    getTryOnStatus().then(setStatus);
  }, [product.category]);

  // Lock page scroll while the dialog is open.
  useEffect(() => {
    if (!open) return;
    const prev = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && setOpen(false);
    window.addEventListener("keydown", onKey);
    return () => {
      document.body.style.overflow = prev;
      window.removeEventListener("keydown", onKey);
    };
  }, [open]);

  useEffect(() => () => {
    if (photoUrl) URL.revokeObjectURL(photoUrl);
  }, [photoUrl]);

  if (!status?.enabled || !gallery.length) return null;

  const remaining = status.remaining ?? 0;
  const exhausted = remaining <= 0;

  async function onPick(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    e.target.value = "";
    if (!file) return;
    setError(null);
    try {
      const blob = await shrinkPhoto(file);
      if (photoUrl) URL.revokeObjectURL(photoUrl);
      setPhoto(blob);
      setPhotoUrl(URL.createObjectURL(blob));
      setResult(null);
      setStep("pick");
    } catch {
      setError("เปิดรูปไม่ได้ ลองเลือกไฟล์ JPG หรือ PNG");
    }
  }

  async function run() {
    if (!photo) return;
    setError(null);
    setElapsed(0);
    setStep("working");
    const res = await generateTryOn(product.id, photo, garment, setElapsed);
    if (res.remaining != null) setStatus((s) => (s ? { ...s, remaining: res.remaining } : s));
    if (!res.ok || !res.image) {
      setError(res.error || "สร้างภาพไม่สำเร็จ");
      setStep("pick");
      return;
    }
    setResult(res.image);
    setStep("result");
  }

  function reset() {
    setResult(null);
    setError(null);
    setStep("pick");
  }

  return (
    <>
      <button type="button" className={styles.trigger} onClick={() => setOpen(true)}>
        <span className={styles.triggerIcon} aria-hidden>✦</span>
        <span>
          <strong>ลองใส่ดูก่อน — Virtual Try-On</strong>
          <small>อัปโหลดรูปตัวเอง แล้วดูว่าใส่ชิ้นนี้เป็นยังไง (ใช้เวลาประมาณ 1 นาที)</small>
        </span>
        <span className="arrow">→</span>
      </button>

      {open && (
        <div className={styles.scrim} onClick={() => setOpen(false)}>
          <div
            className={styles.dialog}
            role="dialog"
            aria-modal="true"
            aria-label="Virtual try-on"
            onClick={(e) => e.stopPropagation()}
          >
            <div className={styles.head}>
              <div>
                <div className={styles.kicker}>Virtual Try-On</div>
                <div className={styles.title}>{product.name}</div>
              </div>
              <button type="button" className={styles.close} onClick={() => setOpen(false)} aria-label="Close">
                ✕
              </button>
            </div>

            <div className={styles.body}>
              {step === "result" && result ? (
                <div className={styles.resultWrap}>
                  {/* eslint-disable-next-line @next/next/no-img-element */}
                  <img src={result} alt={`คุณใส่ ${product.name}`} className={styles.resultImg} />
                  <p className={styles.fine}>
                    ภาพจำลองด้วย AI — สี ทรง และความพอดีตัวอาจต่างจากของจริงเล็กน้อย
                  </p>
                </div>
              ) : (
                <>
                  <div className={styles.grid}>
                    <div className={styles.pane}>
                      <div className={styles.paneLabel}>1 · รูปของคุณ</div>
                      <button
                        type="button"
                        className={`${styles.drop} ${photoUrl ? styles.dropHas : ""}`}
                        onClick={() => fileInput.current?.click()}
                        disabled={step === "working"}
                      >
                        {photoUrl ? (
                          // eslint-disable-next-line @next/next/no-img-element
                          <img src={photoUrl} alt="รูปของคุณ" />
                        ) : (
                          <span className={styles.dropHint}>
                            <strong>เลือกรูป / ถ่ายรูป</strong>
                            <small>{photoHint(product.category)}</small>
                          </span>
                        )}
                      </button>
                      <input
                        ref={fileInput}
                        type="file"
                        accept="image/*"
                        hidden
                        onChange={onPick}
                      />
                      {photoUrl && step !== "working" && (
                        <button type="button" className={styles.link} onClick={() => fileInput.current?.click()}>
                          เปลี่ยนรูป
                        </button>
                      )}
                    </div>

                    <div className={styles.pane}>
                      <div className={styles.paneLabel}>2 · สินค้า</div>
                      <div className={styles.garment}>
                        {/* eslint-disable-next-line @next/next/no-img-element */}
                        <img src={imageSrc(garment)} alt={product.name} />
                      </div>
                      {gallery.length > 1 && (
                        <div className={styles.thumbs}>
                          {gallery.map((g) => (
                            <button
                              key={g}
                              type="button"
                              className={`${styles.thumb} ${g === garment ? styles.thumbOn : ""}`}
                              onClick={() => setGarment(g)}
                              disabled={step === "working"}
                              aria-label="เลือกรูปสินค้า"
                            >
                              {/* eslint-disable-next-line @next/next/no-img-element */}
                              <img src={imageSrc(g)} alt="" />
                            </button>
                          ))}
                        </div>
                      )}
                    </div>
                  </div>

                  {step === "working" && (
                    <div className={styles.working}>
                      <span className={styles.spinner} aria-hidden />
                      กำลังสร้างภาพ… ใช้เวลาประมาณ 1 นาที{elapsed > 0 ? ` (${elapsed} วิ)` : ""}
                    </div>
                  )}
                </>
              )}

              {error && <p className={styles.error}>{error}</p>}
            </div>

            <div className={styles.foot}>
              <div className={styles.quota}>
                {exhausted ? (
                  status.member ? (
                    "วันนี้ครบจำนวนแล้ว พรุ่งนี้ลองใหม่ได้"
                  ) : (
                    <>
                      วันนี้ครบจำนวนแล้ว — <Link href="/member">สมัครสมาชิก</Link> เพื่อลองได้มากขึ้น
                    </>
                  )
                ) : (
                  `ลองได้อีก ${remaining} ครั้งวันนี้`
                )}
                <span className={styles.privacy}>รูปของคุณใช้สร้างภาพเท่านั้น ไม่ถูกเก็บไว้</span>
              </div>
              {step === "result" ? (
                <div className={styles.actions}>
                  <button type="button" className={styles.ghost} onClick={reset}>
                    ลองรูปอื่น
                  </button>
                  <a className={styles.primary} href={result!} download={`bruno-tryon-${product.slug || product.id}.png`}>
                    บันทึกรูป
                  </a>
                </div>
              ) : (
                <button
                  type="button"
                  className={styles.primary}
                  onClick={run}
                  disabled={!photo || step === "working" || exhausted}
                >
                  {step === "working" ? "กำลังสร้าง…" : "ลองใส่เลย"} <span className="arrow">→</span>
                </button>
              )}
            </div>
          </div>
        </div>
      )}
    </>
  );
}
