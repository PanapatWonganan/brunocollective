package models

import "time"

// Affiliate is a referrer (นายหน้า/KOL) created by the admin. Shoppers are
// attributed via a share link (?ref=CODE) or by typing the code at checkout.
// Phone + PasswordHash power the affiliate's own storefront portal login
// (mirrors the member login).
type Affiliate struct {
	ID    uint   `json:"id" gorm:"primaryKey"`
	Code  string `json:"code" gorm:"uniqueIndex;not null"` // stored uppercase, like Coupon.Code
	Name  string `json:"name" gorm:"not null"`
	Phone string `json:"phone" gorm:"uniqueIndex"` // portal login key
	Email string `json:"email"`
	// PasswordHash for the affiliate portal. Set by the admin on create/reset.
	PasswordHash string `json:"-"`
	// CommissionPercent is the affiliate's default rate. A product-level
	// override (Product.CommissionPercent) wins when set.
	CommissionPercent float64 `json:"commission_percent"`
	IsActive          bool    `json:"is_active" gorm:"default:true"`
	Notes             string  `json:"notes"`
	// ClickCount counts ?ref link hits (storefront fire-and-forget tracking).
	ClickCount int       `json:"click_count"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type CommissionStatus string

const (
	CommissionPending   CommissionStatus = "pending"   // order not yet delivered
	CommissionConfirmed CommissionStatus = "confirmed" // order delivered — payable
	CommissionPaid      CommissionStatus = "paid"      // admin marked จ่ายแล้ว (final)
	CommissionCancelled CommissionStatus = "cancelled" // order cancelled
)

// AffiliateCommission is the ledger: one row per order item that earns
// commission. RatePercent is snapshotted at order creation; BaseAmount and
// Amount are recomputed from the order's current numbers whenever totals
// change and again when the order is delivered (confirmation) — so pending
// amounts are estimates that self-heal. Paid rows are never touched.
type AffiliateCommission struct {
	ID          uint    `json:"id" gorm:"primaryKey"`
	AffiliateID uint    `json:"affiliate_id" gorm:"index;not null"`
	OrderID     uint    `json:"order_id" gorm:"index;not null"`
	OrderItemID uint    `json:"order_item_id" gorm:"not null"`
	ProductID   uint    `json:"product_id"`
	CustomerID  uint    `json:"customer_id"`
	RatePercent float64 `json:"rate_percent"`
	// BaseAmount is the item's net revenue: qty * price * (total/subtotal) —
	// order-level discounts (coupon + member) pro-rated onto the line.
	BaseAmount  float64          `json:"base_amount"`
	Amount      float64          `json:"amount"` // roundSatang(BaseAmount * RatePercent / 100)
	Status      CommissionStatus `json:"status" gorm:"index;default:pending"`
	ConfirmedAt *time.Time       `json:"confirmed_at"`
	PaidAt      *time.Time       `json:"paid_at"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}
