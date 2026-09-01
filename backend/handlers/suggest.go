package handlers

import (
	"sort"
	"strconv"
	"strings"

	"brunocollective_inventory/database"
	"brunocollective_inventory/models"

	"github.com/gofiber/fiber/v2"
)

// Suggest returns cross-sell suggestions for a set of product ids — products
// most often bought in the same order (the same co-occurrence signal as the
// analytics "bought together" pairs), topped up with overall best sellers so
// a young shop with little pair data still gets suggestions. In-stock
// products only; the input ids are excluded.
//
// With no ids at all the co-occurrence step is skipped and the endpoint
// simply returns best sellers (topped up with the catalogue display order) —
// the storefront home page uses this for its "Best sellers" row.
//
// GET /api/shop/products/suggest?ids=1,2&limit=4 (public — used by the
// storefront checkout, the storefront home page and the admin chat-order
// dialog alike).
func (h *ShopHandler) Suggest(c *fiber.Ctx) error {
	given := make(map[uint]bool)
	ids := make([]uint, 0, 4)
	for _, s := range strings.Split(c.Query("ids"), ",") {
		if n, err := strconv.Atoi(strings.TrimSpace(s)); err == nil && n > 0 && !given[uint(n)] {
			given[uint(n)] = true
			ids = append(ids, uint(n))
		}
	}
	limit := 4
	if n, err := strconv.Atoi(c.Query("limit")); err == nil && n > 0 && n <= 12 {
		limit = n
	}
	// Orders (non-cancelled) that contain any of the given products.
	var orderIDs []uint
	if len(ids) > 0 {
		database.DB.Model(&models.OrderItem{}).
			Joins("JOIN orders ON orders.id = order_items.order_id").
			Where("orders.status != ? AND order_items.product_id IN ?", models.StatusCancelled, ids).
			Distinct("order_items.order_id").
			Pluck("order_items.order_id", &orderIDs)
	}

	// Co-purchased products ranked by how many of those orders they appear in.
	counts := make(map[uint]int)
	if len(orderIDs) > 0 {
		type coRow struct {
			ProductID uint
			N         int
		}
		var rows []coRow
		database.DB.Model(&models.OrderItem{}).
			Where("order_id IN ? AND product_id NOT IN ?", orderIDs, ids).
			Select("product_id, COUNT(DISTINCT order_id) as n").
			Group("product_id").
			Scan(&rows)
		for _, r := range rows {
			counts[r.ProductID] = r.N
		}
	}
	ranked := make([]uint, 0, len(counts))
	for id := range counts {
		ranked = append(ranked, id)
	}
	sort.Slice(ranked, func(a, b int) bool { return counts[ranked[a]] > counts[ranked[b]] })

	// Fill the tail with overall best sellers (by units) not already covered.
	var best []uint
	database.DB.Model(&models.OrderItem{}).
		Joins("JOIN orders ON orders.id = order_items.order_id").
		Where("orders.status != ?", models.StatusCancelled).
		Select("order_items.product_id").
		Group("order_items.product_id").
		Order("SUM(order_items.quantity) DESC").
		Limit(20).
		Pluck("order_items.product_id", &best)
	for _, id := range best {
		if !given[id] && counts[id] == 0 {
			ranked = append(ranked, id)
		}
	}

	// Young shop with little order history: top up from the catalogue in the
	// owner's storefront display order so the list is never short.
	var catalog []uint
	database.DB.Model(&models.Product{}).
		Order(ProductDisplayOrder).
		Limit(40).
		Pluck("id", &catalog)
	for _, id := range catalog {
		if !given[id] {
			ranked = append(ranked, id)
		}
	}

	out := make([]models.Product, 0, limit)
	seen := make(map[uint]bool)
	for _, id := range ranked {
		if seen[id] {
			continue
		}
		seen[id] = true
		var p models.Product
		if err := database.DB.Preload("Variants").First(&p, id).Error; err != nil {
			continue
		}
		p.ComputeTotalStock()
		if p.TotalStock <= 0 {
			continue
		}
		out = append(out, p)
		if len(out) >= limit {
			break
		}
	}
	return c.JSON(out)
}
