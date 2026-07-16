package handlers

import (
	"strings"
	"time"

	"brunocollective_inventory/config"
	"brunocollective_inventory/database"
	"brunocollective_inventory/models"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// MemberDiscountPercent is the flat membership discount applied to every order,
// separate from (and stackable with) coupon discounts.
const MemberDiscountPercent = 5.0

// MemberHandler serves the public storefront membership: register/login by
// phone, profile, order history. Error messages are Thai — shown to shoppers
// as-is (same convention as coupons).
type MemberHandler struct {
	Config *config.Config
}

func NewMemberHandler(cfg *config.Config) *MemberHandler {
	return &MemberHandler{Config: cfg}
}

// ---- Discount + eligibility (shared with the order-creation paths) ----

// computeMemberDiscount returns the membership discount on a subtotal.
func computeMemberDiscount(subtotal float64) float64 {
	return roundSatang(subtotal * MemberDiscountPercent / 100)
}

// customerHasPriorOrder reports whether the customer already has at least one
// order — the auto-membership rule for returning customers. Call BEFORE the
// new order row is created so the current purchase doesn't count itself.
func customerHasPriorOrder(tx *gorm.DB, customerID uint) (bool, error) {
	if customerID == 0 {
		return false, nil
	}
	var count int64
	err := tx.Model(&models.Order{}).Where("customer_id = ?", customerID).Count(&count).Error
	return count > 0, err
}

// ensureMembership decides whether the customer gets the member discount and,
// the first time a returning customer qualifies, promotes them to member so
// the status is visible in the admin. Runs inside the order transaction and
// must be called BEFORE tx.Create(&order).
func ensureMembership(tx *gorm.DB, customer *models.Customer) (bool, error) {
	if customer == nil || customer.ID == 0 {
		return false, nil
	}
	if customer.IsMember {
		return true, nil
	}
	prior, err := customerHasPriorOrder(tx, customer.ID)
	if err != nil || !prior {
		return false, err
	}
	now := time.Now()
	if err := tx.Model(&models.Customer{}).Where("id = ?", customer.ID).
		Updates(map[string]interface{}{"is_member": true, "member_since": now}).Error; err != nil {
		return false, err
	}
	customer.IsMember = true
	customer.MemberSince = &now
	return true, nil
}

// ---- Auth ----

const memberTokenTTL = 30 * 24 * time.Hour

// memberToken signs a JWT for a storefront member. The "role":"member" claim
// is what keeps these tokens out of the admin API (see middleware.JWTAuth).
func (h *MemberHandler) memberToken(customerID uint) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"customer_id": customerID,
		"role":        "member",
		"exp":         time.Now().Add(memberTokenTTL).Unix(),
	})
	return token.SignedString([]byte(h.Config.JWTSecret))
}

func memberProfile(customer *models.Customer) fiber.Map {
	return fiber.Map{
		"id":               customer.ID,
		"name":             customer.Name,
		"phone":            customer.Phone,
		"email":            customer.Email,
		"address":          customer.Address,
		"is_member":        customer.IsMember,
		"member_since":     customer.MemberSince,
		"discount_percent": MemberDiscountPercent,
	}
}

type memberRegisterRequest struct {
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Register creates a member account keyed by phone. If the phone already
// belongs to a customer (e.g. they ordered before), the account attaches to
// that record — their history and auto-membership carry over.
func (h *MemberHandler) Register(c *fiber.Ctx) error {
	var req memberRegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Phone = strings.TrimSpace(req.Phone)
	req.Email = strings.TrimSpace(req.Email)

	if req.Name == "" || req.Phone == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "กรุณากรอกชื่อและเบอร์โทรศัพท์"})
	}
	if len(req.Password) < 6 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "รหัสผ่านต้องมีอย่างน้อย 6 ตัวอักษร"})
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "สมัครสมาชิกไม่สำเร็จ กรุณาลองใหม่"})
	}

	var customer models.Customer
	now := time.Now()
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("phone = ?", req.Phone).First(&customer).Error; err != nil {
			if err != gorm.ErrRecordNotFound {
				return err
			}
			customer = models.Customer{
				Name:         req.Name,
				Phone:        req.Phone,
				Email:        req.Email,
				IsMember:     true,
				MemberSince:  &now,
				PasswordHash: string(hash),
			}
			return tx.Create(&customer).Error
		}
		if customer.PasswordHash != "" {
			return fiber.NewError(fiber.StatusBadRequest, "เบอร์นี้เป็นสมาชิกอยู่แล้ว กรุณาเข้าสู่ระบบ")
		}
		updates := map[string]interface{}{
			"name":          req.Name,
			"password_hash": string(hash),
			"is_member":     true,
		}
		if req.Email != "" {
			updates["email"] = req.Email
		}
		if customer.MemberSince == nil {
			updates["member_since"] = now
		}
		if err := tx.Model(&customer).Updates(updates).Error; err != nil {
			return err
		}
		return tx.First(&customer, customer.ID).Error
	})
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return c.Status(fe.Code).JSON(fiber.Map{"error": fe.Message})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "สมัครสมาชิกไม่สำเร็จ กรุณาลองใหม่"})
	}

	token, err := h.memberToken(customer.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "สมัครสมาชิกไม่สำเร็จ กรุณาลองใหม่"})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"token": token, "member": memberProfile(&customer)})
}

type memberLoginRequest struct {
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

func (h *MemberHandler) Login(c *fiber.Ctx) error {
	var req memberLoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	req.Phone = strings.TrimSpace(req.Phone)
	if req.Phone == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "กรุณากรอกเบอร์โทรศัพท์และรหัสผ่าน"})
	}

	var customer models.Customer
	if err := database.DB.Where("phone = ?", req.Phone).First(&customer).Error; err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "เบอร์โทรหรือรหัสผ่านไม่ถูกต้อง"})
	}
	if customer.PasswordHash == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "เบอร์นี้ยังไม่ได้สมัครสมาชิก กรุณาสมัครก่อน"})
	}
	if err := bcrypt.CompareHashAndPassword([]byte(customer.PasswordHash), []byte(req.Password)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "เบอร์โทรหรือรหัสผ่านไม่ถูกต้อง"})
	}

	// Logging in makes you a member even if the flag was never set (e.g.
	// registered before this feature stamped it).
	if !customer.IsMember {
		now := time.Now()
		database.DB.Model(&customer).Updates(map[string]interface{}{"is_member": true, "member_since": now})
		customer.IsMember = true
		customer.MemberSince = &now
	}

	token, err := h.memberToken(customer.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "เข้าสู่ระบบไม่สำเร็จ กรุณาลองใหม่"})
	}
	return c.JSON(fiber.Map{"token": token, "member": memberProfile(&customer)})
}

// Check tells the checkout page whether a phone number qualifies for the
// member discount (registered member OR returning customer with a prior
// order). It deliberately returns only a boolean — no name or other details —
// since anyone can call it with any phone number.
func (h *MemberHandler) Check(c *fiber.Ctx) error {
	var req struct {
		Phone string `json:"phone"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	req.Phone = strings.TrimSpace(req.Phone)

	isMember := false
	if req.Phone != "" {
		var customer models.Customer
		if err := database.DB.Where("phone = ?", req.Phone).First(&customer).Error; err == nil {
			if customer.IsMember {
				isMember = true
			} else if prior, err := customerHasPriorOrder(database.DB, customer.ID); err == nil && prior {
				isMember = true
			}
		}
	}
	return c.JSON(fiber.Map{"is_member": isMember, "discount_percent": MemberDiscountPercent})
}

// ---- Logged-in member routes (behind middleware.MemberAuth) ----

// currentMember loads the customer for the authenticated member token.
func currentMember(c *fiber.Ctx) (*models.Customer, error) {
	id, _ := c.Locals("customer_id").(uint)
	var customer models.Customer
	if err := database.DB.First(&customer, id).Error; err != nil {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "ไม่พบข้อมูลสมาชิก")
	}
	return &customer, nil
}

func (h *MemberHandler) Me(c *fiber.Ctx) error {
	customer, err := currentMember(c)
	if err != nil {
		fe := err.(*fiber.Error)
		return c.Status(fe.Code).JSON(fiber.Map{"error": fe.Message})
	}
	return c.JSON(memberProfile(customer))
}

type memberUpdateRequest struct {
	Name            string `json:"name"`
	Email           string `json:"email"`
	Address         string `json:"address"`
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// UpdateMe updates the member's contact details, and optionally the password
// when current_password + new_password are provided.
func (h *MemberHandler) UpdateMe(c *fiber.Ctx) error {
	customer, err := currentMember(c)
	if err != nil {
		fe := err.(*fiber.Error)
		return c.Status(fe.Code).JSON(fiber.Map{"error": fe.Message})
	}

	var req memberUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	updates := map[string]interface{}{}
	if name := strings.TrimSpace(req.Name); name != "" {
		updates["name"] = name
	}
	updates["email"] = strings.TrimSpace(req.Email)
	updates["address"] = strings.TrimSpace(req.Address)

	if req.NewPassword != "" {
		if len(req.NewPassword) < 6 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "รหัสผ่านใหม่ต้องมีอย่างน้อย 6 ตัวอักษร"})
		}
		if bcrypt.CompareHashAndPassword([]byte(customer.PasswordHash), []byte(req.CurrentPassword)) != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "รหัสผ่านปัจจุบันไม่ถูกต้อง"})
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "บันทึกไม่สำเร็จ กรุณาลองใหม่"})
		}
		updates["password_hash"] = string(hash)
	}

	if err := database.DB.Model(customer).Updates(updates).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "บันทึกไม่สำเร็จ กรุณาลองใหม่"})
	}
	database.DB.First(customer, customer.ID)
	return c.JSON(memberProfile(customer))
}

// MyOrders returns the member's order history, newest first.
func (h *MemberHandler) MyOrders(c *fiber.Ctx) error {
	customer, err := currentMember(c)
	if err != nil {
		fe := err.(*fiber.Error)
		return c.Status(fe.Code).JSON(fiber.Map{"error": fe.Message})
	}

	var orders []models.Order
	database.DB.Preload("Items").Preload("Items.Product").
		Where("customer_id = ?", customer.ID).
		Order("created_at DESC").Find(&orders)
	return c.JSON(orders)
}
