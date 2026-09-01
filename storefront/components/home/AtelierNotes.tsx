import Reveal from "@/components/Reveal";
import s from "./home.module.css";

// Quote-card section in the design's testimonial layout — filled with honest
// notes about the making (not invented customer reviews).
const NOTES = [
  {
    q: "“A t-shirt should not be remarkable. Ours quietly is — soft cotton, a collar that keeps its shape.”",
    who: "The Fabric",
    role: "ผ้านุ่ม ใส่สบาย ไม่เสียทรง",
  },
  {
    q: "“Seams pressed open, buttons sewn on by hand, hems checked twice before a piece leaves the studio.”",
    who: "The Finishing",
    role: "เก็บงานละเอียดทุกตะเข็บ",
  },
  {
    q: "“Cut in small runs, not mass-produced — when a run is gone, it is gone.”",
    who: "The Runs",
    role: "ผลิตจำนวนจำกัด",
  },
  {
    q: "“Wrong size? Message us and we will make it right — as many times as it takes.”",
    who: "The Service",
    role: "เปลี่ยนไซส์ได้ คุยกันได้เสมอ",
  },
];

export default function AtelierNotes() {
  return (
    <section className={s.notes}>
      <div className="wrap">
        <Reveal className={s.rowHead}>
          <h2>
            The quiet <em>details.</em>
          </h2>
        </Reveal>
        <div className={s.tgrid}>
          {NOTES.map((n, i) => (
            <Reveal
              as="div"
              key={n.who}
              delay={([undefined, 2, 3][i % 3]) as 2 | 3 | undefined}
              className={s.tcard}
            >
              <div className={s.tq}>{n.q}</div>
              <div>
                <div className={s.twho}>{n.who}</div>
                <div className={s.trole}>{n.role}</div>
              </div>
            </Reveal>
          ))}
        </div>
      </div>
    </section>
  );
}
