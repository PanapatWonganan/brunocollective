import type { Metadata } from "next";
import Link from "next/link";
import Reveal from "@/components/Reveal";
import styles from "./service.module.css";

export const metadata: Metadata = {
  title: "Client Services",
  description:
    "Shipping, size exchange, garment care, private appointments and how to reach Bruno Collective — จัดส่ง เปลี่ยนไซส์ การดูแลเสื้อผ้า และช่องทางติดต่อ",
};

// Client-services page — the real destination behind the footer's Service
// links (previously anchor-only). Copy is honest: no invented policies.
const SECTIONS = [
  {
    id: "shipping",
    num: "I.",
    title: "Shipping — การจัดส่ง",
    body: (
      <>
        <p>
          ทุกออเดอร์แพ็คด้วยมือจากสตูดิโอของเราที่ขอนแก่น และจัดส่งทั่วประเทศไทย
          พร้อมเลขติดตามพัสดุ — every order is packed by hand in Khon Kaen and
          shipped nationwide with a tracking number.
        </p>
        <p>
          หลังชำระเงินและแนบสลิปเรียบร้อย เราจะยืนยันออเดอร์และแจ้งเลขพัสดุให้ทางแชท
          หากต้องการของด่วนหรือส่งเป็นของขวัญ ทักแชทบอกเราได้เลย
        </p>
      </>
    ),
  },
  {
    id: "exchange",
    num: "II.",
    title: "Size Exchange — เปลี่ยนไซส์",
    body: (
      <>
        <p>
          ไซส์ไม่พอดี ไม่ต้องเก็บไว้ใส่ฝืน ๆ — ทักแชทหาเราได้เลย
          เราช่วยเปลี่ยนไซส์ให้จนกว่าจะพอดี (สินค้าต้องยังไม่ผ่านการซักและอยู่ในสภาพเดิม)
        </p>
        <p>
          ก่อนสั่ง แนะนำให้ดูตารางไซส์ในหน้าสินค้า หรือส่งรอบอก/ความยาวตัวโปรดของคุณมาในแชท
          เราช่วยเทียบไซส์ให้ก่อนตัดสินใจได้ — we would rather answer twice than
          ship the wrong size once.
        </p>
      </>
    ),
  },
  {
    id: "care",
    num: "III.",
    title: "Care & Repair — การดูแลรักษา",
    body: (
      <>
        <p>
          ซักเบา ๆ ด้วยน้ำอุณหภูมิปกติ ตากในที่ร่ม และรีดไฟอ่อนจากด้านในของตัวเสื้อ —
          treated gently, good cloth softens with age instead of wearing out.
        </p>
        <p>
          หลีกเลี่ยงเครื่องอบผ้าและการแช่ผงซักฟอกนาน ๆ
          หากตะเข็บหรือกระดุมมีปัญหาจากการตัดเย็บ ทักแชทมาพร้อมรูปถ่าย เราดูแลให้
        </p>
      </>
    ),
  },
  {
    id: "appointment",
    num: "IV.",
    title: "Private Appointment — นัดหมาย",
    body: (
      <>
        <p>
          สตูดิโอของเราที่ขอนแก่นเปิดรับนัดหมายส่วนตัว — ลองสัมผัสผ้าจริง ลองไซส์
          และเลือกชิ้นที่ใช่แบบไม่ต้องรีบ by appointment only.
        </p>
        <p>
          นัดล่วงหน้าทางแชทหรืออีเมลอย่างน้อย 1 วัน
          แล้วเราจะยืนยันวันเวลากลับไปครับ
        </p>
      </>
    ),
  },
  {
    id: "contact",
    num: "V.",
    title: "Contact — ติดต่อเรา",
    body: (
      <>
        <p>
          ช่องทางที่เร็วที่สุดคือแชท — LINE, Facebook หรือ Instagram ของร้าน
          เราอ่านและตอบเองทุกข้อความ every chat is answered by a human.
        </p>
        <p>
          อีเมล{" "}
          <a href="mailto:hello@brunocollective.co">hello@brunocollective.co</a>{" "}
          — สำหรับเรื่องทั่วไป สื่อ/press (
          <a href="mailto:hello@brunocollective.co?subject=Press%20Enquiry">
            Press Enquiries
          </a>
          ) และการสั่งซื้อจำนวนมาก (
          <a href="mailto:hello@brunocollective.co?subject=Wholesale">Wholesale</a>
          )
        </p>
      </>
    ),
  },
];

export default function ServicePage() {
  return (
    <main className={styles.page}>
      <header className={styles.head}>
        <span className="kicker">Client Services</span>
        <h1 className={`display ${styles.title}`}>
          How can we <em>help?</em>
        </h1>
        <p className={styles.sub}>
          จัดส่ง เปลี่ยนไซส์ การดูแลเสื้อผ้า และช่องทางติดต่อ — everything
          practical, in one quiet place.
        </p>
        <nav className={styles.toc} aria-label="Sections">
          {SECTIONS.map((s) => (
            <a key={s.id} href={`#${s.id}`}>
              {s.title.split(" — ")[0]}
            </a>
          ))}
        </nav>
      </header>

      <div className={styles.sections}>
        {SECTIONS.map((s) => (
          <Reveal as="div" key={s.id} className={styles.section}>
            <section id={s.id} className={styles.sectionIn}>
              <div className={styles.num}>{s.num}</div>
              <div className={styles.sectionBody}>
                <h2 className={styles.h2}>{s.title}</h2>
                {s.body}
              </div>
            </section>
          </Reveal>
        ))}
      </div>

      <div className={styles.cta}>
        <p className={styles.ctaNote}>ยังหาคำตอบไม่เจอ? ทักแชทหาเราได้เลย</p>
        <Link href="/shop" className="qlink">
          Back to the Collection <span className="arrow">→</span>
        </Link>
      </div>
    </main>
  );
}
