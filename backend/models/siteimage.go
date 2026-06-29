package models

import "time"

// SiteImage holds an editable image + captions for a fixed slot on the public
// storefront (hero, lookbook tiles, journal entries). Slots are addressed by a
// stable Key so the storefront can look them up and fall back to its built-in
// defaults when a slot has not been customised yet.
type SiteImage struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Key       string    `json:"key" gorm:"uniqueIndex;not null"`
	ImageURL  string    `json:"image_url"`
	CaptionA  string    `json:"caption_a"`
	CaptionB  string    `json:"caption_b"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SiteImageSlots is the canonical list of slots seeded on startup. Label is a
// human-readable name shown in the admin; it is not stored, only used for seeding
// nothing — labels live in the admin UI. Keep these keys in sync with the
// storefront components that consume them.
var SiteImageSlots = []string{
	"hero",
	"lookbook_1", "lookbook_2", "lookbook_3",
	"lookbook_4", "lookbook_5", "lookbook_6",
	"journal_1", "journal_2", "journal_3",
}
