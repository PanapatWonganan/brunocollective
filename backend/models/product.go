package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// StringSlice is a list of strings stored as a JSON-encoded TEXT column in
// SQLite but serialized as a plain JSON array over the API. Used for product
// image galleries so we avoid a separate images table (keeps the AutoMigrate,
// no-migration-files pattern intact).
type StringSlice []string

// Value implements driver.Valuer — encode to JSON for storage.
func (s StringSlice) Value() (driver.Value, error) {
	if s == nil {
		return "[]", nil
	}
	b, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

// Scan implements sql.Scanner — decode the stored JSON back into the slice.
func (s *StringSlice) Scan(value interface{}) error {
	if value == nil {
		*s = StringSlice{}
		return nil
	}
	var data []byte
	switch v := value.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		return errors.New("unsupported type for StringSlice")
	}
	if len(data) == 0 {
		*s = StringSlice{}
		return nil
	}
	return json.Unmarshal(data, s)
}

type Product struct {
	ID          uint        `json:"id" gorm:"primaryKey"`
	Name        string      `json:"name" gorm:"not null"`
	SKU         string      `json:"sku" gorm:"uniqueIndex"`
	Size        string      `json:"size"` // legacy: single-size garments without variants
	Description string      `json:"description"`
	Price       float64     `json:"price" gorm:"not null"`
	Stock       int         `json:"stock" gorm:"default:0"` // legacy: used only when a product has no variants
	ImageURL    string      `json:"image_url"`
	Images      StringSlice `json:"images" gorm:"type:text"`

	// Variants are the sellable size+color combinations. A product with no
	// variants is treated as a single legacy unit (Size/Stock above).
	Variants []ProductVariant `json:"variants" gorm:"foreignKey:ProductID"`

	// TotalStock is derived (sum of variant stock, or legacy Stock when no
	// variants). Not persisted — computed by handlers for list/shop display.
	TotalStock int `json:"total_stock" gorm:"-"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ComputeTotalStock sets TotalStock from variants, falling back to the legacy
// Stock field when the product has no variants. Call after loading a product.
func (p *Product) ComputeTotalStock() {
	if len(p.Variants) == 0 {
		p.TotalStock = p.Stock
		return
	}
	total := 0
	for _, v := range p.Variants {
		total += v.Stock
	}
	p.TotalStock = total
}
