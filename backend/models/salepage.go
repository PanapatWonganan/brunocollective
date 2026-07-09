package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// SalePageSection is one content block on a sale page (hero, pain, story,
// offer stack, …). Data is a free-form bag whose fields are agreed between the
// admin builder (Vue) and the storefront renderer (Next) — the backend only
// stores it. Order in the slice = order on the page.
type SalePageSection struct {
	Type    string                 `json:"type"`
	Enabled bool                   `json:"enabled"`
	Data    map[string]interface{} `json:"data"`
}

// SalePageSections is stored as a JSON TEXT column (same AutoMigrate-friendly
// pattern as ReceiptLines and Product.Images).
type SalePageSections []SalePageSection

func (s SalePageSections) Value() (driver.Value, error) {
	if s == nil {
		return "[]", nil
	}
	b, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func (s *SalePageSections) Scan(value interface{}) error {
	if value == nil {
		*s = SalePageSections{}
		return nil
	}
	var data []byte
	switch v := value.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		return errors.New("unsupported type for SalePageSections")
	}
	if len(data) == 0 {
		*s = SalePageSections{}
		return nil
	}
	return json.Unmarshal(data, s)
}

type SalePageStatus string

const (
	SalePageDraft     SalePageStatus = "draft"
	SalePagePublished SalePageStatus = "published"
)

// SalePage is a ClickFunnels-style long-form landing page for one offer,
// served by the storefront at /s/{slug}. Pricing on it (offer price + bump
// price) overrides catalog prices and is always resolved server-side when the
// order is placed — the page JSON the storefront receives is display-only.
type SalePage struct {
	ID     uint           `json:"id" gorm:"primaryKey"`
	Slug   string         `json:"slug" gorm:"uniqueIndex;not null"`
	Title  string         `json:"title" gorm:"not null"`
	Status SalePageStatus `json:"status" gorm:"default:draft"`

	// Main offer.
	ProductID  uint     `json:"product_id" gorm:"not null"`
	Product    Product  `json:"product" gorm:"foreignKey:ProductID"`
	OfferPrice *float64 `json:"offer_price"` // nil = sell at catalog price

	Sections SalePageSections `json:"sections" gorm:"type:text"`

	// Order bump — an add-on offered as a checkbox on the order form.
	BumpEnabled     bool     `json:"bump_enabled"`
	BumpProductID   *uint    `json:"bump_product_id"`
	BumpProduct     *Product `json:"bump_product" gorm:"foreignKey:BumpProductID"`
	BumpPrice       float64  `json:"bump_price"`
	BumpHeadline    string   `json:"bump_headline"`
	BumpDescription string   `json:"bump_description"`

	// Urgency + checkout behaviour.
	CountdownEndsAt *time.Time `json:"countdown_ends_at"` // nil = no countdown
	ShowStock       bool       `json:"show_stock"`        // "only N left" from real stock
	AllowCoupon     bool       `json:"allow_coupon"`

	// Funnel stats. BumpCount counts orders that took the bump.
	Views       int `json:"views"`
	OrdersCount int `json:"orders_count"`
	BumpCount   int `json:"bump_count"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
