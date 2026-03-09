package handlers

import (
	"strconv"

	"brunocollective_inventory/database"
	"brunocollective_inventory/models"

	"github.com/gofiber/fiber/v2"
)

type CustomerHandler struct{}

func NewCustomerHandler() *CustomerHandler {
	return &CustomerHandler{}
}

func (h *CustomerHandler) List(c *fiber.Ctx) error {
	var customers []models.Customer

	query := database.DB
	if search := c.Query("search"); search != "" {
		query = query.Where("name LIKE ? OR phone LIKE ? OR email LIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	query.Order("created_at DESC").Find(&customers)
	return c.JSON(customers)
}

func (h *CustomerHandler) Get(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	var customer models.Customer
	if err := database.DB.First(&customer, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "customer not found"})
	}

	return c.JSON(customer)
}

func (h *CustomerHandler) Create(c *fiber.Ctx) error {
	var customer models.Customer
	if err := c.BodyParser(&customer); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if customer.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
	}

	if err := database.DB.Create(&customer).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create customer"})
	}

	return c.Status(fiber.StatusCreated).JSON(customer)
}

func (h *CustomerHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	var existing models.Customer
	if err := database.DB.First(&existing, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "customer not found"})
	}

	var updates models.Customer
	if err := c.BodyParser(&updates); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	database.DB.Model(&existing).Updates(updates)
	database.DB.First(&existing, id)
	return c.JSON(existing)
}

func (h *CustomerHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	if err := database.DB.Delete(&models.Customer{}, id).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete customer"})
	}

	return c.JSON(fiber.Map{"message": "customer deleted"})
}
