package models

import "time"

type CouponType string

const (
	CouponPercent CouponType = "percent" // Value = percent off (0-100), optionally capped by MaxDiscount
	CouponFixed   CouponType = "fixed"   // Value = THB off, clamped to the order subtotal
)

// Coupon is a discount code. Phase-1 scope: percent/fixed discount, minimum
// order amount, start/expiry window, total + per-customer usage limits, and a
// manual on/off switch. Codes are stored uppercase and matched case-insensitively.
type Coupon struct {
	ID          uint       `json:"id" gorm:"primaryKey"`
	Code        string     `json:"code" gorm:"uniqueIndex;not null"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Type        CouponType `json:"type" gorm:"not null"`
	Value       float64    `json:"value" gorm:"not null"`
	MaxDiscount float64    `json:"max_discount"` // percent type only; 0 = no cap

	MinOrderAmount float64 `json:"min_order_amount"` // 0 = no minimum

	StartsAt  *time.Time `json:"starts_at"`  // nil = starts immediately
	ExpiresAt *time.Time `json:"expires_at"` // nil = never expires

	UsageLimit            int `json:"usage_limit"`              // total redemptions; 0 = unlimited
	UsageLimitPerCustomer int `json:"usage_limit_per_customer"` // per customer; 0 = unlimited
	UsedCount             int `json:"used_count"`

	IsActive  bool      `json:"is_active" gorm:"default:true"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CouponRedemption records one use of a coupon on an order. It both enforces
// the per-customer limit and feeds usage history/reporting. Rows are removed
// (and Coupon.UsedCount decremented) when the order is deleted.
type CouponRedemption struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	CouponID       uint      `json:"coupon_id" gorm:"index;not null"`
	OrderID        uint      `json:"order_id" gorm:"index;not null"`
	CustomerID     uint      `json:"customer_id" gorm:"index;not null"`
	DiscountAmount float64   `json:"discount_amount"`
	CreatedAt      time.Time `json:"created_at"`
}
