package handlers

import (
	"strconv"
	"time"

	"brunocollective_inventory/config"
	"brunocollective_inventory/database"
	"brunocollective_inventory/models"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AffiliateHandler manages the affiliate/referral program: admin CRUD +
// reporting + payouts, the public track/validate endpoints, and (in
// affiliate_portal.go) the affiliate's own login portal.
type AffiliateHandler struct {
	Config *config.Config
}

func NewAffiliateHandler(cfg *config.Config) *AffiliateHandler {
	return &AffiliateHandler{Config: cfg}
}

// ---- Commission engine (shared with the order-creation paths) ----

// commissionFactor pro-rates order-level discounts (coupon + member) onto
// items: TotalAmount/Subtotal — the same convention as analytics'
// itemNetRevenue. Legacy orders with Subtotal 0 use factor 1.
func commissionFactor(order *models.Order) float64 {
	if order.Subtotal > 0 {
		return order.TotalAmount / order.Subtotal
	}
	return 1
}

// resolveCommissionRate picks the product override when set, else the
// affiliate's default. A product override of 0 means "no commission".
func resolveCommissionRate(aff *models.Affiliate, p *models.Product) float64 {
	if p != nil && p.CommissionPercent != nil {
		return *p.CommissionPercent
	}
	return aff.CommissionPercent
}

// applyAffiliateToOrder attributes a freshly created order to an affiliate and
// creates the pending commission ledger rows. Call inside the order-creation
// transaction AFTER tx.Create(&order) AND after applyCouponToOrder (so the
// discount factor sees the final totals). An empty/unknown/inactive code is a
// silent no-op — attribution must never fail a sale.
func applyAffiliateToOrder(tx *gorm.DB, order *models.Order, code string) error {
	code = normalizeCouponCode(code)
	if code == "" {
		return nil
	}

	var aff models.Affiliate
	if err := tx.Where("code = ? AND is_active = ?", code, true).First(&aff).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil // stale/typo'd ref — ignore, the sale goes through
		}
		return err
	}

	// Self-referral guard: an affiliate buying through their own link earns
	// nothing (matched by phone, the shared identity key).
	var customer models.Customer
	if err := tx.First(&customer, order.CustomerID).Error; err == nil {
		if aff.Phone != "" && customer.Phone != "" && aff.Phone == customer.Phone {
			return nil
		}
	}

	order.AffiliateID = &aff.ID
	order.AffiliateCode = aff.Code
	if err := tx.Model(order).Updates(map[string]interface{}{
		"affiliate_id":   order.AffiliateID,
		"affiliate_code": order.AffiliateCode,
	}).Error; err != nil {
		return err
	}

	// One ledger row per commission-bearing item. Item IDs exist because
	// tx.Create(&order) back-filled them.
	ids := make([]uint, 0, len(order.Items))
	for _, item := range order.Items {
		ids = append(ids, item.ProductID)
	}
	var products []models.Product
	if len(ids) > 0 {
		if err := tx.Where("id IN ?", ids).Find(&products).Error; err != nil {
			return err
		}
	}
	byID := make(map[uint]*models.Product, len(products))
	for i := range products {
		byID[products[i].ID] = &products[i]
	}

	factor := commissionFactor(order)
	for _, item := range order.Items {
		rate := resolveCommissionRate(&aff, byID[item.ProductID])
		if rate <= 0 {
			continue
		}
		base := roundSatang(float64(item.Quantity) * item.Price * factor)
		row := models.AffiliateCommission{
			AffiliateID: aff.ID,
			OrderID:     order.ID,
			OrderItemID: item.ID,
			ProductID:   item.ProductID,
			CustomerID:  order.CustomerID,
			RatePercent: rate,
			BaseAmount:  base,
			Amount:      roundSatang(base * rate / 100),
			Status:      models.CommissionPending,
		}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
	}
	return nil
}

// recomputeAffiliateCommissions refreshes Base/Amount from the order's
// current numbers (rates stay snapshotted). Only pending/confirmed rows move —
// paid and cancelled are final.
func recomputeAffiliateCommissions(tx *gorm.DB, orderID uint) error {
	var order models.Order
	if err := tx.Preload("Items").First(&order, orderID).Error; err != nil {
		return err
	}
	itemsByID := make(map[uint]models.OrderItem, len(order.Items))
	for _, item := range order.Items {
		itemsByID[item.ID] = item
	}

	var rows []models.AffiliateCommission
	if err := tx.Where("order_id = ? AND status IN ?", orderID,
		[]models.CommissionStatus{models.CommissionPending, models.CommissionConfirmed}).
		Find(&rows).Error; err != nil {
		return err
	}

	factor := commissionFactor(&order)
	for _, row := range rows {
		item, ok := itemsByID[row.OrderItemID]
		if !ok {
			continue
		}
		base := roundSatang(float64(item.Quantity) * item.Price * factor)
		if err := tx.Model(&models.AffiliateCommission{}).Where("id = ?", row.ID).
			Updates(map[string]interface{}{
				"base_amount": base,
				"amount":      roundSatang(base * row.RatePercent / 100),
			}).Error; err != nil {
			return err
		}
	}
	return nil
}

// confirmAffiliateCommissions locks in the amounts (recompute) and marks
// pending rows confirmed — called when the order reaches "delivered".
func confirmAffiliateCommissions(tx *gorm.DB, orderID uint) error {
	if err := recomputeAffiliateCommissions(tx, orderID); err != nil {
		return err
	}
	now := time.Now()
	return tx.Model(&models.AffiliateCommission{}).
		Where("order_id = ? AND status = ?", orderID, models.CommissionPending).
		Updates(map[string]interface{}{"status": models.CommissionConfirmed, "confirmed_at": now}).Error
}

// cancelAffiliateCommissions voids unpaid rows when the order is cancelled.
func cancelAffiliateCommissions(tx *gorm.DB, orderID uint) error {
	return tx.Model(&models.AffiliateCommission{}).
		Where("order_id = ? AND status IN ?", orderID,
			[]models.CommissionStatus{models.CommissionPending, models.CommissionConfirmed}).
		Update("status", models.CommissionCancelled).Error
}

// unconfirmAffiliateCommissions reverts confirmed rows to pending — the order
// moved back out of "delivered" (paid rows stay paid).
func unconfirmAffiliateCommissions(tx *gorm.DB, orderID uint) error {
	return tx.Model(&models.AffiliateCommission{}).
		Where("order_id = ? AND status = ?", orderID, models.CommissionConfirmed).
		Updates(map[string]interface{}{"status": models.CommissionPending, "confirmed_at": nil}).Error
}

// restoreAffiliateCommissions un-cancels rows when a cancelled order comes
// back to life.
func restoreAffiliateCommissions(tx *gorm.DB, orderID uint) error {
	return tx.Model(&models.AffiliateCommission{}).
		Where("order_id = ? AND status = ?", orderID, models.CommissionCancelled).
		Update("status", models.CommissionPending).Error
}

// releaseAffiliateForOrder removes the ledger rows when an order is deleted —
// mirrors releaseCouponForOrder/stock restoration.
func releaseAffiliateForOrder(tx *gorm.DB, orderID uint) error {
	return tx.Where("order_id = ?", orderID).Delete(&models.AffiliateCommission{}).Error
}

// ---- Public endpoints (storefront) ----

type affiliateCodeRequest struct {
	Code string `json:"code"`
}

// Track counts a ?ref link hit. Deliberately quiet: link refs must never
// surface errors to shoppers, so it always answers {valid: bool}.
func (h *AffiliateHandler) Track(c *fiber.Ctx) error {
	var req affiliateCodeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.JSON(fiber.Map{"valid": false})
	}
	code := normalizeCouponCode(req.Code)
	if code == "" {
		return c.JSON(fiber.Map{"valid": false})
	}
	res := database.DB.Model(&models.Affiliate{}).
		Where("code = ? AND is_active = ?", code, true).
		Update("click_count", gorm.Expr("click_count + 1"))
	return c.JSON(fiber.Map{"valid": res.Error == nil && res.RowsAffected > 0})
}

// Validate checks a typed referral code before checkout. Thai errors — shown
// to shoppers as-is (same convention as coupons).
func (h *AffiliateHandler) Validate(c *fiber.Ctx) error {
	var req affiliateCodeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	code := normalizeCouponCode(req.Code)
	if code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "กรุณาระบุรหัสผู้แนะนำ"})
	}
	var aff models.Affiliate
	if err := database.DB.Where("code = ?", code).First(&aff).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "ไม่พบรหัสผู้แนะนำนี้"})
	}
	if !aff.IsActive {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "รหัสผู้แนะนำนี้ปิดใช้งานอยู่"})
	}
	return c.JSON(fiber.Map{"valid": true, "code": aff.Code, "name": aff.Name})
}

// ---- Admin CRUD + reporting + payout ----

type affiliateRequest struct {
	Code              string  `json:"code"`
	Name              string  `json:"name"`
	Phone             string  `json:"phone"`
	Email             string  `json:"email"`
	Password          string  `json:"password"`
	CommissionPercent float64 `json:"commission_percent"`
	IsActive          *bool   `json:"is_active"`
	Notes             string  `json:"notes"`
}

func (r *affiliateRequest) validate(requirePassword bool) error {
	r.Code = normalizeCouponCode(r.Code)
	if r.Code == "" || r.Name == "" || r.Phone == "" {
		return fiber.NewError(fiber.StatusBadRequest, "กรุณากรอกโค้ด ชื่อ และเบอร์โทร")
	}
	if r.CommissionPercent < 0 || r.CommissionPercent > 100 {
		return fiber.NewError(fiber.StatusBadRequest, "เปอร์เซ็นต์ค่าคอมมิชชั่นต้องอยู่ระหว่าง 0–100")
	}
	if requirePassword && len(r.Password) < 6 {
		return fiber.NewError(fiber.StatusBadRequest, "รหัสผ่านต้องมีอย่างน้อย 6 ตัวอักษร")
	}
	if r.Password != "" && len(r.Password) < 6 {
		return fiber.NewError(fiber.StatusBadRequest, "รหัสผ่านต้องมีอย่างน้อย 6 ตัวอักษร")
	}
	return nil
}

func affiliateHTTPError(c *fiber.Ctx, err error) error {
	if fe, ok := err.(*fiber.Error); ok {
		return c.Status(fe.Code).JSON(fiber.Map{"error": fe.Message})
	}
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
}

// affiliateStats aggregates the ledger for one or all affiliates.
type affiliateStatsRow struct {
	AffiliateID uint
	Status      models.CommissionStatus
	Orders      int64
	Amount      float64
}

func loadAffiliateStats(affiliateID uint) (map[uint]map[models.CommissionStatus]affiliateStatsRow, error) {
	q := database.DB.Model(&models.AffiliateCommission{}).
		Select("affiliate_id, status, COUNT(DISTINCT order_id) as orders, SUM(amount) as amount").
		Group("affiliate_id, status")
	if affiliateID != 0 {
		q = q.Where("affiliate_id = ?", affiliateID)
	}
	var rows []affiliateStatsRow
	if err := q.Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[uint]map[models.CommissionStatus]affiliateStatsRow)
	for _, r := range rows {
		if out[r.AffiliateID] == nil {
			out[r.AffiliateID] = make(map[models.CommissionStatus]affiliateStatsRow)
		}
		out[r.AffiliateID][r.Status] = r
	}
	return out, nil
}

func affiliateWithStats(aff *models.Affiliate, stats map[models.CommissionStatus]affiliateStatsRow) fiber.Map {
	get := func(s models.CommissionStatus) (float64, int64) {
		if r, ok := stats[s]; ok {
			return roundSatang(r.Amount), r.Orders
		}
		return 0, 0
	}
	pending, pendingOrders := get(models.CommissionPending)
	confirmed, confirmedOrders := get(models.CommissionConfirmed)
	paid, paidOrders := get(models.CommissionPaid)
	cancelled, _ := get(models.CommissionCancelled)
	return fiber.Map{
		"affiliate":        aff,
		"pending_amount":   pending,
		"confirmed_amount": confirmed,
		"paid_amount":      paid,
		"cancelled_amount": cancelled,
		"orders_count":     pendingOrders + confirmedOrders + paidOrders,
	}
}

func (h *AffiliateHandler) List(c *fiber.Ctx) error {
	var affiliates []models.Affiliate
	database.DB.Order("id DESC").Find(&affiliates)
	stats, err := loadAffiliateStats(0)
	if err != nil {
		return affiliateHTTPError(c, err)
	}
	out := make([]fiber.Map, 0, len(affiliates))
	for i := range affiliates {
		out = append(out, affiliateWithStats(&affiliates[i], stats[affiliates[i].ID]))
	}
	return c.JSON(out)
}

func (h *AffiliateHandler) Create(c *fiber.Ctx) error {
	var req affiliateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if err := req.validate(true); err != nil {
		return affiliateHTTPError(c, err)
	}

	var count int64
	database.DB.Model(&models.Affiliate{}).Where("code = ?", req.Code).Count(&count)
	if count > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "มีโค้ดนี้อยู่แล้ว"})
	}
	database.DB.Model(&models.Affiliate{}).Where("phone = ?", req.Phone).Count(&count)
	if count > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "เบอร์โทรนี้ถูกใช้แล้ว"})
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return affiliateHTTPError(c, err)
	}
	aff := models.Affiliate{
		Code:              req.Code,
		Name:              req.Name,
		Phone:             req.Phone,
		Email:             req.Email,
		PasswordHash:      string(hash),
		CommissionPercent: req.CommissionPercent,
		IsActive:          req.IsActive == nil || *req.IsActive,
		Notes:             req.Notes,
	}
	if err := database.DB.Create(&aff).Error; err != nil {
		return affiliateHTTPError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(aff)
}

func (h *AffiliateHandler) Get(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var aff models.Affiliate
	if err := database.DB.First(&aff, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "affiliate not found"})
	}
	return c.JSON(aff)
}

func (h *AffiliateHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var aff models.Affiliate
	if err := database.DB.First(&aff, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "affiliate not found"})
	}

	var req affiliateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if err := req.validate(false); err != nil {
		return affiliateHTTPError(c, err)
	}

	var count int64
	database.DB.Model(&models.Affiliate{}).Where("code = ? AND id != ?", req.Code, aff.ID).Count(&count)
	if count > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "มีโค้ดนี้อยู่แล้ว"})
	}
	database.DB.Model(&models.Affiliate{}).Where("phone = ? AND id != ?", req.Phone, aff.ID).Count(&count)
	if count > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "เบอร์โทรนี้ถูกใช้แล้ว"})
	}

	updates := map[string]interface{}{
		"code":               req.Code,
		"name":               req.Name,
		"phone":              req.Phone,
		"email":              req.Email,
		"commission_percent": req.CommissionPercent,
		"notes":              req.Notes,
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if req.Password != "" { // optional password reset
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return affiliateHTTPError(c, err)
		}
		updates["password_hash"] = string(hash)
	}
	if err := database.DB.Model(&aff).Updates(updates).Error; err != nil {
		return affiliateHTTPError(c, err)
	}
	database.DB.First(&aff, aff.ID)
	return c.JSON(aff)
}

// Delete removes the affiliate account. Past orders keep the code snapshot and
// their ledger rows stay for history (mirrors coupon deletion semantics).
func (h *AffiliateHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	if err := database.DB.Delete(&models.Affiliate{}, id).Error; err != nil {
		return affiliateHTTPError(c, err)
	}
	return c.JSON(fiber.Map{"message": "affiliate deleted"})
}

func (h *AffiliateHandler) Toggle(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var aff models.Affiliate
	if err := database.DB.First(&aff, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "affiliate not found"})
	}
	if err := database.DB.Model(&aff).Update("is_active", !aff.IsActive).Error; err != nil {
		return affiliateHTTPError(c, err)
	}
	aff.IsActive = !aff.IsActive
	return c.JSON(aff)
}

// affiliateOrderRow is one attributed order in a report/portal listing.
// Commission rows of one order always share a status (transitions are
// order-scoped), so grouping by order+status yields one row per order.
type affiliateOrderRow struct {
	OrderID          uint      `json:"order_id"`
	CreatedAt        time.Time `json:"created_at"`
	OrderStatus      string    `json:"order_status"`
	CustomerName     string    `json:"customer_name,omitempty"`
	ItemCount        int       `json:"item_count"`
	OrderTotal       float64   `json:"order_total"`
	Commission       float64   `json:"commission"`
	CommissionStatus string    `json:"commission_status"`
}

func affiliateOrderRows(affiliateID uint, withCustomer bool, limit int) ([]affiliateOrderRow, error) {
	sel := "affiliate_commissions.order_id, orders.created_at, orders.status as order_status, " +
		"COUNT(affiliate_commissions.id) as item_count, orders.total_amount as order_total, " +
		"SUM(affiliate_commissions.amount) as commission, affiliate_commissions.status as commission_status"
	if withCustomer {
		sel += ", customers.name as customer_name"
	}
	q := database.DB.Model(&models.AffiliateCommission{}).
		Select(sel).
		Joins("JOIN orders ON orders.id = affiliate_commissions.order_id").
		Where("affiliate_commissions.affiliate_id = ?", affiliateID).
		Group("affiliate_commissions.order_id, affiliate_commissions.status").
		Order("orders.created_at DESC").
		Limit(limit)
	if withCustomer {
		q = q.Joins("JOIN customers ON customers.id = orders.customer_id")
	}
	// Initialized (not nil) so an empty result serializes as [] — the portal
	// crashes on null.length otherwise.
	rows := make([]affiliateOrderRow, 0)
	err := q.Scan(&rows).Error
	return rows, err
}

// Report returns the per-affiliate admin report: lifetime totals by status +
// recent attributed orders (with customer names — admin only).
func (h *AffiliateHandler) Report(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var aff models.Affiliate
	if err := database.DB.First(&aff, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "affiliate not found"})
	}
	stats, err := loadAffiliateStats(aff.ID)
	if err != nil {
		return affiliateHTTPError(c, err)
	}
	orders, err := affiliateOrderRows(aff.ID, true, 200)
	if err != nil {
		return affiliateHTTPError(c, err)
	}
	out := affiliateWithStats(&aff, stats[aff.ID])
	out["orders"] = orders
	return c.JSON(out)
}

// Pay marks every confirmed commission as paid, in one transaction. The
// guarded WHERE makes a double-click harmless (second call pays 0).
func (h *AffiliateHandler) Pay(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var aff models.Affiliate
	if err := database.DB.First(&aff, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "affiliate not found"})
	}

	var count int64
	var amount float64
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		var sum struct {
			N int64
			A float64
		}
		if err := tx.Model(&models.AffiliateCommission{}).
			Select("COUNT(*) as n, COALESCE(SUM(amount),0) as a").
			Where("affiliate_id = ? AND status = ?", aff.ID, models.CommissionConfirmed).
			Scan(&sum).Error; err != nil {
			return err
		}
		count, amount = sum.N, roundSatang(sum.A)
		now := time.Now()
		return tx.Model(&models.AffiliateCommission{}).
			Where("affiliate_id = ? AND status = ?", aff.ID, models.CommissionConfirmed).
			Updates(map[string]interface{}{"status": models.CommissionPaid, "paid_at": now}).Error
	})
	if err != nil {
		return affiliateHTTPError(c, err)
	}
	return c.JSON(fiber.Map{"count": count, "amount": amount})
}
