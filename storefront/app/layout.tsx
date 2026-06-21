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
    default: "Bruno Collective — A Quiet Inheritance",
    template: "%s — Bruno Collective",
  },
  description:
    "Garments cut for the long quiet of a life lived deliberately — linen, cashmere, and time. A heritage house since 1908.",
  openGraph: {
    title: "Bruno Collective — A Quiet Inheritance",
    description:
      "Garments cut for the long quiet of a life lived deliberately — linen, cashmere, and time.",
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
