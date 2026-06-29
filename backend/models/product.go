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
	Size        string      `json:"size"`
	Description string      `json:"description"`
	Price       float64     `json:"price" gorm:"not null"`
	Stock       int         `json:"stock" gorm:"default:0"`
	ImageURL    string      `json:"image_url"`
	Images      StringSlice `json:"images" gorm:"type:text"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}
