export interface ProductVariant {
  id: number;
  product_id: number;
  size: string;
  color: string;
  sku: string;
  stock: number;
}

export interface Product {
  id: number;
  name: string;
  sku: string;
  size: string;
  description: string;
  price: number;
  stock: number;
  image_url: string;
  images: string[];
  variants: ProductVariant[] | null;
  total_stock: number;
  created_at: string;
  updated_at: string;
}

export interface CartLine {
  product: Product;
  variant: ProductVariant | null;
  quantity: number;
}

export interface CheckoutPayload {
  name: string;
  phone: string;
  email?: string;
  address: string;
  notes?: string;
  coupon_code?: string;
  items: { product_id: number; variant_id: number | null; quantity: number }[];
}

// One content block on a sale/landing page. `data` fields are agreed between
// the admin builder and the renderer in app/s/[slug].
export interface SalePageSection {
  type: string;
  enabled: boolean;
  data: Record<string, any>;
}

// Funnel-style landing page served at /s/{slug}. Prices here are display-only —
// the backend recomputes everything when the order is placed.
export interface SalePage {
  id: number;
  slug: string;
  title: string;
  status: "draft" | "published";
  product_id: number;
  product: Product;
  offer_price: number | null;
  sections: SalePageSection[];
  bump_enabled: boolean;
  bump_product_id: number | null;
  bump_product: Product | null;
  bump_price: number;
  bump_headline: string;
  bump_description: string;
  countdown_ends_at: string | null;
  show_stock: boolean;
  allow_coupon: boolean;
}

// Storefront member profile (from /api/shop/members/*). Members get a flat
// discount_percent off every order, separate from coupon discounts.
export interface MemberProfile {
  id: number;
  name: string;
  phone: string;
  email: string;
  address: string;
  is_member: boolean;
  member_since: string | null;
  discount_percent: number;
}

// One past order in the member's history (subset of the backend Order).
export interface MemberOrder {
  id: number;
  status: string;
  subtotal: number;
  member_discount: number;
  discount_amount: number;
  coupon_code: string;
  total_amount: number;
  created_at: string;
  items: {
    id: number;
    quantity: number;
    price: number;
    size: string;
    color: string;
    product: { id: number; name: string; image_url: string };
  }[];
}

// Successful response from POST /api/shop/coupons/validate.
export interface CouponPreview {
  valid: boolean;
  code: string;
  name: string;
  type: "percent" | "fixed";
  value: number;
  discount: number;
  total: number;
}
