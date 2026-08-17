package models

import "time"

// Broadcast is one LINE campaign push to a chosen set of RFM segments.
// Sending runs in a background goroutine; Sent/Failed tick up as it goes and
// Status flips to "done" at the end, so the admin UI can poll for progress.
type Broadcast struct {
	ID      uint   `json:"id" gorm:"primaryKey"`
	Message string `json:"message" gorm:"not null"`
	// Segments are the selected audience segment names (RFM labels plus the
	// special "ยังไม่เคยซื้อ" / "ยังไม่ผูกลูกค้า" buckets).
	Segments  StringSlice `json:"segments" gorm:"type:text"`
	Total     int         `json:"total"`
	Sent      int         `json:"sent"`
	Failed    int         `json:"failed"`
	Status    string      `json:"status" gorm:"default:sending"` // sending | done
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}
