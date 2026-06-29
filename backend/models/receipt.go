package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// ReceiptLine is a snapshot of one line on a receipt, captured at issue time so
// the document never changes even if the underlying product/order later does.
type ReceiptLine struct {
	Name     string  `json:"name"`
	Size     string  `json:"size"`
	Price    float64 `json:"price"`
	Quantity int     `json:"quantity"`
}

// ReceiptLines is a slice of ReceiptLine stored as a JSON TEXT column (keeps the
// AutoMigrate, no-migration-files pattern — same approach as Product.Images).
type ReceiptLines []ReceiptLine

func (l ReceiptLines) Value() (driver.Value, error) {
	if l == nil {
		return "[]", nil
	}
	b, err := json.Marshal(l)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func (l *ReceiptLines) Scan(value interface{}) error {
	if value == nil {
		*l = ReceiptLines{}
		return nil
	}
	var data []byte
	switch v := value.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		return errors.New("unsupported type for ReceiptLines")
	}
	if len(data) == 0 {
		*l = ReceiptLines{}
		return nil
	}
	return json.Unmarshal(data, l)
}

// Receipt is an issued receipt (ใบเสร็จรับเงิน) — NOT a tax invoice, since the
// shop is not VAT-registered. Each order has at most one receipt (OrderID is a
// unique index); re-issuing returns the existing one. ReceiptNo is a
// human-facing running number assigned per month (RC-YYYYMM-NNNN).
type Receipt struct {
	ID           uint         `json:"id" gorm:"primaryKey"`
	ReceiptNo    string       `json:"receipt_no" gorm:"uniqueIndex;not null"`
	OrderID      uint         `json:"order_id" gorm:"uniqueIndex;not null"`
	BuyerName    string       `json:"buyer_name"`
	BuyerAddress string       `json:"buyer_address"`
	BuyerTaxID   string       `json:"buyer_tax_id"`
	Lines        ReceiptLines `json:"lines" gorm:"type:text"`
	TotalAmount  float64      `json:"total_amount"`
	IssuedAt     time.Time    `json:"issued_at"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}
