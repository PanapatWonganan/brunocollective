package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"brunocollective_inventory/database"
	"brunocollective_inventory/models"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type ReceiptHandler struct{}

func NewReceiptHandler() *ReceiptHandler {
	return &ReceiptHandler{}
}

// List returns all issued receipts, newest first (admin history).
func (h *ReceiptHandler) List(c *fiber.Ctx) error {
	var receipts []models.Receipt
	database.DB.Order("issued_at DESC").Find(&receipts)
	return c.JSON(receipts)
}

// Get returns the receipt for an order, or 404 if none has been issued yet.
func (h *ReceiptHandler) Get(c *fiber.Ctx) error {
	orderID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	var receipt models.Receipt
	if err := database.DB.Where("order_id = ?", orderID).First(&receipt).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "no receipt issued"})
	}
	return c.JSON(receipt)
}

// Issue creates a receipt for an order with a running number, or returns the
// existing one if the order already has a receipt (re-issue is idempotent).
// The buyer fields can override the order's customer details for the document.
func (h *ReceiptHandler) Issue(c *fiber.Ctx) error {
	orderID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	var body struct {
		BuyerName    string `json:"buyer_name"`
		BuyerAddress string `json:"buyer_address"`
		BuyerTaxID   string `json:"buyer_tax_id"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	var receipt models.Receipt
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		// Idempotent: if a receipt already exists for this order, return it as-is.
		if err := tx.Where("order_id = ?", orderID).First(&receipt).Error; err == nil {
			return nil
		} else if err != gorm.ErrRecordNotFound {
			return err
		}

		// Load the order with its items + products to snapshot the lines.
		var order models.Order
		if err := tx.Preload("Customer").Preload("Items").Preload("Items.Product").
			First(&order, orderID).Error; err != nil {
			return fiber.NewError(fiber.StatusNotFound, "order not found")
		}

		lines := make(models.ReceiptLines, 0, len(order.Items))
		for _, item := range order.Items {
			lines = append(lines, models.ReceiptLine{
				Name:     item.Product.Name,
				Size:     item.Product.Size,
				Price:    item.Price,
				Quantity: item.Quantity,
			})
		}

		// Default buyer details to the order's customer when not provided.
		buyerName := strings.TrimSpace(body.BuyerName)
		if buyerName == "" {
			buyerName = order.Customer.Name
		}
		buyerAddress := strings.TrimSpace(body.BuyerAddress)
		if buyerAddress == "" {
			buyerAddress = order.Customer.Address
		}

		issuedAt := time.Now()
		receiptNo, err := nextReceiptNo(tx, issuedAt)
		if err != nil {
			return err
		}

		receipt = models.Receipt{
			ReceiptNo:    receiptNo,
			OrderID:      order.ID,
			BuyerName:    buyerName,
			BuyerAddress: buyerAddress,
			BuyerTaxID:   strings.TrimSpace(body.BuyerTaxID),
			Lines:        lines,
			TotalAmount:  order.TotalAmount,
			IssuedAt:     issuedAt,
		}
		return tx.Create(&receipt).Error
	})

	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return c.Status(fe.Code).JSON(fiber.Map{"error": fe.Message})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to issue receipt"})
	}

	return c.Status(fiber.StatusCreated).JSON(receipt)
}

// nextReceiptNo computes the next running receipt number for the issue month:
// RC-YYYYMM-NNNN, where NNNN restarts at 0001 each month. Must run inside the
// same transaction as the insert so concurrent issues don't collide.
func nextReceiptNo(tx *gorm.DB, issuedAt time.Time) (string, error) {
	prefix := fmt.Sprintf("RC-%04d%02d-", issuedAt.Year(), int(issuedAt.Month()))

	var count int64
	if err := tx.Model(&models.Receipt{}).
		Where("receipt_no LIKE ?", prefix+"%").
		Count(&count).Error; err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%04d", prefix, count+1), nil
}
