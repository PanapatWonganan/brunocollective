package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"brunocollective_inventory/config"
	"brunocollective_inventory/database"
	"brunocollective_inventory/models"
	"brunocollective_inventory/services"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// SalePageHandler manages ClickFunnels-style sale/landing pages: admin CRUD
// for the builder, plus the public endpoints the storefront uses to render
// /s/{slug} and place orders. All pricing (offer price, bump price) is
// resolved server-side from the page config — never from the client.
type SalePageHandler struct {
	Config   *config.Config
	Telegram *services.TelegramNotifier
}

func NewSalePageHandler(cfg *config.Config, telegram *services.TelegramNotifier) *SalePageHandler {
	return &SalePageHandler{Config: cfg, Telegram: telegram}
}

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// normalizeSlug lowercases and converts spaces/underscores to dashes so the
// admin can type a loose value; validation still rejects anything else.
func normalizeSlug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.NewReplacer(" ", "-", "_", "-").Replace(s)
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return strings.Trim(s, "-")
}

type salePageRequest struct {
	Slug            string                  `json:"slug"`
	Title           string                  `json:"title"`
	Status          string                  `json:"status"`
	ProductID       uint                    `json:"product_id"`
	OfferPrice      *float64                `json:"offer_price"`
	Sections        models.SalePageSections `json:"sections"`
	BumpEnabled     bool                    `json:"bump_enabled"`
	BumpProductID   *uint                   `json:"bump_product_id"`
	BumpPrice       float64                 `json:"bump_price"`
	BumpHeadline    string                  `json:"bump_headline"`
	BumpDescription string                  `json:"bump_description"`
	CountdownEndsAt *time.Time              `json:"countdown_ends_at"`
	ShowStock       bool                    `json:"show_stock"`
	AllowCoupon     bool                    `json:"allow_coupon"`
}

func (r *salePageRequest) validate() error {
	r.Slug = normalizeSlug(r.Slug)
	r.Title = strings.TrimSpace(r.Title)
	if r.Slug == "" || !slugPattern.MatchString(r.Slug) {
		return fiber.NewError(fiber.StatusBadRequest, "slug must contain only lowercase letters, numbers and dashes")
	}
	if r.Title == "" {
		return fiber.NewError(fiber.StatusBadRequest, "title is required")
	}
	if r.ProductID == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "product_id is required")
	}
	if r.OfferPrice != nil && *r.OfferPrice < 0 {
		return fiber.NewError(fiber.StatusBadRequest, "offer_price must not be negative")
	}
	if r.Status != string(models.SalePageDraft) && r.Status != string(models.SalePagePublished) {
		r.Status = string(models.SalePageDraft)
	}
	if r.BumpEnabled {
		if r.BumpProductID == nil || *r.BumpProductID == 0 {
			return fiber.NewError(fiber.StatusBadRequest, "bump product is required when the order bump is enabled")
		}
		if r.BumpPrice <= 0 {
			return fiber.NewError(fiber.StatusBadRequest, "bump price must be greater than 0")
		}
	}
	return nil
}

func (r *salePageRequest) apply(page *models.SalePage) {
	page.Slug = r.Slug
	page.Title = r.Title
	page.Status = models.SalePageStatus(r.Status)
	page.ProductID = r.ProductID
	page.OfferPrice = r.OfferPrice
	page.Sections = r.Sections
	page.BumpEnabled = r.BumpEnabled
	page.BumpProductID = r.BumpProductID
	page.BumpPrice = r.BumpPrice
	page.BumpHeadline = strings.TrimSpace(r.BumpHeadline)
	page.BumpDescription = strings.TrimSpace(r.BumpDescription)
	page.CountdownEndsAt = r.CountdownEndsAt
	page.ShowStock = r.ShowStock
	page.AllowCoupon = r.AllowCoupon
}

// ---- Admin ----

func (h *SalePageHandler) List(c *fiber.Ctx) error {
	var pages []models.SalePage
	database.DB.Preload("Product").Order("created_at DESC").Find(&pages)
	return c.JSON(pages)
}

func (h *SalePageHandler) Get(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var page models.SalePage
	if err := database.DB.Preload("Product").Preload("BumpProduct").First(&page, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "sale page not found"})
	}
	return c.JSON(page)
}

func (h *SalePageHandler) Create(c *fiber.Ctx) error {
	var req salePageRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if err := req.validate(); err != nil {
		return couponHTTPError(c, err)
	}

	var count int64
	database.DB.Model(&models.SalePage{}).Where("slug = ?", req.Slug).Count(&count)
	if count > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "slug already exists"})
	}

	var page models.SalePage
	req.apply(&page)
	if err := database.DB.Create(&page).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create sale page"})
	}
	database.DB.Preload("Product").Preload("BumpProduct").First(&page, page.ID)
	return c.Status(fiber.StatusCreated).JSON(page)
}

func (h *SalePageHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var page models.SalePage
	if err := database.DB.First(&page, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "sale page not found"})
	}

	var req salePageRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if err := req.validate(); err != nil {
		return couponHTTPError(c, err)
	}

	var count int64
	database.DB.Model(&models.SalePage{}).Where("slug = ? AND id <> ?", req.Slug, page.ID).Count(&count)
	if count > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "slug already exists"})
	}

	req.apply(&page)
	// Save with Select so booleans/nils (bump off, countdown cleared) persist.
	if err := database.DB.Model(&page).Select(
		"slug", "title", "status", "product_id", "offer_price", "sections",
		"bump_enabled", "bump_product_id", "bump_price", "bump_headline", "bump_description",
		"countdown_ends_at", "show_stock", "allow_coupon",
	).Updates(&page).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update sale page"})
	}
	database.DB.Preload("Product").Preload("BumpProduct").First(&page, id)
	return c.JSON(page)
}

func (h *SalePageHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	if err := database.DB.Delete(&models.SalePage{}, id).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete sale page"})
	}
	return c.JSON(fiber.Map{"message": "sale page deleted"})
}

// Duplicate clones a page as a draft with a "-copy" slug — the cheap way to
// spin up an A/B variant or reuse a past campaign's layout.
func (h *SalePageHandler) Duplicate(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var page models.SalePage
	if err := database.DB.First(&page, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "sale page not found"})
	}

	copy := page
	copy.ID = 0
	copy.Status = models.SalePageDraft
	copy.Views = 0
	copy.OrdersCount = 0
	copy.BumpCount = 0
	copy.CreatedAt = time.Time{}
	copy.UpdatedAt = time.Time{}

	// Find a free slug: base-copy, base-copy-2, base-copy-3, …
	base := page.Slug + "-copy"
	slug := base
	for i := 2; ; i++ {
		var count int64
		database.DB.Model(&models.SalePage{}).Where("slug = ?", slug).Count(&count)
		if count == 0 {
			break
		}
		slug = fmt.Sprintf("%s-%d", base, i)
	}
	copy.Slug = slug
	copy.Title = page.Title + " (copy)"

	if err := database.DB.Create(&copy).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to duplicate sale page"})
	}
	database.DB.Preload("Product").First(&copy, copy.ID)
	return c.Status(fiber.StatusCreated).JSON(copy)
}

// TogglePublish flips draft <-> published.
func (h *SalePageHandler) TogglePublish(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var page models.SalePage
	if err := database.DB.First(&page, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "sale page not found"})
	}
	next := models.SalePagePublished
	if page.Status == models.SalePagePublished {
		next = models.SalePageDraft
	}
	database.DB.Model(&page).Update("status", next)
	database.DB.Preload("Product").First(&page, id)
	return c.JSON(page)
}

// UploadImage stores one image for use in sale page sections and returns its
// public URL. Reuses the slip upload directory/static route.
func (h *SalePageHandler) UploadImage(c *fiber.Ctx) error {
	file, err := c.FormFile("image")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "image file is required"})
	}
	ext := filepath.Ext(file.Filename)
	filename := fmt.Sprintf("salepage_%d%s", time.Now().UnixNano(), ext)
	if err := c.SaveFile(file, filepath.Join(h.Config.UploadDir, filename)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save image"})
	}
	return c.JSON(fiber.Map{"url": "/uploads/" + filename})
}

// ---- Public (storefront) ----

// PublicGet returns a page for rendering at /s/{slug}. Published pages only,
// unless ?preview=1 (used by the admin's preview link, which also skips the
// view counter so previews don't pollute conversion stats).
func (h *SalePageHandler) PublicGet(c *fiber.Ctx) error {
	slug := normalizeSlug(c.Params("slug"))
	preview := c.Query("preview") == "1"

	var page models.SalePage
	query := database.DB.Preload("Product").Preload("Product.Variants").
		Preload("BumpProduct").Preload("BumpProduct.Variants")
	if err := query.Where("slug = ?", slug).First(&page).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "page not found"})
	}
	if page.Status != models.SalePagePublished && !preview {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "page not found"})
	}

	if !preview {
		database.DB.Model(&page).UpdateColumn("views", gorm.Expr("views + 1"))
	}

	page.Product.ComputeTotalStock()
	if page.BumpProduct != nil {
		page.BumpProduct.ComputeTotalStock()
	}
	return c.JSON(page)
}

// salePageOrderRequest is the order form payload from /s/{slug}. Quantity
// applies to the main offer; the bump is always quantity 1.
type salePageOrderRequest struct {
	Name          string
	Phone         string
	Email         string
	Address       string
	Notes         string
	Quantity      int
	VariantID     *uint
	Bump          bool
	BumpVariantID *uint
	CouponCode    string
}

// PublicOrder places an order from a sale page. Mirrors the storefront
// checkout (multipart with required slip, find-or-create customer by phone)
// but prices the items from the page config and stamps the funnel stats.
// Error messages are Thai — they are shown to shoppers as-is.
func (h *SalePageHandler) PublicOrder(c *fiber.Ctx) error {
	slug := normalizeSlug(c.Params("slug"))

	var page models.SalePage
	if err := database.DB.Where("slug = ? AND status = ?", slug, models.SalePagePublished).
		First(&page).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "ไม่พบหน้าข้อเสนอนี้"})
	}
	// Honest scarcity: a countdown is a real deadline, not decoration.
	if page.CountdownEndsAt != nil && time.Now().After(*page.CountdownEndsAt) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ข้อเสนอนี้สิ้นสุดแล้ว"})
	}

	var req salePageOrderRequest
	req.Name = strings.TrimSpace(c.FormValue("name"))
	req.Phone = strings.TrimSpace(c.FormValue("phone"))
	req.Email = strings.TrimSpace(c.FormValue("email"))
	req.Address = strings.TrimSpace(c.FormValue("address"))
	req.Notes = c.FormValue("notes")
	req.CouponCode = c.FormValue("coupon_code")
	req.Quantity, _ = strconv.Atoi(c.FormValue("quantity"))
	if req.Quantity <= 0 {
		req.Quantity = 1
	}
	if v, err := strconv.Atoi(c.FormValue("variant_id")); err == nil && v > 0 {
		id := uint(v)
		req.VariantID = &id
	}
	req.Bump = c.FormValue("bump") == "1"
	if v, err := strconv.Atoi(c.FormValue("bump_variant_id")); err == nil && v > 0 {
		id := uint(v)
		req.BumpVariantID = &id
	}

	if req.Name == "" || req.Phone == "" || req.Address == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "กรุณากรอกชื่อ เบอร์โทร และที่อยู่จัดส่ง"})
	}

	slipFile, slipErr := c.FormFile("slip")
	if slipErr != nil || slipFile == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "กรุณาแนบสลิปการโอนเงิน"})
	}
	slipFilename := fmt.Sprintf("slip_new_%d%s", time.Now().UnixNano(), filepath.Ext(slipFile.Filename))
	if err := c.SaveFile(slipFile, filepath.Join(h.Config.UploadDir, slipFilename)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "บันทึกสลิปไม่สำเร็จ กรุณาลองใหม่"})
	}

	var order models.Order
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		customer, err := findOrCreateCustomerByPhone(tx, req.Name, req.Phone, req.Email, req.Address)
		if err != nil {
			return err
		}

		var totalAmount float64
		var items []models.OrderItem

		// Main offer, priced from the page (nil OfferPrice = catalog price).
		mainItem, price, err := buildOrderItem(tx, models.CreateOrderItem{
			ProductID: page.ProductID,
			VariantID: req.VariantID,
			Quantity:  req.Quantity,
		}, page.OfferPrice)
		if err != nil {
			return err
		}
		totalAmount += price * float64(req.Quantity)
		items = append(items, mainItem)

		// Order bump — always quantity 1, at the page's bump price.
		bumpTaken := req.Bump && page.BumpEnabled && page.BumpProductID != nil
		if bumpTaken {
			bumpItem, bumpPrice, err := buildOrderItem(tx, models.CreateOrderItem{
				ProductID: *page.BumpProductID,
				VariantID: req.BumpVariantID,
				Quantity:  1,
			}, &page.BumpPrice)
			if err != nil {
				return err
			}
			totalAmount += bumpPrice
			items = append(items, bumpItem)
		}

		order = models.Order{
			CustomerID:  customer.ID,
			Status:      models.StatusPending,
			Subtotal:    totalAmount,
			TotalAmount: totalAmount,
			Notes:       req.Notes,
			SlipImage:   slipFilename,
			SalePageID:  &page.ID,
			Items:       items,
		}
		if err := tx.Create(&order).Error; err != nil {
			return err
		}

		if page.AllowCoupon {
			if err := applyCouponToOrder(tx, &order, req.CouponCode, totalAmount); err != nil {
				return err
			}
		}

		// Funnel stats — same transaction so a failed order counts nothing.
		updates := map[string]interface{}{"orders_count": gorm.Expr("orders_count + 1")}
		if bumpTaken {
			updates["bump_count"] = gorm.Expr("bump_count + 1")
		}
		return tx.Model(&models.SalePage{}).Where("id = ?", page.ID).Updates(updates).Error
	})

	if err != nil {
		os.Remove(filepath.Join(h.Config.UploadDir, slipFilename))
		if fe, ok := err.(*fiber.Error); ok {
			return c.Status(fe.Code).JSON(fiber.Map{"error": fe.Message})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "สั่งซื้อไม่สำเร็จ กรุณาลองใหม่"})
	}

	// Rename the slip with the real order ID (mirrors the other order paths).
	newFilename := fmt.Sprintf("slip_%d_%d%s", order.ID, time.Now().Unix(), filepath.Ext(slipFilename))
	if err := os.Rename(
		filepath.Join(h.Config.UploadDir, slipFilename),
		filepath.Join(h.Config.UploadDir, newFilename),
	); err == nil {
		database.DB.Model(&order).Update("slip_image", newFilename)
		order.SlipImage = newFilename
	}

	database.DB.Preload("Customer").Preload("Items").Preload("Items.Product").First(&order, order.ID)

	h.Telegram.NotifyNewOrder(&order)

	return c.Status(fiber.StatusCreated).JSON(order)
}
