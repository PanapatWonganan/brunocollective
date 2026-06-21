import Reveal from "@/components/Reveal";
import s from "./sections.module.css";

const ENTRIES = [
  {
    img: "https://images.unsplash.com/photo-1531572753322-ad063cecc140?auto=format&fit=crop&w=1200&q=80",
    tag: "Essay — N° 17", read: "8 min",
    title: <>Notes from the <em>Italian Riviera</em></>,
    body: "On the slow afternoons of Camogli, the quiet language of linen, and the particular blue of a sea that has been watched for a thousand years.",
  },
  {
    img: "https://images.pexels.com/photos/2474308/pexels-photo-2474308.jpeg?auto=compress&cs=tinysrgb&w=1200",
    tag: "Essay — N° 16", read: "12 min", d: 2 as const,
    title: <>The art of the <em>wardrobe</em></>,
    body: "A small library, fifteen garments, a single trunk — on building a wardrobe that travels with you, and outlives the seasons that bore it.",
  },
  {
    img: "https://images.unsplash.com/photo-1481627834876-b7833e8f5570?auto=format&fit=crop&w=1200&q=80",
    tag: "Essay — N° 15", read: "6 min", d: 3 as const,
    title: <>A library, a <em>linen suit</em></>,
    body: "Conversations with the master cutter Stefano Lorenzi, on inheritance, on patience, and on the books one keeps beside the cutting table.",
  },
];

export default function Journal() {
  return (
    <section className={`${s.journal} section`} id="journal">
      <div className="wrap">
        <Reveal className="sec-head">
          <div className="num">IV.</div>
          <div className="right">
            <span className="kicker">The Journal</span>
            <h2>
              Stories from <em>the house.</em>
            </h2>
          </div>
        </Reveal>

        <div className={s.journGrid}>
          {ENTRIES.map((e, i) => (
            <Reveal as="article" key={i} delay={e.d}>
              <div className={s.journImgbox}>
                <div className={s.journImg} style={{ backgroundImage: `url('${e.img}')` }} />
              </div>
              <div className={s.meta}>
                <span>{e.tag}</span>
                <span>{e.read}</span>
              </div>
              <h3>{e.title}</h3>
              <p>{e.body}</p>
              <a href="#" className="qlink qlink--ghost read">
                Read — <span className="arrow">→</span>
              </a>
            </Reveal>
          ))}
        </div>
      </div>
    </section>
  );
}
