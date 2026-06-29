import type { Metadata } from "next";
import { Cormorant_Garamond, Inter, Playfair_Display } from "next/font/google";
import { CartProvider } from "@/lib/cart";
import TopBar from "@/components/TopBar";
import Footer from "@/components/Footer";
import BagDrawer from "@/components/BagDrawer";
import "./globals.css";

const cormorant = Cormorant_Garamond({
  subsets: ["latin"],
  weight: ["300", "400", "500", "600"],
  style: ["normal", "italic"],
  variable: "--font-cormorant",
});
const playfair = Playfair_Display({
  subsets: ["latin"],
  weight: ["400", "500", "600"],
  style: ["normal", "italic"],
  variable: "--font-playfair",
});
const inter = Inter({
  subsets: ["latin"],
  weight: ["300", "400", "500", "600"],
  variable: "--font-inter",
});

export const metadata: Metadata = {
  metadataBase: new URL("https://brunocollective.example"),
  title: {
    default: "Bruno Collective — Quietly Made in Thailand",
    template: "%s — Bruno Collective",
  },
  description:
    "Considered clothing, cut and finished by hand in Khon Kaen, Thailand — born from a love of fine cloth and quiet luxury. เสื้อผ้าคุณภาพ ตัดเย็บในไทย.",
  openGraph: {
    title: "Bruno Collective — Quietly Made in Thailand",
    description:
      "Considered clothing, cut and finished by hand in Khon Kaen, Thailand — born from a love of fine cloth and quiet luxury.",
    type: "website",
  },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html
      lang="en"
      className={`${cormorant.variable} ${playfair.variable} ${inter.variable}`}
    >
      <body>
        <CartProvider>
          <TopBar />
          {children}
          <Footer />
          <BagDrawer />
        </CartProvider>
      </body>
    </html>
  );
}
