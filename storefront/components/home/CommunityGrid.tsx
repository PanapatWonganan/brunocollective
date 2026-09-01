import Reveal from "@/components/Reveal";
import { imageSrc } from "@/lib/format";
import type { SiteImage } from "@/lib/api";
import s from "./home.module.css";

// "In the wild" — six square photos managed entirely from the admin Site
// Images page (community_1..6). caption_a is the hover label (e.g. an IG
// handle), caption_b an optional link. The whole section hides until at least
// one photo has been uploaded — no placeholder imagery.
export default function CommunityGrid({ site }: { site?: Record<string, SiteImage> }) {
  const cells = Array.from({ length: 6 }, (_, i) => site?.[`community_${i + 1}`])
    .filter((slot): slot is SiteImage => !!slot?.image_url);
  if (cells.length === 0) return null;

  return (
    <section className={s.comm}>
      <div className="wrap">
        <Reveal className={s.commHead}>
          <h2>
            In the <em>wild.</em>
          </h2>
          <a
            className="qlink"
            href="https://www.instagram.com/bruno.collective/"
            target="_blank"
            rel="noreferrer"
          >
            Follow the House <span className="arrow">→</span>
          </a>
        </Reveal>
        <div className={s.cgrid}>
          {cells.map((slot, i) => {
            const inner = (
              <>
                {slot.caption_a && <span className={s.h}>{slot.caption_a}</span>}
              </>
            );
            const style = { backgroundImage: `url('${imageSrc(slot.image_url)}')` };
            return (
              <Reveal
                as="div"
                key={slot.key || i}
                delay={([undefined, 2, 3][i % 3]) as 2 | 3 | undefined}
              >
                {slot.caption_b ? (
                  <a
                    className={s.ccell}
                    style={style}
                    href={slot.caption_b}
                    target="_blank"
                    rel="noreferrer"
                  >
                    {inner}
                  </a>
                ) : (
                  <div className={s.ccell} style={style}>
                    {inner}
                  </div>
                )}
              </Reveal>
            );
          })}
        </div>
      </div>
    </section>
  );
}
