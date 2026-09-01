import Reveal from "@/components/Reveal";
import s from "./home.module.css";

// The four brand values — honest copy about how the clothes are actually made
// (no invented heritage), in the design's dark numbered-cell layout.
const VALUES = [
  {
    num: "I.",
    title: "Cut with care",
    body: "ตัดเย็บทีละชิ้นอย่างประณีตในสตูดิโอของเราที่ขอนแก่น — finished by hand, checked twice.",
  },
  {
    num: "II.",
    title: "Considered fabrics",
    body: "ผ้าที่เลือกเพราะสวมสบายและอยู่ทน — chosen for hand-feel, breath, and the way it ages.",
  },
  {
    num: "III.",
    title: "Limited runs",
    body: "ผลิตจำนวนจำกัดในแต่ละรอบ — small batches, never mass-produced, made to be worn for years.",
  },
  {
    num: "IV.",
    title: "Here to help",
    body: "ทักแชทได้เสมอ ทั้งเรื่องไซส์และการเปลี่ยนสินค้า — LINE, Facebook, or Instagram.",
  },
];

export default function Values() {
  return (
    <section className={s.values}>
      <div className="wrap">
        <div className={s.vgrid}>
          {VALUES.map((v, i) => (
            <Reveal
              as="div"
              key={v.num}
              delay={([undefined, 2, 3][i % 3]) as 2 | 3 | undefined}
              className={s.vcell}
            >
              <div className={s.vnum}>{v.num}</div>
              <h3>{v.title}</h3>
              <p>{v.body}</p>
            </Reveal>
          ))}
        </div>
      </div>
    </section>
  );
}
