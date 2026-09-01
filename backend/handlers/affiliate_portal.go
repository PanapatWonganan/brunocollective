package handlers

import (
	"strings"
	"time"

	"brunocollective_inventory/database"
	"brunocollective_inventory/models"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// Affiliate portal — the referrer's own storefront login (mirrors the member
// portal). Error messages are Thai, shown as-is.

const affiliateTokenTTL = 30 * 24 * time.Hour

// affiliateToken signs a JWT for the affiliate portal. role:"affiliate" keeps
// these tokens out of the admin API and the member API alike.
func (h *AffiliateHandler) affiliateToken(affiliateID uint) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"affiliate_id": affiliateID,
		"role":         "affiliate",
		"exp":          time.Now().Add(affiliateTokenTTL).Unix(),
	})
	return token.SignedString([]byte(h.Config.JWTSecret))
}

func affiliateProfile(aff *models.Affiliate) fiber.Map {
	return fiber.Map{
		"id":                 aff.ID,
		"code":               aff.Code,
		"name":               aff.Name,
		"phone":              aff.Phone,
		"email":              aff.Email,
		"commission_percent": aff.CommissionPercent,
	}
}

type affiliateLoginRequest struct {
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

// Login authenticates an affiliate by phone + password.
func (h *AffiliateHandler) Login(c *fiber.Ctx) error {
	var req affiliateLoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	req.Phone = strings.TrimSpace(req.Phone)
	if req.Phone == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "กรุณากรอกเบอร์โทรและรหัสผ่าน"})
	}

	var aff models.Affiliate
	if err := database.DB.Where("phone = ?", req.Phone).First(&aff).Error; err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "เบอร์โทรหรือรหัสผ่านไม่ถูกต้อง"})
	}
	if bcrypt.CompareHashAndPassword([]byte(aff.PasswordHash), []byte(req.Password)) != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "เบอร์โทรหรือรหัสผ่านไม่ถูกต้อง"})
	}
	if !aff.IsActive {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "บัญชีนี้ถูกปิดใช้งาน กรุณาติดต่อร้าน"})
	}

	token, err := h.affiliateToken(aff.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "เข้าสู่ระบบไม่สำเร็จ กรุณาลองใหม่"})
	}
	return c.JSON(fiber.Map{"token": token, "affiliate": affiliateProfile(&aff)})
}

func (h *AffiliateHandler) currentAffiliate(c *fiber.Ctx) (*models.Affiliate, error) {
	id, _ := c.Locals("affiliate_id").(uint)
	if id == 0 {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "invalid affiliate token")
	}
	var aff models.Affiliate
	if err := database.DB.First(&aff, id).Error; err != nil {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "invalid affiliate token")
	}
	return &aff, nil
}

// Me returns the affiliate's profile + real-time stats.
func (h *AffiliateHandler) Me(c *fiber.Ctx) error {
	aff, err := h.currentAffiliate(c)
	if err != nil {
		return affiliateHTTPError(c, err)
	}
	stats, err := loadAffiliateStats(aff.ID)
	if err != nil {
		return affiliateHTTPError(c, err)
	}
	out := affiliateWithStats(aff, stats[aff.ID])
	out["affiliate"] = affiliateProfile(aff)
	out["clicks"] = aff.ClickCount
	return c.JSON(out)
}

// MyOrders lists the affiliate's attributed orders — deliberately WITHOUT any
// customer PII (no names, phones or addresses).
func (h *AffiliateHandler) MyOrders(c *fiber.Ctx) error {
	aff, err := h.currentAffiliate(c)
	if err != nil {
		return affiliateHTTPError(c, err)
	}
	rows, err := affiliateOrderRows(aff.ID, false, 100)
	if err != nil {
		return affiliateHTTPError(c, err)
	}
	return c.JSON(rows)
}

type affiliateUpdateMeRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// UpdateMe lets the affiliate change their own password.
func (h *AffiliateHandler) UpdateMe(c *fiber.Ctx) error {
	aff, err := h.currentAffiliate(c)
	if err != nil {
		return affiliateHTTPError(c, err)
	}
	var req affiliateUpdateMeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if len(req.NewPassword) < 6 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "รหัสผ่านใหม่ต้องมีอย่างน้อย 6 ตัวอักษร"})
	}
	if bcrypt.CompareHashAndPassword([]byte(aff.PasswordHash), []byte(req.CurrentPassword)) != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "รหัสผ่านปัจจุบันไม่ถูกต้อง"})
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "บันทึกไม่สำเร็จ กรุณาลองใหม่"})
	}
	if err := database.DB.Model(aff).Update("password_hash", string(hash)).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "บันทึกไม่สำเร็จ กรุณาลองใหม่"})
	}
	return c.JSON(fiber.Map{"message": "เปลี่ยนรหัสผ่านเรียบร้อย"})
}
