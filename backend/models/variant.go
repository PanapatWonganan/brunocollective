package models

// ProductVariant is one sellable size+color combination of a Product. Stock and
// SKU live here (per variant); price stays on the Product (one price for all
// variants). A garment with no variants falls back to the legacy Product.Size /
// Product.Stock fields, so old single-size products keep working unchanged.
type ProductVariant struct {
	ID        uint   `json:"id" gorm:"primaryKey"`
	ProductID uint   `json:"product_id" gorm:"not null;index"`
	Size      string `json:"size"`  // "" allowed (color-only or one-size garments)
	Color     string `json:"color"` // "" allowed (size-only garments)
	SKU       string `json:"sku"`   // per-variant; intentionally NOT globally unique
	Stock     int    `json:"stock" gorm:"default:0"`
}
