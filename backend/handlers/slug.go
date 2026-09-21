package handlers

import (
	"fmt"
	"log"
	"strings"

	"brunocollective_inventory/database"
	"brunocollective_inventory/models"
)

// Product URL slugs (storefront /products/{slug}).
//
// Slugs are ASCII-only so URLs stay clean and shareable (a Thai product name
// would otherwise percent-encode into an unreadable string). Slugify keeps
// [a-z0-9] and collapses everything else to single hyphens; when a name has
// no usable ASCII (e.g. a purely Thai name) the SKU is tried, then
// "product-{id}". The admin form lets the owner replace the generated slug
// with a hand-written English one.

// Slugify lowercases and reduces s to [a-z0-9-]. Returns "" when nothing
// survives (non-Latin names).
func Slugify(s string) string {
	var b strings.Builder
	lastDash := true // suppress a leading dash
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

// productSlugCandidate picks the base slug for a product: the explicit slug
// if the owner typed one, else the name, else the SKU, else product-{id}.
func productSlugCandidate(p *models.Product) string {
	if s := Slugify(p.Slug); s != "" {
		return s
	}
	if s := Slugify(p.Name); len(s) >= 3 {
		return s
	}
	if s := Slugify(p.SKU); s != "" {
		return "product-" + s
	}
	return fmt.Sprintf("product-%d", p.ID)
}

// uniqueProductSlug appends -2, -3, … until no other product uses the slug.
func uniqueProductSlug(base string, selfID uint) string {
	slug := base
	for n := 2; ; n++ {
		var count int64
		database.DB.Model(&models.Product{}).
			Where("slug = ? AND id <> ?", slug, selfID).
			Count(&count)
		if count == 0 {
			return slug
		}
		slug = fmt.Sprintf("%s-%d", base, n)
	}
}

// assignProductSlug normalises p.Slug (generating one when blank) and makes
// it unique. Call before saving.
func assignProductSlug(p *models.Product) {
	p.Slug = uniqueProductSlug(productSlugCandidate(p), p.ID)
}

// EnsureProductSlugs backfills slugs for products created before the slug
// column existed. Runs once at startup (idempotent — only blank slugs change).
func EnsureProductSlugs() {
	var products []models.Product
	database.DB.Where("slug IS NULL OR slug = ''").Find(&products)
	for i := range products {
		assignProductSlug(&products[i])
		database.DB.Model(&products[i]).Update("slug", products[i].Slug)
	}
	if len(products) > 0 {
		log.Printf("Backfilled slugs for %d products", len(products))
	}
}
