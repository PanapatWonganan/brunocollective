package handlers

// Order-from-chat + public payment page.
//
// The admin creates an order for the customer linked to a chat conversation
// (POST /api/chats/:id/order) and gets back a payment URL to push into the
// chat. The customer opens /pay/{token} on the storefront, sees the order
// summary, and uploads a payment slip — no login involved; the unguessable
// token is the only credential, so the page never exposes phone/address.

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"brunocollective_inventory/database"
	"brunocollective_inventory/models"
	"brunocollective_inventory/services"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// newPaymentToken returns a URL-safe random token (18 bytes → 24 chars).
func newPaymentToken() (string, error) {
	buf := make([]byte, 18)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func (h *OrderHandler) payURL(token string) string {
	return strings.TrimRight(h.Config.BaseURL, "/") + "/pay/" + token
}

// ChatOrderRequest is the JSON body for creating an order from a chat
// conversation. The customer comes from the conversation's linked customer
// and the channel is stamped from the conversation's platform.
type ChatOrderRequest struct {
	Notes      string                   `json:"notes"`
	CouponCode string                   `json:"coupon_code"`
	Items      []models.CreateOrderItem `json:"items"`
}

// CreateFromChat handles POST /api/chats/:id/order (admin). Same pricing
// pipeline as the other creation paths (stock deduction, membership, coupon)
// plus a payment token so the link can be sent into the chat.
func (h *OrderHandler) CreateFromChat(c *fiber.Ctx) error {
	convID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	var conv models.Conversation
	if err := database.DB.First(&conv, convID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "conversation not found"})
	}
	if conv.CustomerID == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ยังไม่ได้ผูกลูกค้ากับแชทนี้ — ผูกลูกค้าก่อนสร้างออเดอร์"})
	}

	var req ChatOrderRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if len(req.Items) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "items are required"})
	}

	var customer models.Customer
	if err := database.DB.First(&customer, *conv.CustomerID).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "customer not found"})
	}

	token, err := newPaymentToken()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate payment token"})
	}

	var order models.Order
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		var totalAmount float64
		var items []models.OrderItem

		for _, item := range req.Items {
			orderItem, price, err := buildOrderItem(tx, item, nil)
			if err != nil {
				return err
			}
			totalAmount += price * float64(item.Quantity)
			items = append(items, orderItem)
		}

		// Membership discount (5%) — same rule as the other creation paths.
		isMember, err := ensureMembership(tx, &customer)
		if err != nil {
			return err
		}
		var memberDiscount float64
		if isMember {
			memberDiscount = computeMemberDiscount(totalAmount)
		}

		order = models.Order{
			CustomerID:     customer.ID,
			Status:         models.StatusPending,
			Subtotal:       totalAmount,
			MemberDiscount: memberDiscount,
			TotalAmount:    roundSatang(totalAmount - memberDiscount),
			Notes:          req.Notes,
			Channel:        conv.Platform,
			ConversationID: &conv.ID,
			PaymentToken:   token,
			Items:          items,
		}

		if err := tx.Create(&order).Error; err != nil {
			return err
		}
		return applyCouponToOrder(tx, &order, req.CouponCode, totalAmount)
	})
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	database.DB.Preload("Customer").Preload("Items").Preload("Items.Product").First(&order, order.ID)
	h.Telegram.NotifyNewOrder(&order)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"order":   order,
		"pay_url": h.payURL(token),
	})
}

func findOrderByPayToken(token string) (*models.Order, bool) {
	if token == "" {
		return nil, false
	}
	var order models.Order
	if err := database.DB.
		Preload("Customer").Preload("Items").Preload("Items.Product").
		Where("payment_token = ?", token).First(&order).Error; err != nil {
		return nil, false
	}
	return &order, true
}

// payOrderJSON is the customer-facing view of an order — items and amounts
// only, no phone/address/internal notes.
func payOrderJSON(order *models.Order) fiber.Map {
	items := make([]fiber.Map, 0, len(order.Items))
	for _, it := range order.Items {
		items = append(items, fiber.Map{
			"name":     it.Product.Name,
			"size":     it.Size,
			"color":    it.Color,
			"quantity": it.Quantity,
			"price":    it.Price,
		})
	}
	// Legacy orders have Subtotal 0 — fall back to the total.
	subtotal := order.Subtotal
	if subtotal == 0 {
		subtotal = order.TotalAmount
	}
	return fiber.Map{
		"order_no":        order.ID,
		"status":          order.Status,
		"created_at":      order.CreatedAt,
		"customer_name":   order.Customer.Name,
		"items":           items,
		"subtotal":        subtotal,
		"member_discount": order.MemberDiscount,
		"discount_amount": order.DiscountAmount,
		"coupon_code":     order.CouponCode,
		"total_amount":    order.TotalAmount,
		"has_slip":        order.SlipImage != "",
	}
}

// PayGet handles GET /api/pay/:token (public).
func (h *OrderHandler) PayGet(c *fiber.Ctx) error {
	order, ok := findOrderByPayToken(c.Params("token"))
	if !ok {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "ไม่พบคำสั่งซื้อ"})
	}
	return c.JSON(payOrderJSON(order))
}

// ChatFollowups returns conversations that have unpaid chat-created orders
// (pending, no slip yet) — the "ดีลค้าง" tab in ChatView. Ordered oldest
// unpaid order first so the queue works top-down. GET /api/chats/followups.
func (h *OrderHandler) ChatFollowups(c *fiber.Ctx) error {
	var orders []models.Order
	database.DB.
		Where("conversation_id IS NOT NULL AND status = ? AND (slip_image = '' OR slip_image IS NULL)",
			models.StatusPending).
		Order("created_at ASC").
		Find(&orders)
	if len(orders) == 0 {
		return c.JSON([]fiber.Map{})
	}

	convIDs := make([]uint, 0, len(orders))
	byConv := make(map[uint][]models.Order)
	for _, o := range orders {
		id := *o.ConversationID
		if _, seen := byConv[id]; !seen {
			convIDs = append(convIDs, id)
		}
		byConv[id] = append(byConv[id], o)
	}

	var convs []models.Conversation
	database.DB.Preload("Customer").Where("id IN ?", convIDs).Find(&convs)
	convByID := make(map[uint]models.Conversation, len(convs))
	for _, cv := range convs {
		convByID[cv.ID] = cv
	}

	out := make([]fiber.Map, 0, len(convIDs))
	for _, id := range convIDs {
		conv, ok := convByID[id]
		if !ok {
			continue // conversation was deleted
		}
		list := make([]fiber.Map, 0, len(byConv[id]))
		for _, o := range byConv[id] {
			list = append(list, fiber.Map{
				"id":           o.ID,
				"total_amount": o.TotalAmount,
				"coupon_code":  o.CouponCode,
				"created_at":   o.CreatedAt,
				"pay_url":      h.payURL(o.PaymentToken),
			})
		}
		out = append(out, fiber.Map{"conversation": conv, "orders": list})
	}
	return c.JSON(out)
}

type followupDiscountRequest struct {
	Type         string  `json:"type"` // "percent" | "fixed"
	Value        float64 `json:"value"`
	ExpiresHours int     `json:"expires_hours"` // urgency window told to the customer
}

// FollowupDiscount sweetens an existing unpaid order to close a stale deal:
// creates a single-use coupon and applies it to the order in one transaction,
// so the /pay page shows the new total immediately — the customer never has
// to type a code. POST /api/orders/:id/followup-discount.
func (h *OrderHandler) FollowupDiscount(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	var order models.Order
	if err := database.DB.First(&order, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "order not found"})
	}
	if order.Status != models.StatusPending {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ให้ส่วนลดได้เฉพาะออเดอร์ที่ยังรอชำระ"})
	}
	if order.CouponID != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ออเดอร์นี้มีส่วนลดคูปองอยู่แล้ว"})
	}

	var req followupDiscountRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if req.Type != string(models.CouponPercent) && req.Type != string(models.CouponFixed) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "type ต้องเป็น percent หรือ fixed"})
	}
	if req.Value <= 0 || (req.Type == string(models.CouponPercent) && req.Value >= 100) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "มูลค่าส่วนลดไม่ถูกต้อง"})
	}
	hours := req.ExpiresHours
	if hours <= 0 {
		hours = 24
	}
	if hours > 168 {
		hours = 168
	}

	subtotal := order.Subtotal
	if subtotal == 0 {
		subtotal = order.TotalAmount
	}
	code := fmt.Sprintf("CHAT%d-%s", order.ID, randFollowupCode())
	expires := time.Now().Add(time.Duration(hours) * time.Hour)

	err = database.DB.Transaction(func(tx *gorm.DB) error {
		coupon := models.Coupon{
			Code:                  code,
			Name:                  fmt.Sprintf("ตามดีลออเดอร์ #%d", order.ID),
			Type:                  models.CouponType(req.Type),
			Value:                 req.Value,
			ExpiresAt:             &expires,
			UsageLimit:            1,
			UsageLimitPerCustomer: 1,
			IsActive:              true,
		}
		if err := tx.Create(&coupon).Error; err != nil {
			return err
		}
		return applyCouponToOrder(tx, &order, code, subtotal)
	})
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	database.DB.Preload("Customer").Preload("Items").Preload("Items.Product").First(&order, order.ID)
	return c.JSON(fiber.Map{
		"order":      order,
		"pay_url":    h.payURL(order.PaymentToken),
		"expires_at": expires,
	})
}

// randFollowupCode returns 4 chars from an unambiguous alphabet (no 0/O/1/I/L).
func randFollowupCode() string {
	const alphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	buf := make([]byte, 4)
	rand.Read(buf)
	for i, b := range buf {
		buf[i] = alphabet[int(b)%len(alphabet)]
	}
	return string(buf)
}

// PayUploadSlip handles POST /api/pay/:token/slip (public, multipart).
// Mirrors the admin UploadSlip but is token-scoped; re-upload is allowed so
// the customer can fix a wrong image while the order is still pending.
func (h *OrderHandler) PayUploadSlip(c *fiber.Ctx) error {
	order, ok := findOrderByPayToken(c.Params("token"))
	if !ok {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "ไม่พบคำสั่งซื้อ"})
	}
	if order.Status == models.StatusCancelled {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ออเดอร์นี้ถูกยกเลิกแล้ว กรุณาติดต่อร้าน"})
	}

	file, err := c.FormFile("slip")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "กรุณาแนบสลิปการโอนเงิน"})
	}
	if !strings.HasPrefix(file.Header.Get("Content-Type"), "image/") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ไฟล์สลิปต้องเป็นรูปภาพ"})
	}

	base := fmt.Sprintf("slip_%d_%d", order.ID, time.Now().Unix())
	filename, err := services.SaveOptimizedImage(file, h.Config.UploadDir, base, services.MaxDimProduct)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "บันทึกสลิปไม่สำเร็จ กรุณาลองใหม่"})
	}

	database.DB.Model(order).Update("slip_image", filename)
	order.SlipImage = filename

	h.Telegram.NotifySlipUploaded(order)

	return c.JSON(payOrderJSON(order))
}
