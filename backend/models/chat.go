package models

import "time"

// Conversation is one chat thread with a shopper on an external platform.
// Platform + ExternalID identify the counterpart (e.g. "line" + LINE userId);
// the same schema serves Facebook/Instagram later (platform "facebook"/
// "instagram", ExternalID = PSID/IGSID).
type Conversation struct {
	ID         uint   `json:"id" gorm:"primaryKey"`
	Platform   string `json:"platform" gorm:"index:idx_conv_platform_ext,unique;not null"`
	ExternalID string `json:"external_id" gorm:"index:idx_conv_platform_ext,unique;not null"`

	// Display info fetched from the platform's profile API.
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`

	// CustomerID links the thread to a customer record so orders created from
	// chat carry attribution. Nil until the admin links it.
	CustomerID *uint     `json:"customer_id"`
	Customer   *Customer `json:"customer,omitempty" gorm:"foreignKey:CustomerID"`

	UnreadCount     int       `json:"unread_count"`
	LastMessageText string    `json:"last_message_text"`
	LastMessageAt   time.Time `json:"last_message_at"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ChatMessage is one message in a conversation. Direction "in" = from the
// shopper, "out" = sent by the shop (admin reply or, later, auto-reply).
type ChatMessage struct {
	ID             uint   `json:"id" gorm:"primaryKey"`
	ConversationID uint   `json:"conversation_id" gorm:"index;not null"`
	Direction      string `json:"direction" gorm:"not null"` // in | out
	Type           string `json:"type" gorm:"default:text"`  // text | image | sticker | unsupported
	Text           string `json:"text"`
	ImageURL       string `json:"image_url"` // local /uploads/ path for downloaded media
	// ExternalID is the platform's message id — used to dedupe webhook
	// redeliveries.
	ExternalID string    `json:"external_id" gorm:"index"`
	CreatedAt  time.Time `json:"created_at"`
}
