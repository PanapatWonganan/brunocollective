package models

// ProductVariant is one sellable size+color combination of a Product. Stock and
// SKU live here (per variant). Price normally stays on the Product (one price
// for all variants); a variant may carry its own Price (e.g. a ring whose gold
// colour costs more than silver) — 0 means "same as the product price".
// A garment with no variants falls back to the legacy Product.Size /
// Product.Stock fields, so old single-size products keep working unchanged.
type ProductVariant struct {
	ID        uint   `json:"id" gorm:"primaryKey"`
	ProductID uint   `json:"product_id" gorm:"not null;index"`
	Size      string `json:"size"`  // "" allowed (color-only or one-size garments)
	Color     string `json:"color"` // "" allowed (size-only garments)
	SKU       string `json:"sku"`   // per-variant; intentionally NOT globally unique
	Stock     int    `json:"stock" gorm:"default:0"`
	// Price overrides the product price for this variant; 0 = inherit.
	Price float64 `json:"price" gorm:"default:0"`
}

// UnitPrice is the selling price of this variant: its own Price when set,
// otherwise the product's base price.
func (v ProductVariant) UnitPrice(base float64) float64 {
	if v.Price > 0 {
		return v.Price
	}
	return base
}
