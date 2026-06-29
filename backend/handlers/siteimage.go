package handlers

import (
	"fmt"
	"path/filepath"
	"time"

	"brunocollective_inventory/config"
	"brunocollective_inventory/database"
	"brunocollective_inventory/models"

	"github.com/gofiber/fiber/v2"
)

// SiteImageHandler manages the editable storefront images (hero, lookbook,
// journal). Admin routes are JWT-protected; a public read route lives on the
// shop handler/group.
type SiteImageHandler struct {
	Config *config.Config
}

func NewSiteImageHandler(cfg *config.Config) *SiteImageHandler {
	return &SiteImageHandler{Config: cfg}
}

// List returns all site image slots (admin).
func (h *SiteImageHandler) List(c *fiber.Ctx) error {
	var images []models.SiteImage
	database.DB.Order("id ASC").Find(&images)
	return c.JSON(images)
}

// findOrInit fetches the slot by key, creating an empty row if it does not yet
// exist (so newly-added slot keys work without a re-seed).
func findOrInit(key string) (*models.SiteImage, error) {
	var img models.SiteImage
	err := database.DB.Where("key = ?", key).First(&img).Error
	if err != nil {
		img = models.SiteImage{Key: key}
		if cErr := database.DB.Create(&img).Error; cErr != nil {
			return nil, cErr
		}
	}
	return &img, nil
}

// UploadImage replaces the image for a slot (admin, multipart field "image").
func (h *SiteImageHandler) UploadImage(c *fiber.Ctx) error {
	key := c.Params("key")
	if key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "key is required"})
	}

	img, err := findOrInit(key)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to load slot"})
	}

	file, err := c.FormFile("image")
	if err != nil || file == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "no image provided"})
	}

	ext := filepath.Ext(file.Filename)
	filename := fmt.Sprintf("site_%s_%d%s", key, time.Now().UnixNano(), ext)
	if err := c.SaveFile(file, filepath.Join(h.Config.UploadDir, filename)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save image"})
	}

	// Remove the previous file (best effort) before pointing at the new one.
	removeUpload(h.Config.UploadDir, img.ImageURL)

	img.ImageURL = "/uploads/" + filename
	if err := database.DB.Model(img).Update("image_url", img.ImageURL).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update slot"})
	}

	return c.JSON(img)
}

// UpdateCaptions edits the captions for a slot (admin, JSON body).
func (h *SiteImageHandler) UpdateCaptions(c *fiber.Ctx) error {
	key := c.Params("key")
	if key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "key is required"})
	}

	img, err := findOrInit(key)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to load slot"})
	}

	var body struct {
		CaptionA string `json:"caption_a"`
		CaptionB string `json:"caption_b"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	// Use a map so empty strings are persisted (clearing a caption is allowed).
	database.DB.Model(img).Updates(map[string]interface{}{
		"caption_a": body.CaptionA,
		"caption_b": body.CaptionB,
	})
	img.CaptionA = body.CaptionA
	img.CaptionB = body.CaptionB

	return c.JSON(img)
}
