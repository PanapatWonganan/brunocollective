package handlers

import (
	"bytes"
	"encoding/csv"
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

type OrderHandler struct {
	Config   *config.Config
	Telegram *services.TelegramNotifier
}

func NewOrderHandler(cfg *config.Config, telegram *services.TelegramNotifier) *OrderHandler {
	return &OrderHandler{Config: cfg, Telegram: telegram}
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

// ExportCSV returns orders within a date range as a CSV file for accounting.
// One row per order. Query params:
//   from, to            — inclusive date range, YYYY-MM-DD (server local time).
//                         Defaults: from = start of current month, to = today.
//   include_cancelled=1 — include cancelled orders (excluded by default).
// The file is UTF-8 with a BOM so Excel renders Thai text correctly.
func (h *OrderHandler) ExportCSV(c *fiber.Ctx) error {
	loc := time.Now().Location()

	// Parse the range; default to the current month.
	now := time.Now()
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
	to := now
	if v := c.Query("from"); v != "" {
		if t, err := time.ParseInLocation("2006-01-02", v, loc); err == nil {
			from = t
		} else {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid 'from' date (use YYYY-MM-DD)"})
		}
	}
	if v := c.Query("to"); v != "" {
		if t, err := time.ParseInLocation("2006-01-02", v, loc); err == nil {
			to = t
		} else {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid 'to' date (use YYYY-MM-DD)"})
		}
	}
	// Make 'to' inclusive of the whole day.
	to = time.Date(to.Year(), to.Month(), to.Day(), 23, 59, 59, 0, loc)

	query := database.DB.
		Preload("Customer").Preload("Items").Preload("Items.Product").
		Where("created_at BETWEEN ? AND ?", from, to)
	if c.Query("include_cancelled") != "1" {
		query = query.Where("status <> ?", models.StatusCancelled)
	}

	var orders []models.Order
	query.Order("created_at ASC").Find(&orders)

	// Build the CSV in memory (order volumes are small).
	var buf bytes.Buffer
	buf.WriteString("\xEF\xBB\xBF") // UTF-8 BOM for Excel
	w := csv.NewWriter(&buf)
	w.Write([]string{
		"Order No", "Date", "Time", "Receipt No", "Customer", "Phone",
		"Address", "Tax ID", "Items", "Item Count", "Status", "Total (THB)",
	})

	// Map order -> receipt number (orders without a receipt show blank).
	var receipts []models.Receipt
	database.DB.Find(&receipts)
	receiptByOrder := make(map[uint]string, len(receipts))
	taxByOrder := make(map[uint]string, len(receipts))
	for _, r := range receipts {
		receiptByOrder[r.OrderID] = r.ReceiptNo
		taxByOrder[r.OrderID] = r.BuyerTaxID
	}

	for _, o := range orders {
		// Summarise the line items into one cell, e.g. "Polo (M/White) x2; Cap x1".
		parts := make([]string, 0, len(o.Items))
		itemCount := 0
		for _, it := range o.Items {
			label := it.Product.Name
			variant := strings.TrimSpace(strings.Join([]string{it.Size, it.Color}, "/"))
			variant = strings.Trim(variant, "/")
			if variant != "" {
				label += " (" + variant + ")"
			}
			parts = append(parts, fmt.Sprintf("%s x%d", label, it.Quantity))
			itemCount += it.Quantity
		}

		w.Write([]string{
			strconv.FormatUint(uint64(o.ID), 10),
			o.CreatedAt.In(loc).Format("2006-01-02"),
			o.CreatedAt.In(loc).Format("15:04"),
			receiptByOrder[o.ID],
			o.Customer.Name,
			o.Customer.Phone,
			o.Customer.Address,
			taxByOrder[o.ID],
			strings.Join(parts, "; "),
			strconv.Itoa(itemCount),
			string(o.Status),
			strconv.FormatFloat(o.TotalAmount, 'f', 2, 64),
		})
	}
	w.Flush()

	filename := fmt.Sprintf("orders_%s_to_%s.csv", from.Format("2006-01-02"), to.Format("2006-01-02"))
	c.Set("Content-Type", "text/csv; charset=utf-8")
	c.Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	return c.Send(buf.Bytes())
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

	// Support both JSON and multipart form
	contentType := c.Get("Content-Type")
	if len(contentType) >= 19 && contentType[:19] == "multipart/form-data" {
		customerID, _ := strconv.Atoi(c.FormValue("customer_id"))
		req.CustomerID = uint(customerID)
		req.Notes = c.FormValue("notes")
		itemsJSON := c.FormValue("items")
		if itemsJSON != "" {
			json.Unmarshal([]byte(itemsJSON), &req.Items)
		}
	} else {
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}
	}

	if req.CustomerID == 0 || len(req.Items) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "customer_id and items are required"})
	}

	// Verify customer exists
	var customer models.Customer
	if err := database.DB.First(&customer, req.CustomerID).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "customer not found"})
	}

	// Handle slip file if provided
	var slipFilename string
	slipFile, err := c.FormFile("slip")
	if err == nil && slipFile != nil {
		ext := filepath.Ext(slipFile.Filename)
		slipFilename = fmt.Sprintf("slip_new_%d%s", time.Now().UnixNano(), ext)
		savePath := filepath.Join(h.Config.UploadDir, slipFilename)
		if err := c.SaveFile(slipFile, savePath); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save slip"})
		}
	}

	// Build order within a transaction
	var order models.Order
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		var totalAmount float64
		var items []models.OrderItem

		for _, item := range req.Items {
			orderItem, price, err := buildOrderItem(tx, item)
			if err != nil {
				return err
			}
			totalAmount += price * float64(item.Quantity)
			items = append(items, orderItem)
		}

		order = models.Order{
			CustomerID:  req.CustomerID,
			Status:      models.StatusPending,
			TotalAmount: totalAmount,
			Notes:       req.Notes,
			SlipImage:   slipFilename,
			Items:       items,
		}

		return tx.Create(&order).Error
	})

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	// Rename slip file with actual order ID
	if slipFilename != "" {
		newFilename := fmt.Sprintf("slip_%d_%d%s", order.ID, time.Now().Unix(), filepath.Ext(slipFilename))
		oldPath := filepath.Join(h.Config.UploadDir, slipFilename)
		newPath := filepath.Join(h.Config.UploadDir, newFilename)
		if err := fileRename(oldPath, newPath); err == nil {
			database.DB.Model(&order).Update("slip_image", newFilename)
			order.SlipImage = newFilename
		}
	}

	// Reload with relations
	database.DB.Preload("Customer").Preload("Items").Preload("Items.Product").First(&order, order.ID)

	h.Telegram.NotifyNewOrder(&order)

	return c.Status(fiber.StatusCreated).JSON(order)
}

func fileRename(oldPath, newPath string) error {
	return os.Rename(oldPath, newPath)
}

// buildOrderItem validates one requested line, deducts stock (from the chosen
// variant when VariantID is set, else from the legacy Product.Stock), and
// returns the OrderItem with a size/color snapshot plus the unit price. It runs
// inside the caller's transaction so a later failure rolls back the deduction.
// Shared by the admin order handler and the public storefront checkout.
func buildOrderItem(tx *gorm.DB, item models.CreateOrderItem) (models.OrderItem, float64, error) {
	if item.Quantity <= 0 {
		return models.OrderItem{}, 0, fiber.NewError(fiber.StatusBadRequest, "quantity must be positive")
	}

	var product models.Product
	if err := tx.First(&product, item.ProductID).Error; err != nil {
		return models.OrderItem{}, 0, fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("product %d not found", item.ProductID))
	}

	orderItem := models.OrderItem{
		ProductID: item.ProductID,
		Quantity:  item.Quantity,
		Price:     product.Price,
	}

	if item.VariantID != nil {
		var variant models.ProductVariant
		if err := tx.Where("id = ? AND product_id = ?", *item.VariantID, item.ProductID).First(&variant).Error; err != nil {
			return models.OrderItem{}, 0, fiber.NewError(fiber.StatusBadRequest, "variant not found")
		}
		if variant.Stock < item.Quantity {
			return models.OrderItem{}, 0, fiber.NewError(fiber.StatusBadRequest, "insufficient stock for "+product.Name+" ("+variant.Size+" "+variant.Color+")")
		}
		if err := tx.Model(&variant).Update("stock", variant.Stock-item.Quantity).Error; err != nil {
			return models.OrderItem{}, 0, err
		}
		orderItem.VariantID = item.VariantID
		orderItem.Size = variant.Size
		orderItem.Color = variant.Color
	} else {
		// Legacy / variant-less product: deduct the product-level stock.
		if product.Stock < item.Quantity {
			return models.OrderItem{}, 0, fiber.NewError(fiber.StatusBadRequest,
				fmt.Sprintf("insufficient stock for %s (available: %d, requested: %d)", product.Name, product.Stock, item.Quantity))
		}
		if err := tx.Model(&product).Update("stock", product.Stock-item.Quantity).Error; err != nil {
			return models.OrderItem{}, 0, err
		}
		orderItem.Size = product.Size
	}

	return orderItem, product.Price, nil
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

	h.Telegram.NotifyStatusChange(&order, body.Status)

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

	h.Telegram.NotifySlipUploaded(&order)

	return c.JSON(order)
}

func (h *OrderHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	err = database.DB.Transaction(func(tx *gorm.DB) error {
		// Restore stock for each item — to the variant if the line had one,
		// otherwise to the legacy product-level stock.
		var items []models.OrderItem
		tx.Where("order_id = ?", id).Find(&items)
		for _, item := range items {
			if item.VariantID != nil {
				tx.Model(&models.ProductVariant{}).Where("id = ?", *item.VariantID).
					Update("stock", gorm.Expr("stock + ?", item.Quantity))
			} else {
				tx.Model(&models.Product{}).Where("id = ?", item.ProductID).
					Update("stock", gorm.Expr("stock + ?", item.Quantity))
			}
		}
		tx.Where("order_id = ?", id).Delete(&models.OrderItem{})
		return tx.Delete(&models.Order{}, id).Error
	})

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete order"})
	}

	return c.JSON(fiber.Map{"message": "order deleted"})
}
