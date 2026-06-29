package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

// ShopHandler serves the public storefront: product browsing and customer checkout.
// These routes are intentionally unauthenticated (no JWT) so the public storefront
// can read products and place orders without an admin login.
type ShopHandler struct {
	Config   *config.Config
	Telegram *services.TelegramNotifier
}

func NewShopHandler(cfg *config.Config, telegram *services.TelegramNotifier) *ShopHandler {
	return &ShopHandler{Config: cfg, Telegram: telegram}
}

// Products lists products available to shoppers. By default only in-stock items
// are returned; pass ?include_out=1 to include sold-out products too.
func (h *ShopHandler) Products(c *fiber.Ctx) error {
	var products []models.Product

	query := database.DB
	if c.Query("include_out") != "1" {
		query = query.Where("stock > 0")
	}
	if search := c.Query("search"); search != "" {
		query = query.Where("name LIKE ? OR sku LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	query.Order("created_at DESC").Find(&products)
	return c.JSON(products)
}

// Product returns a single product for the detail page.
func (h *ShopHandler) Product(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	var product models.Product
	if err := database.DB.First(&product, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "product not found"})
	}

	return c.JSON(product)
}

// SiteImages returns the editable storefront images keyed by slot, e.g.
// {"hero": {...}, "lookbook_1": {...}}. Only slots that have an image set are
// returned; the storefront falls back to its built-in defaults for the rest.
func (h *ShopHandler) SiteImages(c *fiber.Ctx) error {
	var images []models.SiteImage
	database.DB.Where("image_url <> ''").Find(&images)

	out := make(map[string]models.SiteImage, len(images))
	for _, img := range images {
		out[img.Key] = img
	}
	return c.JSON(out)
}

// ShopCheckoutRequest is the public order payload posted from the storefront.
type ShopCheckoutRequest struct {
	Name    string                   `json:"name"`
	Phone   string                   `json:"phone"`
	Email   string                   `json:"email"`
	Address string                   `json:"address"`
	Notes   string                   `json:"notes"`
	Items   []models.CreateOrderItem `json:"items"`
}

// Checkout creates an order from the public storefront. It finds or creates a
// customer (matched by phone), then deducts stock atomically — mirroring the
// admin order-creation path so inventory stays consistent across both entry points.
func (h *ShopHandler) Checkout(c *fiber.Ctx) error {
	var req ShopCheckoutRequest

	// The storefront posts multipart/form-data so it can attach the payment slip.
	// A JSON body is still accepted (e.g. for API clients) but then has no slip.
	isMultipart := strings.HasPrefix(c.Get("Content-Type"), "multipart/form-data")
	if isMultipart {
		req.Name = c.FormValue("name")
		req.Phone = c.FormValue("phone")
		req.Email = c.FormValue("email")
		req.Address = c.FormValue("address")
		req.Notes = c.FormValue("notes")
		if itemsJSON := c.FormValue("items"); itemsJSON != "" {
			json.Unmarshal([]byte(itemsJSON), &req.Items)
		}
	} else {
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Phone = strings.TrimSpace(req.Phone)
	req.Address = strings.TrimSpace(req.Address)

	if req.Name == "" || req.Phone == "" || req.Address == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name, phone and address are required"})
	}
	if len(req.Items) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "items are required"})
	}

	// Payment slip is required for storefront (multipart) checkout.
	slipFile, slipErr := c.FormFile("slip")
	if isMultipart && (slipErr != nil || slipFile == nil) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "payment slip is required"})
	}

	// Save the slip up front with a temporary name; we rename it with the real
	// order ID once the order is created (mirrors the admin order handler).
	var slipFilename string
	if slipFile != nil {
		ext := filepath.Ext(slipFile.Filename)
		slipFilename = fmt.Sprintf("slip_new_%d%s", time.Now().UnixNano(), ext)
		if err := c.SaveFile(slipFile, filepath.Join(h.Config.UploadDir, slipFilename)); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save slip"})
		}
	}

	var order models.Order
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// Find or create the customer by phone.
		var customer models.Customer
		if err := tx.Where("phone = ?", req.Phone).First(&customer).Error; err != nil {
			if err != gorm.ErrRecordNotFound {
				return err
			}
			customer = models.Customer{
				Name:    req.Name,
				Phone:   req.Phone,
				Email:   req.Email,
				Address: req.Address,
			}
			if err := tx.Create(&customer).Error; err != nil {
				return err
			}
		} else {
			// Keep the customer's contact details fresh from the latest order.
			tx.Model(&customer).Updates(models.Customer{
				Name:    req.Name,
				Email:   req.Email,
				Address: req.Address,
			})
		}

		var totalAmount float64
		var items []models.OrderItem

		for _, item := range req.Items {
			if item.Quantity <= 0 {
				return fiber.NewError(fiber.StatusBadRequest, "quantity must be positive")
			}

			var product models.Product
			if err := tx.First(&product, item.ProductID).Error; err != nil {
				return fiber.NewError(fiber.StatusBadRequest, "product not found")
			}

			if product.Stock < item.Quantity {
				return fiber.NewError(fiber.StatusBadRequest,
					"insufficient stock for "+product.Name)
			}

			tx.Model(&product).Update("stock", product.Stock-item.Quantity)

			totalAmount += product.Price * float64(item.Quantity)
			items = append(items, models.OrderItem{
				ProductID: item.ProductID,
				Quantity:  item.Quantity,
				Price:     product.Price,
			})
		}

		order = models.Order{
			CustomerID:  customer.ID,
			Status:      models.StatusPending,
			TotalAmount: totalAmount,
			Notes:       req.Notes,
			SlipImage:   slipFilename,
			Items:       items,
		}
		return tx.Create(&order).Error
	})

	if err != nil {
		// Order failed — discard the orphaned slip file we saved earlier.
		if slipFilename != "" {
			os.Remove(filepath.Join(h.Config.UploadDir, slipFilename))
		}
		if fe, ok := err.(*fiber.Error); ok {
			return c.Status(fe.Code).JSON(fiber.Map{"error": fe.Message})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create order"})
	}

	// Rename the slip with the real order ID now that we have one.
	if slipFilename != "" {
		newFilename := fmt.Sprintf("slip_%d_%d%s", order.ID, time.Now().Unix(), filepath.Ext(slipFilename))
		oldPath := filepath.Join(h.Config.UploadDir, slipFilename)
		newPath := filepath.Join(h.Config.UploadDir, newFilename)
		if err := os.Rename(oldPath, newPath); err == nil {
			database.DB.Model(&order).Update("slip_image", newFilename)
			order.SlipImage = newFilename
		}
	}

	// Reload with relations for the response + notification.
	database.DB.Preload("Customer").Preload("Items").Preload("Items.Product").First(&order, order.ID)

	h.Telegram.NotifyNewOrder(&order)

	return c.Status(fiber.StatusCreated).JSON(order)
}
