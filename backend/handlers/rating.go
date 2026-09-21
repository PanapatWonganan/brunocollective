package handlers

import (
	"math"

	"brunocollective_inventory/database"
	"brunocollective_inventory/models"
)

// Storefront star ratings.
//
// The shop has no customer reviews, but a product card without stars reads as
// "nobody buys this" — so the storefront shows a rating derived from real
// sales volume instead. The rating is deterministic (same sales → same stars)
// and always in the 4.5–5.0 band the owner asked for: a product with a single
// sale shows 4.5 and climbs towards 5.0 as units sold grow. The count shown
// next to the stars is the real number of units sold on non-cancelled orders.
// Products with no sales get 0/0 and the storefront hides the stars.

// ratingHalfwayUnits is the sales volume at which a product sits halfway up
// the band (4.75). Tuned for a small label: ~8 units → 4.8, ~30 → 4.9, ~90 → 5.0.
const ratingHalfwayUnits = 8.0

// ratingFor returns the star rating to show: the admin override when set
// (clamped to 0–5, one decimal), otherwise a 4.5–5.0 value mapped from units
// sold. A tiny id-based offset keeps products with identical sales from all
// showing the exact same number, which would look generated.
func ratingFor(productID uint, override *float64, units int) float64 {
	if override != nil && *override > 0 {
		r := math.Round(*override*10) / 10
		if r > 5 {
			r = 5
		}
		return r
	}
	if units <= 0 {
		return 0
	}
	u := float64(units)
	r := 4.5 + 0.5*(u/(u+ratingHalfwayUnits))
	r += float64(productID%5) * 0.01 // 0.00–0.04 jitter, stable per product
	r = math.Round(r*10) / 10
	if r < 4.5 {
		r = 4.5
	}
	if r > 5 {
		r = 5
	}
	return r
}

// loadUnitsSold returns units sold per product across non-cancelled orders.
// Merged products already have their order items re-pointed at the survivor,
// so the count follows the product the shopper actually sees.
func loadUnitsSold() map[uint]int {
	type row struct {
		ProductID uint
		Units     int
	}
	var rows []row
	database.DB.Model(&models.OrderItem{}).
		Joins("JOIN orders ON orders.id = order_items.order_id").
		Where("orders.status != ?", models.StatusCancelled).
		Select("order_items.product_id AS product_id, SUM(order_items.quantity) AS units").
		Group("order_items.product_id").
		Scan(&rows)
	out := make(map[uint]int, len(rows))
	for _, r := range rows {
		out[r.ProductID] = r.Units
	}
	return out
}

// applyRatings fills Rating/RatingCount on every product in the slice with a
// single aggregate query.
func applyRatings(products []models.Product) {
	if len(products) == 0 {
		return
	}
	sold := loadUnitsSold()
	for i := range products {
		units := sold[products[i].ID]
		products[i].RatingCount = units
		products[i].Rating = ratingFor(products[i].ID, products[i].RatingOverride, units)
	}
}

// applyRating fills Rating/RatingCount on a single product.
func applyRating(p *models.Product) {
	var units int
	database.DB.Model(&models.OrderItem{}).
		Joins("JOIN orders ON orders.id = order_items.order_id").
		Where("orders.status != ? AND order_items.product_id = ?", models.StatusCancelled, p.ID).
		Select("COALESCE(SUM(order_items.quantity), 0)").
		Scan(&units)
	p.RatingCount = units
	p.Rating = ratingFor(p.ID, p.RatingOverride, units)
}
