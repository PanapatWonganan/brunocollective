import type { Metadata } from "next";
import Link from "next/link";
import styles from "../legal.module.css";

export const metadata: Metadata = {
  title: "วิธีขอลบข้อมูล (Data Deletion Instructions)",
  description:
    "ขั้นตอนการขอลบข้อมูลส่วนตัวของคุณจากระบบของ Bruno Collective",
};

export default function DataDeletionPage() {
  return (
    <main className={styles.page}>
      <p className={styles.kicker}>Bruno Collective</p>
      <h1 className={styles.title}>วิธีขอลบข้อมูลของคุณ</h1>
      <p className={styles.updated}>Data Deletion Instructions — ปรับปรุงล่าสุด 18 สิงหาคม 2026</p>

      <p>
        หากคุณต้องการให้เราลบข้อมูลส่วนตัวของคุณ — เช่น ประวัติการสั่งซื้อ
        ข้อมูลสมาชิก หรือข้อความแชทที่เคยติดต่อเราผ่าน Facebook, Instagram
        หรือ LINE — ทำตามขั้นตอนนี้ได้เลย
      </p>

      <h2>ขั้นตอนการขอลบข้อมูล</h2>
      <ul>
        <li>
          ติดต่อเราผ่านช่องทางใดก็ได้ด้านล่าง
          พร้อมแจ้งชื่อและเบอร์โทรศัพท์ที่ใช้สั่งซื้อ
          (หรือชื่อบัญชีที่ใช้ทักแชท)
        </li>
        <li>แจ้งว่า &ldquo;ขอลบข้อมูลส่วนตัว&rdquo;</li>
        <li>
          เราจะยืนยันตัวตน ดำเนินการลบข้อมูลของคุณออกจากระบบ
          และแจ้งผลกลับภายใน <strong>30 วัน</strong>
        </li>
      </ul>

      <h2>ช่องทางติดต่อ</h2>
      <ul>
        <li>Facebook Page: Bruno Collective (ส่งข้อความถึงเพจ)</li>
        <li>LINE Official Account: @brunocollective</li>
        <li>อีเมล: theapppresso@gmail.com</li>
      </ul>

      <h2>ข้อมูลที่จะถูกลบ</h2>
      <ul>
        <li>ข้อมูลลูกค้า: ชื่อ เบอร์โทรศัพท์ ที่อยู่ อีเมล และบัญชีสมาชิก</li>
        <li>ประวัติข้อความแชทและไฟล์แนบทุกช่องทาง</li>
        <li>
          ข้อมูลคำสั่งซื้ออาจถูกเก็บไว้เท่าที่กฎหมายกำหนด (เช่น หลักฐานทางบัญชี)
          โดยตัดข้อมูลที่ระบุตัวตนออก
        </li>
      </ul>

      <p>
        อ่านเพิ่มเติมได้ที่{" "}
        <Link href="/privacy" style={{ textDecoration: "underline" }}>
          นโยบายความเป็นส่วนตัว
        </Link>
      </p>

      <div className={styles.note}>
        <p>
          <strong>English summary:</strong> To request deletion of your personal
          data (orders, membership, chat history from Facebook/Instagram/LINE),
          message the Bruno Collective Facebook Page, LINE @brunocollective, or
          email theapppresso@gmail.com with the name and phone number you used.
          We will verify your identity, delete your data, and confirm within 30
          days.
        </p>
      </div>
    </main>
  );
}
