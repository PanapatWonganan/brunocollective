import Reveal from "@/components/Reveal";
import s from "./home.module.css";

export default function ServiceStrip() {
  return (
    <div className={s.svc}>
      <div className={s.svcIn}>
        <Reveal as="div" className={s.svcCell}>
          <h4>Ships across Thailand</h4>
          <p>แพ็คอย่างดี ส่งไวทั่วไทย — tracked on every order.</p>
        </Reveal>
        <Reveal as="div" delay={2} className={s.svcCell}>
          <h4>Easy size exchange</h4>
          <p>ไซส์ไม่พอดี ทักแชทเปลี่ยนได้ — we&apos;ll make it right.</p>
        </Reveal>
        <Reveal as="div" delay={3} className={s.svcCell}>
          <h4>Talk to a human</h4>
          <p>LINE · Facebook · Instagram — ตอบเองทุกแชท</p>
        </Reveal>
      </div>
    </div>
  );
}
