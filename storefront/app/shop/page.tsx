import type { Metadata } from "next";
import { getProducts } from "@/lib/api";
import type { Product } from "@/lib/types";
import ShopClient from "./ShopClient";

export const metadata: Metadata = {
  title: "The Collection",
  description:
    "Shop the full Bruno Collective collection — limited runs, cut and finished by hand in Khon Kaen, Thailand.",
};

interface Props {
  searchParams: Promise<{ cat?: string; sort?: string }>;
}

export default async function ShopPage({ searchParams }: Props) {
  const { cat, sort } = await searchParams;
  let products: Product[] = [];
  try {
    products = await getProducts({ includeOut: true });
  } catch {
    products = [];
  }

  return (
    <ShopClient
      products={products}
      initialCat={cat || ""}
      initialSort={sort === "new" ? "new" : "default"}
    />
  );
}
