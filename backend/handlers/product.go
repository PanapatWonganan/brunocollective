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
	"brunocollective_inventory/services"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type ProductHandler struct {
	Config *config.Config
}

func NewProductHandler(cfg *config.Config) *ProductHandler {
	return &ProductHandler{Config: cfg}
}

// ProductDisplayOrder sorts explicitly-ordered products first (1..N), then
// unordered ones (display_order 0) newest-first. Used by both the admin list
// and the public shop so the admin sees exactly what the storefront shows.
const ProductDisplayOrder = "CASE WHEN display_order = 0 THEN 1 ELSE 0 END, display_order ASC, created_at DESC"

func (h *ProductHandler) List(c *fiber.Ctx) error {
	var products []models.Product

	query := database.DB.Preload("Variants")
	if search := c.Query("search"); search != "" {
		query = query.Where("name LIKE ? OR sku LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	query.Order(ProductDisplayOrder).Find(&products)
	for i := range products {
		products[i].ComputeTotalStock()
	}
	return c.JSON(products)
}

func (h *ProductHandler) Get(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	var product models.Product
	if err := database.DB.Preload("Variants").First(&product, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "product not found"})
	}

	product.ComputeTotalStock()
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

	// GORM creates the nested Variants alongside the product (full-save association).
	if err := database.DB.Create(&product).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create product"})
	}

	product.ComputeTotalStock()
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

	// Scalar fields: GORM Updates skips zero values, so use a map for the fields
	// that legitimately may be set to a zero/empty value (e.g. clearing size, or
	// stock going to 0). Images are managed via the dedicated upload endpoints.
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&existing).Updates(map[string]interface{}{
			"name":        updates.Name,
			"sku":         updates.SKU,
			"size":        updates.Size,
			"description": updates.Description,
			"category":    updates.Category,
			"price":       updates.Price,
			"cost":        updates.Cost,
			// Pointer: nil clears the override (inherit), 0 = no commission.
			"commission_percent": updates.CommissionPercent,
			"stock":              updates.Stock,
		}).Error; err != nil {
			return err
		}

		// Replace the variant set: delete the old rows, recreate from the payload.
		// A request that omits "variants" leaves variants untouched only if nil;
		// an explicit empty array clears them (variant-less / legacy product).
		if updates.Variants != nil {
			if err := tx.Where("product_id = ?", existing.ID).Delete(&models.ProductVariant{}).Error; err != nil {
				return err
			}
			for i := range updates.Variants {
				v := updates.Variants[i]
				v.ID = 0
				v.ProductID = existing.ID
				if err := tx.Create(&v).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update product"})
	}

	database.DB.Preload("Variants").First(&existing, id)
	existing.ComputeTotalStock()
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

	// Remove the product's variants too (no cascade in SQLite/GORM here).
	database.DB.Where("product_id = ?", id).Delete(&models.ProductVariant{})

	// Best-effort cleanup of the product's uploaded images.
	for _, img := range product.Images {
		removeUpload(h.Config.UploadDir, img)
	}

	return c.JSON(fiber.Map{"message": "product deleted"})
}

// Reorder sets the storefront display order in bulk: body {ids: [...]} assigns
// display_order 1..N following the array. The admin dialog always sends the
// full product list, so every product ends up explicitly ordered.
func (h *ProductHandler) Reorder(c *fiber.Ctx) error {
	var body struct {
		IDs []uint `json:"ids"`
	}
	if err := c.BodyParser(&body); err != nil || len(body.IDs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ids is required"})
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		for i, id := range body.IDs {
			if err := tx.Model(&models.Product{}).Where("id = ?", id).
				Update("display_order", i+1).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to reorder products"})
	}

	return c.JSON(fiber.Map{"message": "reordered"})
}

// Merge folds duplicate product :id into another product (target_id): all order
// history, variant stock, sale-page references and images move to the target,
// then the duplicate is deleted. Sales analytics combine automatically because
// every aggregate keys on order_items.product_id. Used when staff accidentally
// create the same physical product twice under different SKUs.
func (h *ProductHandler) Merge(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	var body struct {
		TargetID uint `json:"target_id"`
	}
	if err := c.BodyParser(&body); err != nil || body.TargetID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "target_id is required"})
	}
	if body.TargetID == uint(id) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot merge a product into itself"})
	}

	var source, target models.Product
	if err := database.DB.Preload("Variants").First(&source, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "product not found"})
	}
	if err := database.DB.Preload("Variants").First(&target, body.TargetID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "target product not found"})
	}

	var movedItems int64
	database.DB.Model(&models.OrderItem{}).Where("product_id = ?", source.ID).Count(&movedItems)

	err = database.DB.Transaction(func(tx *gorm.DB) error {
		variantKey := func(size, color string) string { return size + "\x00" + color }

		// If the target is a legacy single-size product but the source brings
		// variants, convert the target's legacy stock into a variant first so it
		// stays visible once the product has a variant list.
		if len(target.Variants) == 0 && len(source.Variants) > 0 && target.Stock > 0 {
			v := models.ProductVariant{ProductID: target.ID, Size: target.Size, Stock: target.Stock}
			if err := tx.Create(&v).Error; err != nil {
				return err
			}
			if err := tx.Model(&target).Update("stock", 0).Error; err != nil {
				return err
			}
			target.Variants = append(target.Variants, v)
		}

		targetByKey := map[string]*models.ProductVariant{}
		for i := range target.Variants {
			v := &target.Variants[i]
			targetByKey[variantKey(v.Size, v.Color)] = v
		}

		// Variants: same size+color merges stock into the target's variant (order
		// lines follow); otherwise the whole variant moves across unchanged.
		for i := range source.Variants {
			sv := source.Variants[i]
			if tv, ok := targetByKey[variantKey(sv.Size, sv.Color)]; ok {
				if err := tx.Model(&models.ProductVariant{}).Where("id = ?", tv.ID).
					Update("stock", gorm.Expr("stock + ?", sv.Stock)).Error; err != nil {
					return err
				}
				if err := tx.Model(&models.OrderItem{}).Where("variant_id = ?", sv.ID).
					Update("variant_id", tv.ID).Error; err != nil {
					return err
				}
				if err := tx.Delete(&models.ProductVariant{}, sv.ID).Error; err != nil {
					return err
				}
			} else {
				if err := tx.Model(&models.ProductVariant{}).Where("id = ?", sv.ID).
					Update("product_id", target.ID).Error; err != nil {
					return err
				}
			}
		}

		// Legacy source stock (no variants): sum into the target's legacy stock,
		// or into the matching/new variant when the target sells by variant.
		if len(source.Variants) == 0 && source.Stock > 0 {
			if len(target.Variants) == 0 {
				if err := tx.Model(&target).Update("stock", gorm.Expr("stock + ?", source.Stock)).Error; err != nil {
					return err
				}
			} else if tv, ok := targetByKey[variantKey(source.Size, "")]; ok {
				if err := tx.Model(&models.ProductVariant{}).Where("id = ?", tv.ID).
					Update("stock", gorm.Expr("stock + ?", source.Stock)).Error; err != nil {
					return err
				}
			} else {
				v := models.ProductVariant{ProductID: target.ID, Size: source.Size, Stock: source.Stock}
				if err := tx.Create(&v).Error; err != nil {
					return err
				}
			}
		}

		// Order history and sale-page references follow the merge.
		if err := tx.Model(&models.OrderItem{}).Where("product_id = ?", source.ID).
			Update("product_id", target.ID).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.SalePage{}).Where("product_id = ?", source.ID).
			Update("product_id", target.ID).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.SalePage{}).Where("bump_product_id = ?", source.ID).
			Update("bump_product_id", target.ID).Error; err != nil {
			return err
		}

		// Keep the target's details; adopt the source's images (files stay on
		// disk) and fill in fields the target is missing.
		updates := map[string]interface{}{}
		gallery := append(models.StringSlice{}, target.Images...)
		seen := map[string]bool{}
		for _, img := range gallery {
			seen[img] = true
		}
		for _, img := range append(models.StringSlice{source.ImageURL}, source.Images...) {
			if img != "" && !seen[img] {
				gallery = append(gallery, img)
				seen[img] = true
			}
		}
		updates["images"] = gallery
		if target.ImageURL == "" && source.ImageURL != "" {
			updates["image_url"] = source.ImageURL
		}
		if target.Description == "" && source.Description != "" {
			updates["description"] = source.Description
		}
		if target.Category == "" && source.Category != "" {
			updates["category"] = source.Category
		}
		if target.Cost == 0 && source.Cost > 0 {
			updates["cost"] = source.Cost
		}
		if err := tx.Model(&target).Updates(updates).Error; err != nil {
			return err
		}

		// Drop the duplicate row only — its images now belong to the target.
		return tx.Delete(&models.Product{}, source.ID).Error
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to merge products"})
	}

	var merged models.Product
	database.DB.Preload("Variants").First(&merged, target.ID)
	merged.ComputeTotalStock()
	return c.JSON(fiber.Map{"product": merged, "moved_items": movedItems})
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
		base := fmt.Sprintf("product_%d_%d", product.ID, time.Now().UnixNano())
		filename, err := services.SaveOptimizedImage(file, h.Config.UploadDir, base, services.MaxDimProduct)
		if err != nil {
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
