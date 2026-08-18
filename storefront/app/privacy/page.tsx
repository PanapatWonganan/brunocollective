import type { Metadata } from "next";
import Link from "next/link";
import styles from "../legal.module.css";

export const metadata: Metadata = {
  title: "นโยบายความเป็นส่วนตัว (Privacy Policy)",
  description:
    "นโยบายความเป็นส่วนตัวของ Bruno Collective — ข้อมูลที่เราเก็บ วัตถุประสงค์ และสิทธิ์ของคุณ",
};

export default function PrivacyPage() {
  return (
    <main className={styles.page}>
      <p className={styles.kicker}>Bruno Collective</p>
      <h1 className={styles.title}>นโยบายความเป็นส่วนตัว</h1>
      <p className={styles.updated}>Privacy Policy — ปรับปรุงล่าสุด 18 สิงหาคม 2026</p>

      <p>
        Bruno Collective (&ldquo;เรา&rdquo;) ให้ความสำคัญกับความเป็นส่วนตัวของลูกค้า
        นโยบายฉบับนี้อธิบายว่าเราเก็บข้อมูลอะไร ใช้ทำอะไร
        และคุณมีสิทธิ์อย่างไรบ้าง เมื่อคุณสั่งซื้อสินค้าหรือติดต่อเราผ่านเว็บไซต์
        brunocollective.io, Facebook, Instagram และ LINE
      </p>

      <h2>ข้อมูลที่เราเก็บ</h2>
      <ul>
        <li>
          <strong>ข้อมูลการสั่งซื้อ</strong> — ชื่อ เบอร์โทรศัพท์ ที่อยู่จัดส่ง
          รายการสินค้า และหลักฐานการชำระเงินที่คุณอัพโหลด
        </li>
        <li>
          <strong>ข้อมูลสมาชิก</strong> — ชื่อ เบอร์โทรศัพท์ อีเมล (ถ้าให้ไว้)
          และประวัติคำสั่งซื้อ สำหรับสิทธิ์ส่วนลดสมาชิก
        </li>
        <li>
          <strong>ข้อความแชท</strong> — เมื่อคุณทักเราผ่าน Facebook Messenger,
          Instagram หรือ LINE เราได้รับข้อความ ชื่อโปรไฟล์
          และรูปโปรไฟล์ของคุณจากแพลตฟอร์มนั้น ๆ
          เพื่อให้ทีมงานตอบกลับและดูแลคำสั่งซื้อของคุณได้
        </li>
      </ul>

      <h2>เราใช้ข้อมูลเพื่ออะไร</h2>
      <ul>
        <li>จัดการคำสั่งซื้อ จัดส่งสินค้า และแจ้งสถานะ</li>
        <li>ตอบคำถามและให้บริการลูกค้าผ่านช่องทางแชท</li>
        <li>มอบสิทธิประโยชน์สมาชิกและคูปองส่วนลด</li>
        <li>ปรับปรุงสินค้าและบริการจากภาพรวมการขาย (ไม่ระบุตัวตน)</li>
      </ul>

      <h2>การเปิดเผยข้อมูล</h2>
      <p>
        เราไม่ขายหรือให้เช่าข้อมูลส่วนตัวของคุณแก่บุคคลภายนอก
        ข้อมูลจะถูกเปิดเผยเท่าที่จำเป็นต่อการให้บริการเท่านั้น เช่น
        ชื่อ–ที่อยู่–เบอร์โทรสำหรับบริษัทขนส่ง
        และการรับส่งข้อความผ่านแพลตฟอร์มของ Meta (Facebook/Instagram) และ LINE
        ตามนโยบายของแพลตฟอร์มนั้น ๆ
      </p>

      <h2>ระยะเวลาเก็บรักษา</h2>
      <p>
        เราเก็บข้อมูลคำสั่งซื้อและข้อความแชทไว้ตราบเท่าที่จำเป็นต่อการให้บริการ
        การรับประกัน และภาระผูกพันทางกฎหมาย
        เมื่อพ้นความจำเป็นแล้วข้อมูลจะถูกลบหรือทำให้ไม่สามารถระบุตัวตนได้
      </p>

      <h2>สิทธิ์ของคุณ</h2>
      <p>
        คุณมีสิทธิ์ขอเข้าถึง แก้ไข หรือลบข้อมูลส่วนตัวของคุณได้ตลอดเวลา
        ดูขั้นตอนการขอลบข้อมูลได้ที่{" "}
        <Link href="/data-deletion" style={{ textDecoration: "underline" }}>
          วิธีขอลบข้อมูล
        </Link>
      </p>

      <h2>ติดต่อเรา</h2>
      <ul>
        <li>Facebook Page: Bruno Collective</li>
        <li>LINE Official Account: @brunocollective</li>
        <li>อีเมล: theapppresso@gmail.com</li>
      </ul>

      <div className={styles.note}>
        <p>
          <strong>English summary:</strong> Bruno Collective collects order details
          (name, phone, shipping address, payment slips) and chat messages sent to
          us via Facebook Messenger, Instagram, or LINE, solely to fulfil orders
          and provide customer support. We never sell personal data. To request
          access or deletion, see{" "}
          <Link href="/data-deletion" style={{ textDecoration: "underline" }}>
            /data-deletion
          </Link>{" "}
          or email theapppresso@gmail.com.
        </p>
      </div>
    </main>
  );
}
