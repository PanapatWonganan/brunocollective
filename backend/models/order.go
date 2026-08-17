package models

import "time"

type OrderStatus string

const (
	StatusPending   OrderStatus = "pending"
	StatusConfirmed OrderStatus = "confirmed"
	StatusShipped   OrderStatus = "shipped"
	StatusDelivered OrderStatus = "delivered"
	StatusCancelled OrderStatus = "cancelled"
)

type Order struct {
	ID         uint        `json:"id" gorm:"primaryKey"`
	CustomerID uint        `json:"customer_id" gorm:"not null"`
	Customer   Customer    `json:"customer" gorm:"foreignKey:CustomerID"`
	Status     OrderStatus `json:"status" gorm:"default:pending"`
	// Subtotal is the item total before discount; TotalAmount is the amount the
	// customer pays (Subtotal - DiscountAmount). Legacy pre-coupon orders have
	// Subtotal 0 — readers should fall back to TotalAmount when Subtotal is 0.
	Subtotal       float64 `json:"subtotal"`
	DiscountAmount float64 `json:"discount_amount"`
	// MemberDiscount is the membership discount (5% of Subtotal), kept separate
	// from the coupon DiscountAmount — the two stack:
	// TotalAmount = Subtotal - MemberDiscount - DiscountAmount.
	MemberDiscount float64 `json:"member_discount"`
	// CouponCode is snapshotted so the order keeps showing the code even if the
	// coupon is later edited or deleted.
	CouponID   *uint  `json:"coupon_id"`
	CouponCode string `json:"coupon_code"`
	// SalePageID marks orders placed through a sale/landing page (/s/{slug})
	// so per-campaign conversion can be tracked. Nil for normal orders.
	SalePageID *uint `json:"sale_page_id"`
	// Channel is where the sale came from (facebook, line, instagram, tiktok,
	// walk-in, storefront, sale-page, …). Admin-created orders send it from the
	// order form; storefront/sale-page checkouts stamp it server-side. Empty on
	// legacy orders = unknown.
	Channel string `json:"channel"`
	// ConversationID links orders created from the chat inbox back to the
	// conversation they were sold in (attribution). Nil for other orders.
	ConversationID *uint `json:"conversation_id"`
	// PaymentToken is an unguessable token for the public payment page
	// (/pay/{token}) where the customer sees the order summary and uploads a
	// slip without logging in. Empty = no public payment page for this order.
	PaymentToken string      `json:"payment_token" gorm:"index"`
	TotalAmount  float64     `json:"total_amount"`
	SlipImage   string      `json:"slip_image"`
	Notes       string      `json:"notes"`
	Items       []OrderItem `json:"items" gorm:"foreignKey:OrderID"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

type OrderItem struct {
	ID        uint    `json:"id" gorm:"primaryKey"`
	OrderID   uint    `json:"order_id" gorm:"not null"`
	ProductID uint    `json:"product_id" gorm:"not null"`
	Product   Product `json:"product" gorm:"foreignKey:ProductID"`
	// VariantID is the chosen size+color variant (nullable — legacy items and
	// variant-less products leave it null). Size/Color are snapshotted at order
	// time so the line stays stable even if the variant later changes/deletes.
	VariantID *uint   `json:"variant_id"`
	Size      string  `json:"size"`
	Color     string  `json:"color"`
	Quantity  int     `json:"quantity" gorm:"not null"`
	Price     float64 `json:"price" gorm:"not null"`
}

type CreateOrderRequest struct {
	CustomerID uint              `json:"customer_id"`
	Notes      string            `json:"notes"`
	CouponCode string            `json:"coupon_code"`
	Channel    string            `json:"channel"`
	Items      []CreateOrderItem `json:"items"`
}

type CreateOrderItem struct {
	ProductID uint  `json:"product_id"`
	VariantID *uint `json:"variant_id"`
	Quantity  int   `json:"quantity"`
}
