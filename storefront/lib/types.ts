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
  items: { product_id: number; variant_id: number | null; quantity: number }[];
}
