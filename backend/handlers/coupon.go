package handlers

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"brunocollective_inventory/database"
	"brunocollective_inventory/models"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type CouponHandler struct{}

func NewCouponHandler() *CouponHandler {
	return &CouponHandler{}
}

// normalizeCouponCode uppercases and trims a code so lookups are
// case-insensitive regardless of how the shopper typed it.
func normalizeCouponCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

// roundSatang rounds to 2 decimals so discounts never carry float dust.
func roundSatang(v float64) float64 {
	return math.Round(v*100) / 100
}

// computeCouponDiscount returns the discount a coupon gives on a subtotal,
// assuming conditions have already been checked. Never exceeds the subtotal.
func computeCouponDiscount(coupon *models.Coupon, subtotal float64) float64 {
	var discount float64
	switch coupon.Type {
	case models.CouponPercent:
		discount = subtotal * coupon.Value / 100
		if coupon.MaxDiscount > 0 && discount > coupon.MaxDiscount {
			discount = coupon.MaxDiscount
		}
	case models.CouponFixed:
		discount = coupon.Value
	}
	if discount > subtotal {
		discount = subtotal
	}
	return roundSatang(discount)
}

// validateCouponForCart looks up a coupon by code and checks every phase-1
// condition against the cart subtotal and (optional) customer. customerID 0
// means "unknown customer" (e.g. storefront preview before checkout) — the
// per-customer limit is then checked again at order time when the customer is
// known. Error messages are Thai because they are shown to shoppers as-is.
func validateCouponForCart(tx *gorm.DB, code string, subtotal float64, customerID uint) (*models.Coupon, float64, error) {
	code = normalizeCouponCode(code)
	if code == "" {
		return nil, 0, fiber.NewError(fiber.StatusBadRequest, "กรุณาระบุโค้ดส่วนลด")
	}

	var coupon models.Coupon
	if err := tx.Where("code = ?", code).First(&coupon).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, 0, fiber.NewError(fiber.StatusNotFound, "ไม่พบโค้ดส่วนลดนี้")
		}
		return nil, 0, err
	}

	if !coupon.IsActive {
		return nil, 0, fiber.NewError(fiber.StatusBadRequest, "โค้ดนี้ปิดใช้งานอยู่")
	}

	now := time.Now()
	if coupon.StartsAt != nil && now.Before(*coupon.StartsAt) {
		return nil, 0, fiber.NewError(fiber.StatusBadRequest,
			"โค้ดนี้เริ่มใช้ได้วันที่ "+coupon.StartsAt.Format("02/01/2006"))
	}
	if coupon.ExpiresAt != nil && now.After(*coupon.ExpiresAt) {
		return nil, 0, fiber.NewError(fiber.StatusBadRequest, "โค้ดนี้หมดอายุแล้ว")
	}

	if coupon.UsageLimit > 0 && coupon.UsedCount >= coupon.UsageLimit {
		return nil, 0, fiber.NewError(fiber.StatusBadRequest, "โค้ดนี้ถูกใช้ครบจำนวนสิทธิ์แล้ว")
	}

	if coupon.MinOrderAmount > 0 && subtotal < coupon.MinOrderAmount {
		missing := coupon.MinOrderAmount - subtotal
		return nil, 0, fiber.NewError(fiber.StatusBadRequest,
			fmt.Sprintf("โค้ดนี้ใช้ได้เมื่อยอดสั่งซื้อครบ %.0f บาท (ขาดอีก %.2f บาท)", coupon.MinOrderAmount, missing))
	}

	if coupon.UsageLimitPerCustomer > 0 && customerID != 0 {
		var used int64
		if err := tx.Model(&models.CouponRedemption{}).
			Where("coupon_id = ? AND customer_id = ?", coupon.ID, customerID).
			Count(&used).Error; err != nil {
			return nil, 0, err
		}
		if used >= int64(coupon.UsageLimitPerCustomer) {
			return nil, 0, fiber.NewError(fiber.StatusBadRequest, "คุณใช้โค้ดนี้ครบจำนวนสิทธิ์แล้ว")
		}
	}

	return &coupon, computeCouponDiscount(&coupon, subtotal), nil
}

// claimCoupon consumes one usage of the coupon and records the redemption.
// It must run inside the same transaction that creates the order. The guarded
// UPDATE makes the total-limit check atomic: two concurrent orders racing for
// the last slot can't both win.
func claimCoupon(tx *gorm.DB, coupon *models.Coupon, orderID, customerID uint, discount float64) error {
	res := tx.Model(&models.Coupon{}).
		Where("id = ? AND is_active = ? AND (usage_limit = 0 OR used_count < usage_limit)", coupon.ID, true).
		Update("used_count", gorm.Expr("used_count + 1"))
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "โค้ดนี้ถูกใช้ครบจำนวนสิทธิ์แล้ว")
	}

	return tx.Create(&models.CouponRedemption{
		CouponID:       coupon.ID,
		OrderID:        orderID,
		CustomerID:     customerID,
		DiscountAmount: discount,
	}).Error
}

// applyCouponToOrder validates + claims a coupon for a freshly created order
// and stamps the discount fields onto it. Call inside the order-creation
// transaction, after tx.Create(&order) (the redemption needs the order ID).
// An empty code is a no-op so both order paths can call it unconditionally.
// The coupon discount is computed on the full subtotal — independent of the
// membership discount (the two stack) — but is capped so the payable total
// never goes below zero when both apply.
func applyCouponToOrder(tx *gorm.DB, order *models.Order, code string, subtotal float64) error {
	order.Subtotal = subtotal
	if normalizeCouponCode(code) == "" {
		return nil
	}

	coupon, discount, err := validateCouponForCart(tx, code, subtotal, order.CustomerID)
	if err != nil {
		return err
	}
	if remaining := roundSatang(subtotal - order.MemberDiscount); discount > remaining {
		discount = remaining
	}
	if err := claimCoupon(tx, coupon, order.ID, order.CustomerID, discount); err != nil {
		return err
	}

	order.CouponID = &coupon.ID
	order.CouponCode = coupon.Code
	order.DiscountAmount = discount
	order.TotalAmount = roundSatang(subtotal - order.MemberDiscount - discount)

	return tx.Model(order).Updates(map[string]interface{}{
		"subtotal":        order.Subtotal,
		"discount_amount": order.DiscountAmount,
		"coupon_id":       order.CouponID,
		"coupon_code":     order.CouponCode,
		"total_amount":    order.TotalAmount,
	}).Error
}

// releaseCouponForOrder returns any coupon usage consumed by an order — used
// when the order is deleted, mirroring how stock is restored. UsedCount is
// guarded so it never goes below zero.
func releaseCouponForOrder(tx *gorm.DB, orderID uint) error {
	var redemptions []models.CouponRedemption
	if err := tx.Where("order_id = ?", orderID).Find(&redemptions).Error; err != nil {
		return err
	}
	for _, r := range redemptions {
		if err := tx.Model(&models.Coupon{}).
			Where("id = ? AND used_count > 0", r.CouponID).
			Update("used_count", gorm.Expr("used_count - 1")).Error; err != nil {
			return err
		}
	}
	return tx.Where("order_id = ?", orderID).Delete(&models.CouponRedemption{}).Error
}

// ---- Admin CRUD ----

type couponRequest struct {
	Code                  string     `json:"code"`
	Name                  string     `json:"name"`
	Description           string     `json:"description"`
	Type                  string     `json:"type"`
	Value                 float64    `json:"value"`
	MaxDiscount           float64    `json:"max_discount"`
	MinOrderAmount        float64    `json:"min_order_amount"`
	StartsAt              *time.Time `json:"starts_at"`
	ExpiresAt             *time.Time `json:"expires_at"`
	UsageLimit            int        `json:"usage_limit"`
	UsageLimitPerCustomer int        `json:"usage_limit_per_customer"`
	IsActive              *bool      `json:"is_active"`
}

func (r *couponRequest) validate() error {
	r.Code = normalizeCouponCode(r.Code)
	if r.Code == "" {
		return fiber.NewError(fiber.StatusBadRequest, "กรุณาระบุโค้ด")
	}
	switch models.CouponType(r.Type) {
	case models.CouponPercent:
		if r.Value <= 0 || r.Value > 100 {
			return fiber.NewError(fiber.StatusBadRequest, "ส่วนลดแบบเปอร์เซ็นต์ต้องอยู่ระหว่าง 1-100")
		}
	case models.CouponFixed:
		if r.Value <= 0 {
			return fiber.NewError(fiber.StatusBadRequest, "จำนวนเงินส่วนลดต้องมากกว่า 0")
		}
	default:
		return fiber.NewError(fiber.StatusBadRequest, "ประเภทคูปองไม่ถูกต้อง")
	}
	if r.MaxDiscount < 0 || r.MinOrderAmount < 0 || r.UsageLimit < 0 || r.UsageLimitPerCustomer < 0 {
		return fiber.NewError(fiber.StatusBadRequest, "ค่าต้องไม่ติดลบ")
	}
	if r.StartsAt != nil && r.ExpiresAt != nil && r.ExpiresAt.Before(*r.StartsAt) {
		return fiber.NewError(fiber.StatusBadRequest, "วันหมดอายุต้องอยู่หลังวันเริ่ม")
	}
	return nil
}

func couponHTTPError(c *fiber.Ctx, err error) error {
	if fe, ok := err.(*fiber.Error); ok {
		return c.Status(fe.Code).JSON(fiber.Map{"error": fe.Message})
	}
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
}

func (h *CouponHandler) List(c *fiber.Ctx) error {
	var coupons []models.Coupon
	database.DB.Order("created_at DESC").Find(&coupons)
	return c.JSON(coupons)
}

func (h *CouponHandler) Get(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var coupon models.Coupon
	if err := database.DB.First(&coupon, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "coupon not found"})
	}
	return c.JSON(coupon)
}

func (h *CouponHandler) Create(c *fiber.Ctx) error {
	var req couponRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if err := req.validate(); err != nil {
		return couponHTTPError(c, err)
	}

	var count int64
	database.DB.Model(&models.Coupon{}).Where("code = ?", req.Code).Count(&count)
	if count > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "มีโค้ดนี้อยู่แล้ว"})
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	coupon := models.Coupon{
		Code:                  req.Code,
		Name:                  strings.TrimSpace(req.Name),
		Description:           strings.TrimSpace(req.Description),
		Type:                  models.CouponType(req.Type),
		Value:                 req.Value,
		MaxDiscount:           req.MaxDiscount,
		MinOrderAmount:        req.MinOrderAmount,
		StartsAt:              req.StartsAt,
		ExpiresAt:             req.ExpiresAt,
		UsageLimit:            req.UsageLimit,
		UsageLimitPerCustomer: req.UsageLimitPerCustomer,
		IsActive:              isActive,
	}
	if err := database.DB.Create(&coupon).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create coupon"})
	}
	return c.Status(fiber.StatusCreated).JSON(coupon)
}

func (h *CouponHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var coupon models.Coupon
	if err := database.DB.First(&coupon, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "coupon not found"})
	}

	var req couponRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if err := req.validate(); err != nil {
		return couponHTTPError(c, err)
	}

	var count int64
	database.DB.Model(&models.Coupon{}).Where("code = ? AND id <> ?", req.Code, coupon.ID).Count(&count)
	if count > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "มีโค้ดนี้อยู่แล้ว"})
	}

	updates := map[string]interface{}{
		"code":                     req.Code,
		"name":                     strings.TrimSpace(req.Name),
		"description":              strings.TrimSpace(req.Description),
		"type":                     req.Type,
		"value":                    req.Value,
		"max_discount":             req.MaxDiscount,
		"min_order_amount":         req.MinOrderAmount,
		"starts_at":                req.StartsAt,
		"expires_at":               req.ExpiresAt,
		"usage_limit":              req.UsageLimit,
		"usage_limit_per_customer": req.UsageLimitPerCustomer,
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if err := database.DB.Model(&coupon).Updates(updates).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update coupon"})
	}
	database.DB.First(&coupon, id)
	return c.JSON(coupon)
}

// Toggle flips the active switch — a quick pause/resume without editing.
func (h *CouponHandler) Toggle(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var coupon models.Coupon
	if err := database.DB.First(&coupon, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "coupon not found"})
	}
	database.DB.Model(&coupon).Update("is_active", !coupon.IsActive)
	database.DB.First(&coupon, id)
	return c.JSON(coupon)
}

// Delete removes the coupon. Past orders are unaffected: they carry a snapshot
// of the code and discount, and redemption rows are kept for history.
func (h *CouponHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	if err := database.DB.Delete(&models.Coupon{}, id).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete coupon"})
	}
	return c.JSON(fiber.Map{"message": "coupon deleted"})
}

// Redemptions returns the usage history of one coupon, newest first.
func (h *CouponHandler) Redemptions(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var redemptions []models.CouponRedemption
	database.DB.Where("coupon_id = ?", id).Order("created_at DESC").Find(&redemptions)
	return c.JSON(redemptions)
}

// ---- Validation (preview) ----

// validateRequest is shared by the admin order form and the public storefront.
// The subtotal is client-computed for preview only — the authoritative discount
// is recomputed from DB prices inside the order-creation transaction.
type validateCouponBody struct {
	Code       string  `json:"code"`
	Subtotal   float64 `json:"subtotal"`
	CustomerID uint    `json:"customer_id"` // admin order form
	Phone      string  `json:"phone"`       // storefront checkout (customer may not exist yet)
}

// Validate previews a coupon against a cart: returns the discount it would
// give, or a Thai-language reason it can't be used.
func (h *CouponHandler) Validate(c *fiber.Ctx) error {
	var body validateCouponBody
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	customerID := body.CustomerID
	if customerID == 0 && strings.TrimSpace(body.Phone) != "" {
		var customer models.Customer
		if err := database.DB.Where("phone = ?", strings.TrimSpace(body.Phone)).
			First(&customer).Error; err == nil {
			customerID = customer.ID
		}
	}

	coupon, discount, err := validateCouponForCart(database.DB, body.Code, body.Subtotal, customerID)
	if err != nil {
		return couponHTTPError(c, err)
	}

	return c.JSON(fiber.Map{
		"valid":    true,
		"code":     coupon.Code,
		"name":     coupon.Name,
		"type":     coupon.Type,
		"value":    coupon.Value,
		"discount": discount,
		"total":    roundSatang(body.Subtotal - discount),
	})
}
