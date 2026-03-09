package handlers

import (
	"strconv"

	"brunocollective_inventory/database"
	"brunocollective_inventory/models"

	"github.com/gofiber/fiber/v2"
)

type ProductHandler struct{}

func NewProductHandler() *ProductHandler {
	return &ProductHandler{}
}

func (h *ProductHandler) List(c *fiber.Ctx) error {
	var products []models.Product

	query := database.DB
	if search := c.Query("search"); search != "" {
		query = query.Where("name LIKE ? OR sku LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	query.Order("created_at DESC").Find(&products)
	return c.JSON(products)
}

func (h *ProductHandler) Get(c *fiber.Ctx) error {
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

func (h *ProductHandler) Create(c *fiber.Ctx) error {
	var product models.Product
	if err := c.BodyParser(&product); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if product.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
	}

	if err := database.DB.Create(&product).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create product"})
	}

	return c.Status(fiber.StatusCreated).JSON(product)
}

func (h *ProductHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	var existing models.Product
	if err := database.DB.First(&existing, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "product not found"})
	}

	var updates models.Product
	if err := c.BodyParser(&updates); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	database.DB.Model(&existing).Updates(updates)
	database.DB.First(&existing, id)
	return c.JSON(existing)
}

func (h *ProductHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	if err := database.DB.Delete(&models.Product{}, id).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete product"})
	}

	return c.JSON(fiber.Map{"message": "product deleted"})
}
