package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"brunocollective_inventory/config"
	"brunocollective_inventory/database"
	"brunocollective_inventory/models"
	"brunocollective_inventory/services"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// WebhookHandler receives platform callbacks (LINE now; Facebook/Instagram
// later) and turns them into Conversation/ChatMessage rows.
type WebhookHandler struct {
	Config *config.Config
	Line   *services.LineClient
	Hub    *services.ChatHub
}

func NewWebhookHandler(cfg *config.Config, line *services.LineClient, hub *services.ChatHub) *WebhookHandler {
	return &WebhookHandler{Config: cfg, Line: line, Hub: hub}
}

// lineEvent is the subset of LINE webhook event fields we consume.
type lineEvent struct {
	Type      string `json:"type"`
	Timestamp int64  `json:"timestamp"`
	Source    struct {
		Type   string `json:"type"`
		UserID string `json:"userId"`
	} `json:"source"`
	Message struct {
		ID        string `json:"id"`
		Type      string `json:"type"`
		Text      string `json:"text"`
		StickerID string `json:"stickerId"`
	} `json:"message"`
}

// LineWebhook handles POST /api/webhooks/line. Always answers 200 for valid
// signatures (LINE retries non-2xx); events it can't process are skipped.
func (h *WebhookHandler) LineWebhook(c *fiber.Ctx) error {
	if !h.Line.Enabled() {
		// Not configured — acknowledge so LINE doesn't hammer retries while
		// the owner is mid-setup, but do nothing.
		return c.SendStatus(fiber.StatusOK)
	}

	body := c.Body()
	if !h.Line.VerifySignature(body, c.Get("X-Line-Signature")) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid signature"})
	}

	var payload struct {
		Events []lineEvent `json:"events"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid payload"})
	}

	for _, ev := range payload.Events {
		if ev.Type != "message" || ev.Source.UserID == "" {
			continue
		}
		h.handleLineMessage(ev)
	}
	return c.SendStatus(fiber.StatusOK)
}

func (h *WebhookHandler) handleLineMessage(ev lineEvent) {
	// Dedupe webhook redeliveries by the platform message id.
	if ev.Message.ID != "" {
		var count int64
		database.DB.Model(&models.ChatMessage{}).
			Where("external_id = ?", ev.Message.ID).Count(&count)
		if count > 0 {
			return
		}
	}

	conv, err := h.findOrCreateConversation("line", ev.Source.UserID)
	if err != nil {
		log.Println("line webhook: conversation error:", err)
		return
	}

	msg := models.ChatMessage{
		ConversationID: conv.ID,
		Direction:      "in",
		ExternalID:     ev.Message.ID,
	}

	switch ev.Message.Type {
	case "text":
		msg.Type = "text"
		msg.Text = ev.Message.Text
	case "image":
		msg.Type = "image"
		if url, err := h.saveLineImage(conv.ID, ev.Message.ID); err == nil {
			msg.ImageURL = url
		} else {
			log.Println("line webhook: image download failed:", err)
			msg.Text = "[รูปภาพ]"
		}
	case "sticker":
		msg.Type = "sticker"
		msg.Text = "[สติกเกอร์]"
	default:
		msg.Type = "unsupported"
		msg.Text = "[" + ev.Message.Type + "]"
	}

	if err := database.DB.Create(&msg).Error; err != nil {
		log.Println("line webhook: save message failed:", err)
		return
	}

	preview := msg.Text
	if msg.Type == "image" && preview == "" {
		preview = "[รูปภาพ]"
	}
	database.DB.Model(conv).Updates(map[string]interface{}{
		"last_message_text": preview,
		"last_message_at":   time.Now(),
		"unread_count":      gorm.Expr("unread_count + 1"),
	})

	h.Hub.Broadcast(fiber.Map{"type": "message", "conversation_id": conv.ID, "message": msg})
}

// findOrCreateConversation returns the thread for a platform user, creating
// it (with a best-effort profile fetch) on first contact.
func (h *WebhookHandler) findOrCreateConversation(platform, externalID string) (*models.Conversation, error) {
	var conv models.Conversation
	err := database.DB.Where("platform = ? AND external_id = ?", platform, externalID).First(&conv).Error
	if err == nil {
		return &conv, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	conv = models.Conversation{
		Platform:      platform,
		ExternalID:    externalID,
		DisplayName:   "LINE User",
		LastMessageAt: time.Now(),
	}
	if profile, err := h.Line.GetProfile(externalID); err == nil {
		if profile.DisplayName != "" {
			conv.DisplayName = profile.DisplayName
		}
		conv.AvatarURL = profile.PictureURL
	}
	if err := database.DB.Create(&conv).Error; err != nil {
		return nil, err
	}
	return &conv, nil
}

// saveLineImage downloads message media and stores it under uploads/,
// returning the public /uploads/ URL.
func (h *WebhookHandler) saveLineImage(convID uint, messageID string) (string, error) {
	data, contentType, err := h.Line.GetMessageContent(messageID)
	if err != nil {
		return "", err
	}
	ext := ".jpg"
	switch {
	case strings.Contains(contentType, "png"):
		ext = ".png"
	case strings.Contains(contentType, "gif"):
		ext = ".gif"
	case strings.Contains(contentType, "webp"):
		ext = ".webp"
	}
	filename := fmt.Sprintf("chat_%d_%d%s", convID, time.Now().UnixNano(), ext)
	if err := os.WriteFile(filepath.Join(h.Config.UploadDir, filename), data, 0644); err != nil {
		return "", err
	}
	return "/uploads/" + filename, nil
}
