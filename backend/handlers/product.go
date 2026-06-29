package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"brunocollective_inventory/config"
	"brunocollective_inventory/database"
	"brunocollective_inventory/models"

	"github.com/gofiber/fiber/v2"
)

type ProductHandler struct {
	Config *config.Config
}

func NewProductHandler(cfg *config.Config) *ProductHandler {
	return &ProductHandler{Config: cfg}
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

	var product models.Product
	if err := database.DB.First(&product, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "product not found"})
	}

	if err := database.DB.Delete(&models.Product{}, id).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete product"})
	}

	// Best-effort cleanup of the product's uploaded images.
	for _, img := range product.Images {
		removeUpload(h.Config.UploadDir, img)
	}

	return c.JSON(fiber.Map{"message": "product deleted"})
}

// UploadImages accepts one or more image files (multipart field "images") and
// appends them to the product's gallery. The first image also seeds image_url
// when none is set, so existing single-image consumers keep working.
func (h *ProductHandler) UploadImages(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	var product models.Product
	if err := database.DB.First(&product, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "product not found"})
	}

	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid upload"})
	}
	files := form.File["images"]
	if len(files) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "no images provided"})
	}

	for _, file := range files {
		ext := filepath.Ext(file.Filename)
		filename := fmt.Sprintf("product_%d_%d%s", product.ID, time.Now().UnixNano(), ext)
		if err := c.SaveFile(file, filepath.Join(h.Config.UploadDir, filename)); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save image"})
		}
		product.Images = append(product.Images, "/uploads/"+filename)
	}

	if product.ImageURL == "" && len(product.Images) > 0 {
		product.ImageURL = product.Images[0]
	}

	if err := database.DB.Model(&product).Updates(map[string]interface{}{
		"images":    product.Images,
		"image_url": product.ImageURL,
	}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update product"})
	}

	return c.JSON(product)
}

// DeleteImage removes a single image (by its URL/path) from the product gallery
// and deletes the underlying file. If the removed image was the primary
// image_url, it is re-pointed at the next remaining image (or cleared).
func (h *ProductHandler) DeleteImage(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	var body struct {
		Image string `json:"image"`
	}
	if err := c.BodyParser(&body); err != nil || strings.TrimSpace(body.Image) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "image is required"})
	}

	var product models.Product
	if err := database.DB.First(&product, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "product not found"})
	}

	remaining := make(models.StringSlice, 0, len(product.Images))
	found := false
	for _, img := range product.Images {
		if img == body.Image {
			found = true
			continue
		}
		remaining = append(remaining, img)
	}
	if !found {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "image not found"})
	}

	product.Images = remaining
	if product.ImageURL == body.Image {
		if len(remaining) > 0 {
			product.ImageURL = remaining[0]
		} else {
			product.ImageURL = ""
		}
	}

	if err := database.DB.Model(&product).Updates(map[string]interface{}{
		"images":    product.Images,
		"image_url": product.ImageURL,
	}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update product"})
	}

	removeUpload(h.Config.UploadDir, body.Image)

	return c.JSON(product)
}

// removeUpload deletes an uploaded file given its stored URL/path (e.g.
// "/uploads/product_1_123.jpg"). External (http) URLs are left untouched.
func removeUpload(uploadDir, img string) {
	if img == "" || strings.HasPrefix(img, "http://") || strings.HasPrefix(img, "https://") {
		return
	}
	name := filepath.Base(img)
	if name == "." || name == "/" || name == "" {
		return
	}
	os.Remove(filepath.Join(uploadDir, name))
}
