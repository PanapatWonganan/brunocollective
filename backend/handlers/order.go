package handlers

import (
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	"brunocollective_inventory/config"
	"brunocollective_inventory/database"
	"brunocollective_inventory/models"
	"brunocollective_inventory/services"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type OrderHandler struct {
	Config *config.Config
	Line   *services.LineNotifier
}

func NewOrderHandler(cfg *config.Config, line *services.LineNotifier) *OrderHandler {
	return &OrderHandler{Config: cfg, Line: line}
}

func (h *OrderHandler) List(c *fiber.Ctx) error {
	var orders []models.Order

	query := database.DB.Preload("Customer").Preload("Items").Preload("Items.Product")

	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	query.Order("created_at DESC").Find(&orders)
	return c.JSON(orders)
}

func (h *OrderHandler) Get(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	var order models.Order
	if err := database.DB.
		Preload("Customer").
		Preload("Items").
		Preload("Items.Product").
		First(&order, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "order not found"})
	}

	return c.JSON(order)
}

func (h *OrderHandler) Create(c *fiber.Ctx) error {
	var req models.CreateOrderRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.CustomerID == 0 || len(req.Items) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "customer_id and items are required"})
	}

	// Verify customer exists
	var customer models.Customer
	if err := database.DB.First(&customer, req.CustomerID).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "customer not found"})
	}

	// Build order within a transaction
	var order models.Order
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var totalAmount float64
		var items []models.OrderItem

		for _, item := range req.Items {
			var product models.Product
			if err := tx.First(&product, item.ProductID).Error; err != nil {
				return fmt.Errorf("product %d not found", item.ProductID)
			}

			if product.Stock < item.Quantity {
				return fmt.Errorf("insufficient stock for %s (available: %d, requested: %d)",
					product.Name, product.Stock, item.Quantity)
			}

			// Deduct stock
			tx.Model(&product).Update("stock", product.Stock-item.Quantity)

			lineTotal := product.Price * float64(item.Quantity)
			totalAmount += lineTotal

			items = append(items, models.OrderItem{
				ProductID: item.ProductID,
				Quantity:  item.Quantity,
				Price:     product.Price,
			})
		}

		order = models.Order{
			CustomerID:  req.CustomerID,
			Status:      models.StatusPending,
			TotalAmount: totalAmount,
			Notes:       req.Notes,
			Items:       items,
		}

		return tx.Create(&order).Error
	})

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	// Reload with relations
	database.DB.Preload("Customer").Preload("Items").Preload("Items.Product").First(&order, order.ID)

	h.Line.NotifyNewOrder(&order)

	return c.Status(fiber.StatusCreated).JSON(order)
}

func (h *OrderHandler) UpdateStatus(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	var body struct {
		Status models.OrderStatus `json:"status"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	var order models.Order
	if err := database.DB.First(&order, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "order not found"})
	}

	database.DB.Model(&order).Update("status", body.Status)
	database.DB.Preload("Customer").Preload("Items").Preload("Items.Product").First(&order, id)

	h.Line.NotifyStatusChange(&order, body.Status)

	return c.JSON(order)
}

func (h *OrderHandler) UploadSlip(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	var order models.Order
	if err := database.DB.First(&order, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "order not found"})
	}

	file, err := c.FormFile("slip")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "slip file is required"})
	}

	ext := filepath.Ext(file.Filename)
	filename := fmt.Sprintf("slip_%d_%d%s", id, time.Now().Unix(), ext)
	savePath := filepath.Join(h.Config.UploadDir, filename)

	if err := c.SaveFile(file, savePath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save file"})
	}

	database.DB.Model(&order).Update("slip_image", filename)
	database.DB.Preload("Customer").Preload("Items").Preload("Items.Product").First(&order, id)

	h.Line.NotifySlipUploaded(&order)

	return c.JSON(order)
}

func (h *OrderHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	err = database.DB.Transaction(func(tx *gorm.DB) error {
		// Restore stock for each item
		var items []models.OrderItem
		tx.Where("order_id = ?", id).Find(&items)
		for _, item := range items {
			tx.Model(&models.Product{}).Where("id = ?", item.ProductID).
				Update("stock", gorm.Expr("stock + ?", item.Quantity))
		}
		tx.Where("order_id = ?", id).Delete(&models.OrderItem{})
		return tx.Delete(&models.Order{}, id).Error
	})

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete order"})
	}

	return c.JSON(fiber.Map{"message": "order deleted"})
}
