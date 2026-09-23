package models

import "time"

// LiveTryOnSession records one live (webcam) try-on session for cost
// tracking. Decart bills per streamed second; BilledSeconds is what the
// browser reported from the SDK's generation events (an estimate — the
// invoice is the truth), capped server-side at CapSeconds + grace.
type LiveTryOnSession struct {
	ID            uint       `json:"id" gorm:"primaryKey"`
	CustomerID    *uint      `json:"customer_id" gorm:"index"` // nil = guest
	QuotaKey      string     `json:"quota_key"`                // "member:42" | "ip:…"
	ProductID     uint       `json:"product_id" gorm:"index"`
	CapSeconds    int        `json:"cap_seconds"`
	BilledSeconds float64    `json:"billed_seconds"`
	Reason        string     `json:"reason"` // ended | timeout | hidden | no_person | closed | error
	Reported      bool       `json:"reported"`
	CreatedAt     time.Time  `json:"created_at" gorm:"index"`
	EndedAt       *time.Time `json:"ended_at"`
}
