package handlers

import (
	"sort"
	"strconv"
	"time"

	"brunocollective_inventory/database"
	"brunocollective_inventory/models"

	"github.com/gofiber/fiber/v2"
)

// AnalyticsHandler serves the fashion-business analytics behind the dashboard
// tabs (ภาพรวม / สต็อก / ลูกค้า / สินค้า). Data volumes are small-shop scale,
// so several metrics (RFM, ABC, bought-together) are computed in Go from
// loaded rows rather than pushed into SQL.
type AnalyticsHandler struct{}

func NewAnalyticsHandler() *AnalyticsHandler {
	return &AnalyticsHandler{}
}

// periodDays reads ?days= (7/30/90/365), defaulting to 30.
func periodDays(c *fiber.Ctx) int {
	days, err := strconv.Atoi(c.Query("days", "30"))
	if err != nil || days <= 0 || days > 3650 {
		return 30
	}
	return days
}

// queryDateRange reads ?from=YYYY-MM-DD&to=YYYY-MM-DD (both inclusive, server
// local time). Returns [from, to) with to advanced to the following midnight.
func queryDateRange(c *fiber.Ctx) (time.Time, time.Time, bool) {
	loc := time.Now().Location()
	from, errF := time.ParseInLocation("2006-01-02", c.Query("from"), loc)
	to, errT := time.ParseInLocation("2006-01-02", c.Query("to"), loc)
	if errF != nil || errT != nil || to.Before(from) {
		return time.Time{}, time.Time{}, false
	}
	return from, to.AddDate(0, 0, 1), true
}

// periodRange resolves the analytics window: an explicit from/to date range
// wins over ?days=N. The returned day count sizes the previous-period compare.
func periodRange(c *fiber.Ctx) (time.Time, time.Time, int) {
	if from, to, ok := queryDateRange(c); ok {
		return from, to, int(to.Sub(from).Hours() / 24)
	}
	days := periodDays(c)
	now := time.Now()
	return now.AddDate(0, 0, -days), now, days
}

// ---------------------------------------------------------------------------
// Overview tab
// ---------------------------------------------------------------------------

type periodTotals struct {
	Revenue     float64 `json:"revenue"`
	Orders      int     `json:"orders"`
	UnitsSold   int     `json:"units_sold"`
	AOV         float64 `json:"aov"`
	COGS        float64 `json:"cogs"`
	GrossProfit float64 `json:"gross_profit"`
	// MarginPct is gross profit / revenue. Items whose product has no cost
	// recorded contribute price as profit, so treat this as a lower bound on
	// cost coverage — CostCoverage tells the UI how trustworthy it is.
	MarginPct    float64 `json:"margin_pct"`
	CostCoverage float64 `json:"cost_coverage"` // share of units sold whose product has cost > 0
}

func (h *AnalyticsHandler) totalsBetween(from, to time.Time) periodTotals {
	var t periodTotals

	type orderRow struct {
		Revenue float64
		Orders  int
	}
	var or orderRow
	database.DB.Model(&models.Order{}).
		Where("status != ? AND created_at >= ? AND created_at < ?", models.StatusCancelled, from, to).
		Select("COALESCE(SUM(total_amount), 0) as revenue, COUNT(*) as orders").
		Scan(&or)
	t.Revenue = or.Revenue
	t.Orders = or.Orders

	type itemRow struct {
		Units       int
		COGS        float64
		UnitsCosted int
	}
	var ir itemRow
	database.DB.Model(&models.OrderItem{}).
		Joins("JOIN orders ON orders.id = order_items.order_id").
		Joins("JOIN products ON products.id = order_items.product_id").
		Where("orders.status != ? AND orders.created_at >= ? AND orders.created_at < ?", models.StatusCancelled, from, to).
		Select("COALESCE(SUM(order_items.quantity), 0) as units, " +
			"COALESCE(SUM(order_items.quantity * products.cost), 0) as cogs, " +
			"COALESCE(SUM(CASE WHEN products.cost > 0 THEN order_items.quantity ELSE 0 END), 0) as units_costed").
		Scan(&ir)
	t.UnitsSold = ir.Units
	t.COGS = ir.COGS
	t.GrossProfit = t.Revenue - t.COGS
	if t.Orders > 0 {
		t.AOV = t.Revenue / float64(t.Orders)
	}
	if t.Revenue > 0 {
		t.MarginPct = t.GrossProfit / t.Revenue * 100
	}
	if t.UnitsSold > 0 {
		t.CostCoverage = float64(ir.UnitsCosted) / float64(t.UnitsSold) * 100
	}
	return t
}

type channelRow struct {
	Channel string  `json:"channel"`
	Revenue float64 `json:"revenue"`
	Orders  int     `json:"orders"`
}

type categoryRow struct {
	Category string  `json:"category"`
	Revenue  float64 `json:"revenue"`
	Units    int     `json:"units"`
}

type couponRow struct {
	Code     string  `json:"code"`
	Uses     int     `json:"uses"`
	Revenue  float64 `json:"revenue"`
	Discount float64 `json:"discount"`
}

type salePageRow struct {
	Title       string  `json:"title"`
	Slug        string  `json:"slug"`
	Views       int     `json:"views"`
	OrdersCount int     `json:"orders_count"`
	BumpCount   int     `json:"bump_count"`
	Revenue     float64 `json:"revenue"`
	ConvPct     float64 `json:"conv_pct"`
}

// Overview returns headline totals (with previous-period comparison) plus
// revenue splits by channel and category, coupon performance, and sale-page
// funnel conversion. ?days=7|30|90|365 selects the window, or an explicit
// ?from=YYYY-MM-DD&to=YYYY-MM-DD range (inclusive) overrides it.
func (h *AnalyticsHandler) Overview(c *fiber.Ctx) error {
	from, to, days := periodRange(c)
	prevFrom := from.Add(-to.Sub(from))

	current := h.totalsBetween(from, to)
	previous := h.totalsBetween(prevFrom, from)

	var channels []channelRow
	database.DB.Model(&models.Order{}).
		Where("status != ? AND created_at >= ? AND created_at < ?", models.StatusCancelled, from, to).
		Select("CASE WHEN COALESCE(channel, '') = '' THEN 'unknown' ELSE channel END as channel, " +
			"COALESCE(SUM(total_amount), 0) as revenue, COUNT(*) as orders").
		Group("channel").
		Order("revenue DESC").
		Scan(&channels)

	var categories []categoryRow
	database.DB.Model(&models.OrderItem{}).
		Joins("JOIN orders ON orders.id = order_items.order_id").
		Joins("JOIN products ON products.id = order_items.product_id").
		Where("orders.status != ? AND orders.created_at >= ? AND orders.created_at < ?", models.StatusCancelled, from, to).
		Select("CASE WHEN COALESCE(products.category, '') = '' THEN 'ไม่ระบุหมวด' ELSE products.category END as category, " +
			"COALESCE(SUM(order_items.quantity * order_items.price), 0) as revenue, " +
			"COALESCE(SUM(order_items.quantity), 0) as units").
		Group("category").
		Order("revenue DESC").
		Scan(&categories)

	var coupons []couponRow
	database.DB.Model(&models.Order{}).
		Where("status != ? AND coupon_code != '' AND created_at >= ? AND created_at < ?", models.StatusCancelled, from, to).
		Select("coupon_code as code, COUNT(*) as uses, " +
			"COALESCE(SUM(total_amount), 0) as revenue, COALESCE(SUM(discount_amount), 0) as discount").
		Group("coupon_code").
		Order("revenue DESC").
		Scan(&coupons)

	// Sale pages: lifetime funnel stats from the page row + revenue attributed
	// via orders.sale_page_id (all time — views aren't windowed either).
	var pages []models.SalePage
	database.DB.Order("views DESC").Find(&pages)
	type pageRevenue struct {
		SalePageID uint
		Revenue    float64
	}
	var revs []pageRevenue
	database.DB.Model(&models.Order{}).
		Where("status != ? AND sale_page_id IS NOT NULL", models.StatusCancelled).
		Select("sale_page_id, COALESCE(SUM(total_amount), 0) as revenue").
		Group("sale_page_id").
		Scan(&revs)
	revByPage := make(map[uint]float64, len(revs))
	for _, r := range revs {
		revByPage[r.SalePageID] = r.Revenue
	}
	salePages := make([]salePageRow, 0, len(pages))
	for _, p := range pages {
		row := salePageRow{
			Title: p.Title, Slug: p.Slug,
			Views: p.Views, OrdersCount: p.OrdersCount, BumpCount: p.BumpCount,
			Revenue: revByPage[p.ID],
		}
		if p.Views > 0 {
			row.ConvPct = float64(p.OrdersCount) / float64(p.Views) * 100
		}
		salePages = append(salePages, row)
	}

	return c.JSON(fiber.Map{
		"days":       days,
		"current":    current,
		"previous":   previous,
		"channels":   channels,
		"categories": categories,
		"coupons":    coupons,
		"sale_pages": salePages,
	})
}

// ---------------------------------------------------------------------------
// Inventory tab
// ---------------------------------------------------------------------------

type inventoryProduct struct {
	ID           uint    `json:"id"`
	Name         string  `json:"name"`
	SKU          string  `json:"sku"`
	Category     string  `json:"category"`
	Stock        int     `json:"stock"`
	Price        float64 `json:"price"`
	Cost         float64 `json:"cost"`
	StockValue   float64 `json:"stock_value"` // stock × cost (price when cost unknown)
	Sold30       int     `json:"sold_30"`
	SoldTotal    int     `json:"sold_total"`
	LastSoldDays *int    `json:"last_sold_days"` // nil = never sold
	AgeDays      int     `json:"age_days"`
	SellThrough  float64 `json:"sell_through"` // sold ÷ (sold + stock), lifetime
	CoverDays    *int    `json:"cover_days"`   // stock ÷ 30-day run rate; nil = no recent sales
}

type sizeCurveRow struct {
	Size  string `json:"size"`
	Sold  int    `json:"sold"`
	Stock int    `json:"stock"`
}

type colorRow struct {
	Color string `json:"color"`
	Sold  int    `json:"sold"`
	Stock int    `json:"stock"`
}

// Inventory returns per-product movement metrics (sell-through, aging, stock
// cover, stock value) plus size-curve and color breakdowns. The frontend
// derives dead-stock lists from last_sold_days/age_days.
func (h *AnalyticsHandler) Inventory(c *fiber.Ctx) error {
	now := time.Now()
	from30 := now.AddDate(0, 0, -30)

	var products []models.Product
	database.DB.Preload("Variants").Find(&products)

	// Sold quantities per product: lifetime, last 30 days, and last-sold date.
	// Aggregated in Go — SQLite loses the datetime column type on MAX()
	// expressions, so scanning MAX(created_at) into time.Time fails.
	type soldRow struct {
		Sold     int
		LastSold time.Time
	}
	type itemSale struct {
		ProductID uint
		Quantity  int
		CreatedAt time.Time
	}
	var sales []itemSale
	database.DB.Model(&models.OrderItem{}).
		Joins("JOIN orders ON orders.id = order_items.order_id").
		Where("orders.status != ?", models.StatusCancelled).
		Select("order_items.product_id as product_id, order_items.quantity as quantity, orders.created_at as created_at").
		Scan(&sales)
	soldAll := make(map[uint]soldRow)
	sold30 := make(map[uint]soldRow)
	for _, s := range sales {
		agg := soldAll[s.ProductID]
		agg.Sold += s.Quantity
		if s.CreatedAt.After(agg.LastSold) {
			agg.LastSold = s.CreatedAt
		}
		soldAll[s.ProductID] = agg
		if s.CreatedAt.After(from30) {
			agg30 := sold30[s.ProductID]
			agg30.Sold += s.Quantity
			sold30[s.ProductID] = agg30
		}
	}

	items := make([]inventoryProduct, 0, len(products))
	var totalStockValue float64
	var deadStockValue float64
	for i := range products {
		p := &products[i]
		p.ComputeTotalStock()

		row := inventoryProduct{
			ID: p.ID, Name: p.Name, SKU: p.SKU, Category: p.Category,
			Stock: p.TotalStock, Price: p.Price, Cost: p.Cost,
			AgeDays: int(now.Sub(p.CreatedAt).Hours() / 24),
		}
		unitValue := p.Cost
		if unitValue == 0 {
			unitValue = p.Price
		}
		row.StockValue = float64(p.TotalStock) * unitValue

		if s, ok := soldAll[p.ID]; ok {
			row.SoldTotal = s.Sold
			d := int(now.Sub(s.LastSold).Hours() / 24)
			row.LastSoldDays = &d
		}
		if s, ok := sold30[p.ID]; ok {
			row.Sold30 = s.Sold
		}
		if row.SoldTotal+row.Stock > 0 {
			row.SellThrough = float64(row.SoldTotal) / float64(row.SoldTotal+row.Stock) * 100
		}
		if row.Sold30 > 0 {
			cover := int(float64(row.Stock) / (float64(row.Sold30) / 30.0))
			row.CoverDays = &cover
		}

		totalStockValue += row.StockValue
		// Dead stock: has stock but nothing sold in 60 days (or never, and the
		// product is older than 60 days).
		if row.Stock > 0 {
			if (row.LastSoldDays != nil && *row.LastSoldDays > 60) ||
				(row.LastSoldDays == nil && row.AgeDays > 60) {
				deadStockValue += row.StockValue
			}
		}

		items = append(items, row)
	}
	// Slowest movers first — that's what the tab is for.
	sort.Slice(items, func(a, b int) bool {
		av, bv := items[a], items[b]
		ad, bd := -1, -1
		if av.LastSoldDays != nil {
			ad = *av.LastSoldDays
		} else if av.Stock > 0 {
			ad = av.AgeDays
		}
		if bv.LastSoldDays != nil {
			bd = *bv.LastSoldDays
		} else if bv.Stock > 0 {
			bd = bv.AgeDays
		}
		return ad > bd
	})

	// Size curve: units sold per size vs current stock per size. Legacy items
	// with no size land in "ไม่ระบุ".
	category := c.Query("category")
	sizeSoldQ := database.DB.Model(&models.OrderItem{}).
		Joins("JOIN orders ON orders.id = order_items.order_id").
		Joins("JOIN products ON products.id = order_items.product_id").
		Where("orders.status != ?", models.StatusCancelled)
	if category != "" {
		sizeSoldQ = sizeSoldQ.Where("products.category = ?", category)
	}
	type soldBy struct {
		Key  string
		Sold int
	}
	var sizeSold []soldBy
	sizeSoldQ.Select("CASE WHEN COALESCE(order_items.size, '') = '' THEN 'ไม่ระบุ' ELSE order_items.size END as key, COALESCE(SUM(order_items.quantity), 0) as sold").
		Group("key").Scan(&sizeSold)

	stockBySize := make(map[string]int)
	stockByColor := make(map[string]int)
	for i := range products {
		p := &products[i]
		if category != "" && p.Category != category {
			continue
		}
		if len(p.Variants) == 0 {
			key := p.Size
			if key == "" {
				key = "ไม่ระบุ"
			}
			stockBySize[key] += p.Stock
			stockByColor["ไม่ระบุ"] += p.Stock
			continue
		}
		for _, v := range p.Variants {
			sKey, cKey := v.Size, v.Color
			if sKey == "" {
				sKey = "ไม่ระบุ"
			}
			if cKey == "" {
				cKey = "ไม่ระบุ"
			}
			stockBySize[sKey] += v.Stock
			stockByColor[cKey] += v.Stock
		}
	}

	sizeCurve := make([]sizeCurveRow, 0, len(sizeSold))
	seenSize := make(map[string]bool)
	for _, s := range sizeSold {
		sizeCurve = append(sizeCurve, sizeCurveRow{Size: s.Key, Sold: s.Sold, Stock: stockBySize[s.Key]})
		seenSize[s.Key] = true
	}
	for size, stock := range stockBySize {
		if !seenSize[size] && stock > 0 {
			sizeCurve = append(sizeCurve, sizeCurveRow{Size: size, Stock: stock})
		}
	}
	sort.Slice(sizeCurve, func(a, b int) bool { return sizeCurve[a].Sold > sizeCurve[b].Sold })

	var colorSold []soldBy
	colorSoldQ := database.DB.Model(&models.OrderItem{}).
		Joins("JOIN orders ON orders.id = order_items.order_id").
		Joins("JOIN products ON products.id = order_items.product_id").
		Where("orders.status != ?", models.StatusCancelled)
	if category != "" {
		colorSoldQ = colorSoldQ.Where("products.category = ?", category)
	}
	colorSoldQ.Select("CASE WHEN COALESCE(order_items.color, '') = '' THEN 'ไม่ระบุ' ELSE order_items.color END as key, COALESCE(SUM(order_items.quantity), 0) as sold").
		Group("key").Scan(&colorSold)
	colors := make([]colorRow, 0, len(colorSold))
	seenColor := make(map[string]bool)
	for _, s := range colorSold {
		colors = append(colors, colorRow{Color: s.Key, Sold: s.Sold, Stock: stockByColor[s.Key]})
		seenColor[s.Key] = true
	}
	for color, stock := range stockByColor {
		if !seenColor[color] && stock > 0 {
			colors = append(colors, colorRow{Color: color, Stock: stock})
		}
	}
	sort.Slice(colors, func(a, b int) bool { return colors[a].Sold > colors[b].Sold })

	return c.JSON(fiber.Map{
		"products":          items,
		"total_stock_value": totalStockValue,
		"dead_stock_value":  deadStockValue,
		"size_curve":        sizeCurve,
		"colors":            colors,
	})
}

// ---------------------------------------------------------------------------
// Customers tab
// ---------------------------------------------------------------------------

type rfmCustomer struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Phone       string    `json:"phone"`
	Orders      int       `json:"orders"`
	Spent       float64   `json:"spent"`
	LastOrderAt time.Time `json:"last_order_at"`
	RecencyDays int       `json:"recency_days"`
	Segment     string    `json:"segment"`
}

type monthlyNewReturning struct {
	Month     string `json:"month"`
	New       int    `json:"new"`
	Returning int    `json:"returning"`
}

// segmentOf buckets a customer by recency + frequency. Thresholds are tuned
// for a small fashion shop's repurchase cycle (~1-2 months).
func segmentOf(recencyDays, orders int) string {
	switch {
	case orders >= 3 && recencyDays <= 60:
		return "VIP"
	case orders >= 2 && recencyDays <= 90:
		return "ขาประจำ"
	case recencyDays > 180:
		return "หายไปแล้ว"
	case recencyDays > 90:
		return "กำลังจะหาย"
	case orders == 1:
		return "ลูกค้าใหม่"
	default:
		return "ทั่วไป"
	}
}

// Customers returns repeat-purchase stats, RFM segments with member lists,
// a 6-month new-vs-returning series, and top spenders.
func (h *AnalyticsHandler) Customers(c *fiber.Ctx) error {
	now := time.Now()

	// One row per customer with ≥1 non-cancelled order. Aggregated in Go (see
	// Inventory: MIN/MAX on datetime columns don't scan into time.Time).
	type custAgg struct {
		CustomerID uint
		Name       string
		Phone      string
		Orders     int
		Spent      float64
		FirstOrder time.Time
		LastOrder  time.Time
	}
	type orderRow struct {
		CustomerID  uint
		TotalAmount float64
		CreatedAt   time.Time
	}
	var allOrders []orderRow
	database.DB.Model(&models.Order{}).
		Where("status != ?", models.StatusCancelled).
		Select("customer_id, total_amount, created_at").
		Scan(&allOrders)

	var allCustomers []models.Customer
	database.DB.Find(&allCustomers)
	custByID := make(map[uint]models.Customer, len(allCustomers))
	for _, cu := range allCustomers {
		custByID[cu.ID] = cu
	}

	aggByID := make(map[uint]*custAgg)
	for _, o := range allOrders {
		a, ok := aggByID[o.CustomerID]
		if !ok {
			cu := custByID[o.CustomerID]
			a = &custAgg{CustomerID: o.CustomerID, Name: cu.Name, Phone: cu.Phone,
				FirstOrder: o.CreatedAt, LastOrder: o.CreatedAt}
			aggByID[o.CustomerID] = a
		}
		a.Orders++
		a.Spent += o.TotalAmount
		if o.CreatedAt.Before(a.FirstOrder) {
			a.FirstOrder = o.CreatedAt
		}
		if o.CreatedAt.After(a.LastOrder) {
			a.LastOrder = o.CreatedAt
		}
	}
	aggs := make([]custAgg, 0, len(aggByID))
	for _, a := range aggByID {
		aggs = append(aggs, *a)
	}

	customers := make([]rfmCustomer, 0, len(aggs))
	segments := make(map[string][]rfmCustomer)
	repeatCount := 0
	for _, a := range aggs {
		rec := int(now.Sub(a.LastOrder).Hours() / 24)
		cust := rfmCustomer{
			ID: a.CustomerID, Name: a.Name, Phone: a.Phone,
			Orders: a.Orders, Spent: a.Spent, LastOrderAt: a.LastOrder,
			RecencyDays: rec, Segment: segmentOf(rec, a.Orders),
		}
		customers = append(customers, cust)
		segments[cust.Segment] = append(segments[cust.Segment], cust)
		if a.Orders > 1 {
			repeatCount++
		}
	}
	// Within each segment, biggest spenders first.
	for _, list := range segments {
		sort.Slice(list, func(a, b int) bool { return list[a].Spent > list[b].Spent })
	}

	repeatRate := 0.0
	if len(aggs) > 0 {
		repeatRate = float64(repeatCount) / float64(len(aggs)) * 100
	}

	// New vs returning per month, last 6 months: an order is "new" when it's
	// that customer's first-ever order.
	firstByCustomer := make(map[uint]time.Time, len(aggs))
	for _, a := range aggs {
		firstByCustomer[a.CustomerID] = a.FirstOrder
	}
	sixMonthsAgo := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).AddDate(0, -5, 0)
	var recentOrders []orderRow
	for _, o := range allOrders {
		if !o.CreatedAt.Before(sixMonthsAgo) {
			recentOrders = append(recentOrders, o)
		}
	}

	monthly := make([]monthlyNewReturning, 0, 6)
	monthIdx := make(map[string]int)
	for i := 0; i < 6; i++ {
		m := sixMonthsAgo.AddDate(0, i, 0)
		key := m.Format("2006-01")
		monthIdx[key] = len(monthly)
		monthly = append(monthly, monthlyNewReturning{Month: m.Format("Jan 06")})
	}
	for _, o := range recentOrders {
		idx, ok := monthIdx[o.CreatedAt.Format("2006-01")]
		if !ok {
			continue
		}
		first := firstByCustomer[o.CustomerID]
		if o.CreatedAt.Equal(first) {
			monthly[idx].New++
		} else {
			monthly[idx].Returning++
		}
	}

	topSpenders := make([]rfmCustomer, len(customers))
	copy(topSpenders, customers)
	sort.Slice(topSpenders, func(a, b int) bool { return topSpenders[a].Spent > topSpenders[b].Spent })
	if len(topSpenders) > 10 {
		topSpenders = topSpenders[:10]
	}

	var totalCustomers int64
	database.DB.Model(&models.Customer{}).Count(&totalCustomers)

	return c.JSON(fiber.Map{
		"total_customers":  totalCustomers,
		"buyers":           len(aggs),
		"repeat_customers": repeatCount,
		"repeat_rate":      repeatRate,
		"segments":         segments,
		"monthly":          monthly,
		"top_spenders":     topSpenders,
	})
}

// ---------------------------------------------------------------------------
// Products tab
// ---------------------------------------------------------------------------

type abcProduct struct {
	ID       uint    `json:"id"`
	Name     string  `json:"name"`
	Category string  `json:"category"`
	Revenue  float64 `json:"revenue"`
	Units    int     `json:"units"`
	SharePct float64 `json:"share_pct"`
	CumPct   float64 `json:"cum_pct"`
	Class    string  `json:"class"`
}

type pairRow struct {
	ProductA string `json:"product_a"`
	ProductB string `json:"product_b"`
	Count    int    `json:"count"`
}

// Products returns an ABC (pareto) classification by revenue and the most
// frequently co-purchased product pairs.
func (h *AnalyticsHandler) Products(c *fiber.Ctx) error {
	type revRow struct {
		ProductID uint
		Name      string
		Category  string
		Revenue   float64
		Units     int
	}
	var rows []revRow
	database.DB.Model(&models.OrderItem{}).
		Joins("JOIN orders ON orders.id = order_items.order_id").
		Joins("JOIN products ON products.id = order_items.product_id").
		Where("orders.status != ?", models.StatusCancelled).
		Select("order_items.product_id as product_id, products.name as name, products.category as category, " +
			"COALESCE(SUM(order_items.quantity * order_items.price), 0) as revenue, " +
			"COALESCE(SUM(order_items.quantity), 0) as units").
		Group("order_items.product_id").
		Order("revenue DESC").
		Scan(&rows)

	var totalRevenue float64
	for _, r := range rows {
		totalRevenue += r.Revenue
	}
	abc := make([]abcProduct, 0, len(rows))
	cum := 0.0
	for _, r := range rows {
		share := 0.0
		if totalRevenue > 0 {
			share = r.Revenue / totalRevenue * 100
		}
		// Classify by the cumulative share *before* this product so the items
		// that build the first 80% are all A (a single product = 100% is A,
		// not C).
		class := "C"
		if cum < 80 {
			class = "A"
		} else if cum < 95 {
			class = "B"
		}
		cum += share
		abc = append(abc, abcProduct{
			ID: r.ProductID, Name: r.Name, Category: r.Category,
			Revenue: r.Revenue, Units: r.Units,
			SharePct: share, CumPct: cum, Class: class,
		})
	}

	// Bought-together pairs: distinct product pairs within the same order.
	type itemLite struct {
		OrderID   uint
		ProductID uint
		Name      string
	}
	var items []itemLite
	database.DB.Model(&models.OrderItem{}).
		Joins("JOIN orders ON orders.id = order_items.order_id").
		Joins("JOIN products ON products.id = order_items.product_id").
		Where("orders.status != ?", models.StatusCancelled).
		Select("order_items.order_id as order_id, order_items.product_id as product_id, products.name as name").
		Order("order_items.order_id").
		Scan(&items)

	byOrder := make(map[uint]map[uint]string)
	for _, it := range items {
		if byOrder[it.OrderID] == nil {
			byOrder[it.OrderID] = make(map[uint]string)
		}
		byOrder[it.OrderID][it.ProductID] = it.Name
	}
	type pairKey struct{ a, b uint }
	pairCounts := make(map[pairKey]int)
	pairNames := make(map[pairKey][2]string)
	for _, prods := range byOrder {
		ids := make([]uint, 0, len(prods))
		for id := range prods {
			ids = append(ids, id)
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		for i := 0; i < len(ids); i++ {
			for j := i + 1; j < len(ids); j++ {
				k := pairKey{ids[i], ids[j]}
				pairCounts[k]++
				pairNames[k] = [2]string{prods[ids[i]], prods[ids[j]]}
			}
		}
	}
	pairs := make([]pairRow, 0, len(pairCounts))
	for k, n := range pairCounts {
		if n < 2 {
			continue // a single co-occurrence isn't a pattern
		}
		names := pairNames[k]
		pairs = append(pairs, pairRow{ProductA: names[0], ProductB: names[1], Count: n})
	}
	sort.Slice(pairs, func(a, b int) bool { return pairs[a].Count > pairs[b].Count })
	if len(pairs) > 10 {
		pairs = pairs[:10]
	}

	return c.JSON(fiber.Map{
		"abc":             abc,
		"bought_together": pairs,
	})
}
