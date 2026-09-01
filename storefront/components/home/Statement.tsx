import Link from "next/link";
import Reveal from "@/components/Reveal";
import s from "./home.module.css";

export default function Statement() {
  return (
    <section className={s.statement}>
      <Reveal className="wrap">
        <div className={s.stBig}>
          Made in <em>Khon Kaen.</em>
        </div>
        <p>
          Bruno Collective began with a simple obsession — fine cloth, clean
          lines, and clothes that feel as good as they look. Every piece is cut
          and finished by hand in our studio. ตั้งใจทำให้ดี ในทุกชิ้น.
        </p>
        <Link href="/story" className="qlink">
          Read Our Story <span className="arrow">→</span>
        </Link>
      </Reveal>
    </section>
  );
}
