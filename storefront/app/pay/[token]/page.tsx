import { notFound } from "next/navigation";
import type { Metadata } from "next";
import { getPayOrder } from "@/lib/api";
import PayClient from "./PayClient";

// Payment pages must always show live order state (slip received, status).
export const dynamic = "force-dynamic";

interface Props {
  params: Promise<{ token: string }>;
}

export async function generateMetadata(): Promise<Metadata> {
  return {
    title: "ชำระเงิน · Bruno Collective",
    robots: { index: false }, // personal payment links must never be indexed
  };
}

export default async function PayPage({ params }: Props) {
  const { token } = await params;
  const order = await getPayOrder(token);
  if (!order) notFound();
  return <PayClient token={token} initialOrder={order} />;
}
