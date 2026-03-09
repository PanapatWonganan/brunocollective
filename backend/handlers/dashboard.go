package handlers

import (
	"brunocollective_inventory/database"
	"brunocollective_inventory/models"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
)

type DashboardHandler struct{}

func NewDashboardHandler() *DashboardHandler {
	return &DashboardHandler{}
}

func (h *DashboardHandler) Stats(c *fiber.Ctx) error {
	var productCount int64
	var customerCount int64
	var orderCount int64
	var totalRevenue float64
	var lowStockCount int64
	var pendingOrderCount int64

	database.DB.Model(&models.Product{}).Count(&productCount)
	database.DB.Model(&models.Customer{}).Count(&customerCount)
	database.DB.Model(&models.Order{}).Count(&orderCount)
	database.DB.Model(&models.Order{}).
		Where("status != ?", models.StatusCancelled).
		Select("COALESCE(SUM(total_amount), 0)").
		Scan(&totalRevenue)
	database.DB.Model(&models.Product{}).Where("stock <= 5").Count(&lowStockCount)
	database.DB.Model(&models.Order{}).Where("status = ?", models.StatusPending).Count(&pendingOrderCount)

	var recentOrders []models.Order
	database.DB.Preload("Customer").
		Order("created_at DESC").
		Limit(5).
		Find(&recentOrders)

	var lowStockProducts []models.Product
	database.DB.Where("stock <= 5").Order("stock ASC").Limit(5).Find(&lowStockProducts)

	return c.JSON(fiber.Map{
		"product_count":       productCount,
		"customer_count":      customerCount,
		"order_count":         orderCount,
		"total_revenue":       totalRevenue,
		"low_stock_count":     lowStockCount,
		"pending_order_count": pendingOrderCount,
		"recent_orders":       recentOrders,
		"low_stock_products":  lowStockProducts,
	})
}

// Charts returns data for dashboard charts
func (h *DashboardHandler) Charts(c *fiber.Ctx) error {
	period := c.Query("period", "month") // day, week, month, year

	// --- 1. Revenue over time ---
	revenueSeries := h.getRevenueSeries(period)

	// --- 2. Order status distribution ---
	statusDist := h.getOrderStatusDistribution()

	// --- 3. Stock overview (all products sorted by stock ASC) ---
	stockOverview := h.getStockOverview()

	// --- 4. Top selling products ---
	topSelling := h.getTopSellingProducts()

	return c.JSON(fiber.Map{
		"revenue_series":       revenueSeries,
		"order_status":         statusDist,
		"stock_overview":       stockOverview,
		"top_selling_products": topSelling,
	})
}

type RevenuePoint struct {
	Label   string  `json:"label"`
	Revenue float64 `json:"revenue"`
	Orders  int     `json:"orders"`
}

func (h *DashboardHandler) getRevenueSeries(period string) []RevenuePoint {
	var results []RevenuePoint
	now := time.Now()

	var dateFormat string
	var startDate time.Time
	var labelFormat string

	switch period {
	case "day":
		// Last 24 hours, grouped by hour
		startDate = now.Add(-24 * time.Hour)
		dateFormat = "%Y-%m-%d %H:00"
		labelFormat = "15:04"
	case "week":
		// Last 7 days, grouped by day
		startDate = now.AddDate(0, 0, -7)
		dateFormat = "%Y-%m-%d"
		labelFormat = "Mon"
	case "month":
		// Last 30 days, grouped by day
		startDate = now.AddDate(0, 0, -30)
		dateFormat = "%Y-%m-%d"
		labelFormat = "02 Jan"
	case "year":
		// Last 12 months, grouped by month
		startDate = now.AddDate(-1, 0, 0)
		dateFormat = "%Y-%m"
		labelFormat = "Jan 2006"
	default:
		startDate = now.AddDate(0, 0, -30)
		dateFormat = "%Y-%m-%d"
		labelFormat = "02 Jan"
	}

	type rawRow struct {
		Period  string
		Revenue float64
		Orders  int
	}
	var rows []rawRow

	database.DB.Model(&models.Order{}).
		Where("status != ? AND created_at >= ?", models.StatusCancelled, startDate).
		Select(fmt.Sprintf("strftime('%s', created_at) as period, COALESCE(SUM(total_amount), 0) as revenue, COUNT(*) as orders", dateFormat)).
		Group("period").
		Order("period ASC").
		Scan(&rows)

	// Generate all time slots to fill gaps
	slotMap := make(map[string]rawRow)
	for _, r := range rows {
		slotMap[r.Period] = r
	}

	switch period {
	case "day":
		for i := 24; i >= 0; i-- {
			t := now.Add(-time.Duration(i) * time.Hour)
			key := t.Format("2006-01-02 15:00")
			label := t.Format(labelFormat)
			if row, ok := slotMap[key]; ok {
				results = append(results, RevenuePoint{Label: label, Revenue: row.Revenue, Orders: row.Orders})
			} else {
				results = append(results, RevenuePoint{Label: label, Revenue: 0, Orders: 0})
			}
		}
	case "week":
		for i := 7; i >= 0; i-- {
			t := now.AddDate(0, 0, -i)
			key := t.Format("2006-01-02")
			label := t.Format(labelFormat)
			if row, ok := slotMap[key]; ok {
				results = append(results, RevenuePoint{Label: label, Revenue: row.Revenue, Orders: row.Orders})
			} else {
				results = append(results, RevenuePoint{Label: label, Revenue: 0, Orders: 0})
			}
		}
	case "month":
		for i := 30; i >= 0; i-- {
			t := now.AddDate(0, 0, -i)
			key := t.Format("2006-01-02")
			label := t.Format(labelFormat)
			if row, ok := slotMap[key]; ok {
				results = append(results, RevenuePoint{Label: label, Revenue: row.Revenue, Orders: row.Orders})
			} else {
				results = append(results, RevenuePoint{Label: label, Revenue: 0, Orders: 0})
			}
		}
	case "year":
		for i := 12; i >= 0; i-- {
			t := now.AddDate(0, -i, 0)
			key := t.Format("2006-01")
			label := t.Format(labelFormat)
			if row, ok := slotMap[key]; ok {
				results = append(results, RevenuePoint{Label: label, Revenue: row.Revenue, Orders: row.Orders})
			} else {
				results = append(results, RevenuePoint{Label: label, Revenue: 0, Orders: 0})
			}
		}
	}

	return results
}

type StatusCount struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

func (h *DashboardHandler) getOrderStatusDistribution() []StatusCount {
	var results []StatusCount
	database.DB.Model(&models.Order{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Scan(&results)

	// Ensure all statuses are present
	statusMap := make(map[string]int)
	for _, r := range results {
		statusMap[r.Status] = r.Count
	}

	allStatuses := []string{"pending", "confirmed", "shipped", "delivered", "cancelled"}
	var full []StatusCount
	for _, s := range allStatuses {
		full = append(full, StatusCount{Status: s, Count: statusMap[s]})
	}
	return full
}

type StockItem struct {
	Name  string `json:"name"`
	Stock int    `json:"stock"`
	SKU   string `json:"sku"`
}

func (h *DashboardHandler) getStockOverview() []StockItem {
	var items []StockItem
	database.DB.Model(&models.Product{}).
		Select("name, stock, sku").
		Order("stock ASC").
		Limit(15).
		Scan(&items)
	return items
}

type TopProduct struct {
	Name     string  `json:"name"`
	Quantity int     `json:"quantity"`
	Revenue  float64 `json:"revenue"`
}

func (h *DashboardHandler) getTopSellingProducts() []TopProduct {
	var items []TopProduct
	database.DB.Model(&models.OrderItem{}).
		Joins("JOIN products ON products.id = order_items.product_id").
		Joins("JOIN orders ON orders.id = order_items.order_id").
		Where("orders.status != ?", models.StatusCancelled).
		Select("products.name as name, SUM(order_items.quantity) as quantity, SUM(order_items.quantity * order_items.price) as revenue").
		Group("products.id").
		Order("quantity DESC").
		Limit(5).
		Scan(&items)
	return items
}
