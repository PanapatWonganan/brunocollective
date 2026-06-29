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
  created_at: string;
  updated_at: string;
}

export interface CartLine {
  product: Product;
  quantity: number;
}

export interface CheckoutPayload {
  name: string;
  phone: string;
  email?: string;
  address: string;
  notes?: string;
  items: { product_id: number; quantity: number }[];
}
