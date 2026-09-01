import type { ReactNode } from "react";

// Renders "*words*" as an italic champagne <em> segment (same convention as
// the hero carousel headlines and the design's <em> spans).
export function em(text: string): ReactNode[] {
  return text
    .split(/\*([^*]+)\*/g)
    .map((part, i) => (i % 2 === 1 ? <em key={i}>{part}</em> : <span key={i}>{part}</span>));
}

export interface Essay {
  slug: string;
  tag: string;
  read: string;
  title: string; // *...* = italic segment
  summary: string;
  cover: string; // default — overridden by the journal_{n} site-image slot
  body: string[]; // paragraphs; *...* = italic segment
}

// The three house essays. Order matters: index i pairs with the admin
// site-image slot journal_{i+1} for the cover photo, tag and read time.
export const ESSAYS: Essay[] = [
  {
    slug: "made-in-thailand",
    tag: "Essay — N° 03",
    read: "8 min",
    title: "Why we make it *in Thailand*",
    summary:
      "On choosing to design and finish every piece in Thailand — what local making lets us control, and why slower, smaller runs make better clothes.",
    cover:
      "https://images.unsplash.com/photo-1531572753322-ad063cecc140?auto=format&fit=crop&w=1600&q=80",
    body: [
      "It would be easier not to. Easier to send a tech pack overseas, wait for a container, and sell what arrives. Most brands our size do exactly that, and it is a perfectly sensible way to run a business. It is just not the way to make the clothes we wanted to make.",
      "Bruno Collective is designed and finished in Thailand because closeness is a kind of quality control that no inspection report can replace. เมื่อโต๊ะตัดผ้าอยู่ห่างจากโต๊ะออกแบบไม่กี่ก้าว เราเห็นทุกตะเข็บก่อนถึงมือลูกค้า — when the cutting table is a few steps from the design table, a seam that sits wrong gets fixed the same afternoon, not discovered three months later in a warehouse.",
      "Making locally also changes what we are willing to make. A factory minimum forces you to guess big and discount later; a small run lets you make only what deserves to exist. เราจึงผลิตทีละรอบ จำนวนจำกัด — we cut in small batches, sell through, listen, and cut again a little better. When a run is gone, it is gone; that is not a marketing line, it is simply how small-batch making works.",
      "There is also the quieter reason. The people who press the seams and sew the buttons here are not a line item on a freight invoice — they are the reason the clothes feel the way they do. Keeping the work in Thailand keeps that care close, and keeps us honest about what every piece actually costs to make well.",
      "None of this makes a t-shirt more remarkable to look at, and that is rather the point. *Quiet luxury is care you can feel, not a label you can read.* The making is the luxury; Thailand is where we make it.",
    ],
  },
  {
    slug: "art-of-the-wardrobe",
    tag: "Essay — N° 02",
    read: "12 min",
    title: "The art of the *wardrobe*",
    summary:
      "A few good garments, worn often — on building a quiet wardrobe that travels with you and outlives the season that bore it.",
    body: [
      "Every wardrobe tells one of two stories. The first is a diary of impulses — a rail of single evenings, each piece bought for one occasion and retired soon after. The second is a memoir: fewer garments, worn often, each one earning its place. The first costs more and says less. This essay is about the second.",
      "Start with the uniform, not the exception. Most of us wear the same silhouette eighty days out of a hundred — เสื้อตัวโปรดตัวเดิม กางเกงตัวเดิม — so put the money where the wear is. A t-shirt with real fabric and a collar that holds, trousers that sit right, one shirt you reach for without thinking. The exceptional pieces can wait; the daily ones cannot.",
      "Buy the fabric, not the print. Cloth is the part of a garment you actually live with — it is what you feel at your collar at three in the afternoon. ผ้าที่ดีจะนุ่มขึ้นตามเวลา ไม่ใช่เสื่อมลง — good fabric softens with age instead of wearing out. If you must choose between a beautiful cut in a poor cloth and a plain cut in a beautiful cloth, choose the cloth every time.",
      "Let things repeat. The quiet trick of a good wardrobe is that everything goes with everything, because nothing is shouting. Three colours you love beat twelve you tolerate. A small wardrobe that repeats well travels in one bag and never leaves you staring at a full rail with nothing to wear.",
      "And then, wear things out. A garment worn a hundred times is a better purchase than one worn twice, whatever the price tags said. เสื้อผ้าที่ดีควรถูกใส่ ไม่ใช่ถูกเก็บ — clothes are for wearing, not for keeping. The art of the wardrobe is not acquiring; it is returning, season after season, to the same good things.",
    ],
    cover:
      "https://images.pexels.com/photos/2474308/pexels-photo-2474308.jpeg?auto=compress&cs=tinysrgb&w=1600",
  },
  {
    slug: "quiet-luxury",
    tag: "Essay — N° 01",
    read: "6 min",
    title: "What “quiet luxury” *really means*",
    summary:
      "Less logo, more cloth. On the small details — the hand of a fabric, a clean seam, a hem that hangs right — that separate considered clothing from the rest.",
    cover:
      "https://images.unsplash.com/photo-1481627834876-b7833e8f5570?auto=format&fit=crop&w=1600&q=80",
    body: [
      "The phrase has been worn thin by use, so let us say plainly what we mean by it. Quiet luxury is not a beige colour palette, and it is certainly not a higher price with a smaller logo. It is a simple reordering of priorities: the money goes into the garment, not onto it.",
      "You can audit it with your hands. Feel the weight of the fabric; a good cloth has body without stiffness. ลองจับชายเสื้อ ดูตะเข็บ — turn the hem and look at the stitching, because the inside of a garment is where the truth lives. Check the collar after the tenth wash, not the first wear. None of this requires expertise; it only requires attention.",
      "Loud clothes make a promise to other people; quiet clothes make a promise to you. เสื้อที่ดีไม่ต้องบอกใครว่าแพง — the person across the table may never know what your shirt cost, and that is precisely the point. What you get instead is private: the way it feels at hour nine of the day, the way it hangs after a year of wear.",
      "This is why quiet luxury is really a making problem, not a styling problem. A clean seam costs sewing time. A collar that keeps its shape costs better interfacing and a second pressing. งานเงียบ ๆ พวกนี้แหละคือของแพงจริง — the quiet work is the expensive part, which is why so many brands skip it and print a logo instead.",
      "So our definition, for what it is worth: *luxury is care you can feel, kept quiet.* Everything we make in Thailand is an attempt at that sentence.",
    ],
  },
];

export function essayBySlug(slug: string): Essay | undefined {
  return ESSAYS.find((e) => e.slug === slug);
}
