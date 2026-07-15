package handlers

import (
	"fmt"
	"time"

	"brunocollective_inventory/database"
	"brunocollective_inventory/models"

	"github.com/gofiber/fiber/v2"
)

// notificationItem is one row in the admin bell dropdown. Severity drives the
// icon/color on the frontend: info (pending order), warning (low stock),
// error (out of stock). Link is the admin route the item navigates to.
type notificationItem struct {
	Type      string     `json:"type"`
	Severity  string     `json:"severity"`
	Title     string     `json:"title"`
	Detail    string     `json:"detail"`
	Link      string     `json:"link"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
}

// Notifications aggregates actionable alerts for the bell menu: pending
// orders (newest 10), out-of-stock products, and low-stock products (≤5).
// Stock respects variants via ComputeTotalStock.
func (h *DashboardHandler) Notifications(c *fiber.Ctx) error {
	items := make([]notificationItem, 0, 16)

	var pending []models.Order
	database.DB.Preload("Customer").
		Where("status = ?", models.StatusPending).
		Order("created_at DESC").
		Limit(10).
		Find(&pending)
	var pendingTotal int64
	database.DB.Model(&models.Order{}).Where("status = ?", models.StatusPending).Count(&pendingTotal)

	for i := range pending {
		o := pending[i]
		created := o.CreatedAt
		items = append(items, notificationItem{
			Type:      "order",
			Severity:  "info",
			Title:     fmt.Sprintf("ออเดอร์ #%d รอยืนยัน", o.ID),
			Detail:    fmt.Sprintf("%s · ฿%.2f", o.Customer.Name, o.TotalAmount),
			Link:      "/orders",
			CreatedAt: &created,
		})
	}
	if pendingTotal > int64(len(pending)) {
		items = append(items, notificationItem{
			Type:     "order",
			Severity: "info",
			Title:    fmt.Sprintf("และอีก %d ออเดอร์รอยืนยัน", pendingTotal-int64(len(pending))),
			Link:     "/orders",
		})
	}

	var products []models.Product
	database.DB.Preload("Variants").Find(&products)
	for i := range products {
		p := &products[i]
		p.ComputeTotalStock()
		switch {
		case p.TotalStock == 0:
			items = append(items, notificationItem{
				Type:     "stock",
				Severity: "error",
				Title:    fmt.Sprintf("สินค้าหมดสต็อก: %s", p.Name),
				Link:     "/products",
			})
		case p.TotalStock <= 5:
			items = append(items, notificationItem{
				Type:     "stock",
				Severity: "warning",
				Title:    fmt.Sprintf("สินค้าใกล้หมด: %s", p.Name),
				Detail:   fmt.Sprintf("เหลือ %d ชิ้น", p.TotalStock),
				Link:     "/products",
			})
		}
	}

	return c.JSON(fiber.Map{
		"count": len(items),
		"items": items,
	})
}
