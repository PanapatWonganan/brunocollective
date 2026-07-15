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

// WebhookHandler receives platform callbacks (LINE, Facebook Messenger,
// Instagram DM) and turns them into Conversation/ChatMessage rows.
type WebhookHandler struct {
	Config    *config.Config
	Line      *services.LineClient
	Meta      *services.MetaClient
	Hub       *services.ChatHub
	AutoReply *AutoReplyHandler
}

func NewWebhookHandler(cfg *config.Config, line *services.LineClient, meta *services.MetaClient, hub *services.ChatHub, autoReply *AutoReplyHandler) *WebhookHandler {
	return &WebhookHandler{Config: cfg, Line: line, Meta: meta, Hub: hub, AutoReply: autoReply}
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
	touchConversation(conv, preview, "in")

	h.Hub.Broadcast(fiber.Map{"type": "message", "conversation_id": conv.ID, "message": msg})
}

// touchConversation refreshes a thread's list-row fields after a new
// message. Inbound messages reopen done threads and start the waiting
// timer (kept from the oldest unanswered message); outbound ones clear it.
func touchConversation(conv *models.Conversation, preview, direction string) {
	updates := map[string]interface{}{
		"last_message_text": preview,
		"last_message_at":   time.Now(),
		"last_direction":    direction,
	}
	if direction == "in" {
		updates["unread_count"] = gorm.Expr("unread_count + 1")
		updates["status"] = "open"
		if conv.WaitingSince == nil {
			updates["waiting_since"] = time.Now()
		}
	} else {
		updates["waiting_since"] = nil
	}
	database.DB.Model(conv).Updates(updates)
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
		LastMessageAt: time.Now(),
	}
	switch platform {
	case "line":
		conv.DisplayName = "LINE User"
		if profile, err := h.Line.GetProfile(externalID); err == nil {
			if profile.DisplayName != "" {
				conv.DisplayName = profile.DisplayName
			}
			conv.AvatarURL = profile.PictureURL
		}
	case "facebook", "instagram":
		conv.DisplayName = map[string]string{"facebook": "Facebook User", "instagram": "Instagram User"}[platform]
		if profile, err := h.Meta.GetProfile(externalID); err == nil {
			switch {
			case profile.Name != "":
				conv.DisplayName = profile.Name
			case profile.Username != "":
				conv.DisplayName = "@" + profile.Username
			}
			conv.AvatarURL = profile.ProfilePic
		}
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
	return h.saveChatMedia(convID, data, contentType)
}

// saveChatMedia persists downloaded chat media under uploads/ and returns
// its public /uploads/ URL.
func (h *WebhookHandler) saveChatMedia(convID uint, data []byte, contentType string) (string, error) {
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

// ---------------------------------------------------------------------------
// Meta (Facebook Messenger + Instagram DM)
// ---------------------------------------------------------------------------

// metaMessaging is one messaging event inside a Meta webhook entry. The same
// shape serves both `object: "page"` (Messenger) and `object: "instagram"`.
type metaMessaging struct {
	Sender struct {
		ID string `json:"id"`
	} `json:"sender"`
	Recipient struct {
		ID string `json:"id"`
	} `json:"recipient"`
	Timestamp int64 `json:"timestamp"`
	Message   struct {
		MID         string `json:"mid"`
		Text        string `json:"text"`
		IsEcho      bool   `json:"is_echo"`
		Attachments []struct {
			Type    string `json:"type"`
			Payload struct {
				URL string `json:"url"`
			} `json:"payload"`
		} `json:"attachments"`
	} `json:"message"`
}

// MetaWebhookVerify answers Meta's GET subscribe handshake by echoing
// hub.challenge when the verify token matches.
func (h *WebhookHandler) MetaWebhookVerify(c *fiber.Ctx) error {
	if h.Meta.VerifyToken() == "" {
		return c.SendStatus(fiber.StatusForbidden)
	}
	if c.Query("hub.mode") == "subscribe" && c.Query("hub.verify_token") == h.Meta.VerifyToken() {
		return c.SendString(c.Query("hub.challenge"))
	}
	return c.SendStatus(fiber.StatusForbidden)
}

// MetaWebhook handles POST /api/webhooks/meta for both Facebook pages and
// Instagram. Like LINE, valid payloads always get 200 so Meta doesn't
// retry-storm; unprocessable events are skipped.
func (h *WebhookHandler) MetaWebhook(c *fiber.Ctx) error {
	if !h.Meta.Enabled() {
		return c.SendStatus(fiber.StatusOK)
	}

	body := c.Body()
	if !h.Meta.VerifySignature(body, c.Get("X-Hub-Signature-256")) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid signature"})
	}

	var payload struct {
		Object string `json:"object"`
		Entry  []struct {
			ID        string          `json:"id"` // page id / IG account id
			Messaging []metaMessaging `json:"messaging"`
			Changes   []metaChange    `json:"changes"`
		} `json:"entry"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid payload"})
	}

	var platform string
	switch payload.Object {
	case "page":
		platform = "facebook"
	case "instagram":
		platform = "instagram"
	default:
		return c.SendStatus(fiber.StatusOK)
	}

	for _, entry := range payload.Entry {
		for _, ev := range entry.Messaging {
			h.handleMetaMessage(platform, ev)
		}
		for _, ch := range entry.Changes {
			h.handleMetaChange(platform, entry.ID, ch)
		}
	}
	return c.SendStatus(fiber.StatusOK)
}

// metaChange is one entry.changes item — FB page feed events (field
// "feed") and Instagram comment events (field "comments").
type metaChange struct {
	Field string `json:"field"`
	Value struct {
		// Facebook feed fields
		Item      string `json:"item"` // "comment", "post", "reaction", …
		Verb      string `json:"verb"` // "add", "edited", "remove"
		CommentID string `json:"comment_id"`
		Message   string `json:"message"`
		// Instagram comment fields
		ID   string `json:"id"`   // IG comment id
		Text string `json:"text"` // IG comment text
		From struct {
			ID       string `json:"id"`
			Name     string `json:"name"`     // FB
			Username string `json:"username"` // IG
		} `json:"from"`
	} `json:"value"`
}

// handleMetaChange normalizes FB/IG comment events and hands them to the
// auto-reply engine. accountID is the page/IG account id from the entry —
// comments authored by the page itself are skipped (prevents reply loops).
func (h *WebhookHandler) handleMetaChange(platform, accountID string, ch metaChange) {
	ev := CommentEvent{Platform: platform}

	switch {
	case platform == "facebook" && ch.Field == "feed":
		// Only newly added comments — not posts, reactions, edits, deletes.
		if ch.Value.Item != "comment" || ch.Value.Verb != "add" {
			return
		}
		ev.CommentID = ch.Value.CommentID
		ev.Text = ch.Value.Message
		ev.FromID = ch.Value.From.ID
		ev.FromName = ch.Value.From.Name
	case platform == "instagram" && ch.Field == "comments":
		ev.CommentID = ch.Value.ID
		ev.Text = ch.Value.Text
		ev.FromID = ch.Value.From.ID
		ev.FromName = ch.Value.From.Username
	default:
		return
	}

	if ev.CommentID == "" {
		return
	}
	// Never react to the page's own comments (including our own replies —
	// they arrive as webhook events too). This is the loop guard.
	if ev.FromID == "" || ev.FromID == accountID {
		return
	}

	// Answer the webhook fast; actions run in the background.
	go h.AutoReply.HandleComment(ev)
}

func (h *WebhookHandler) handleMetaMessage(platform string, ev metaMessaging) {
	// Delivery/read receipts and postbacks carry no message — skip.
	if ev.Message.MID == "" {
		return
	}
	// Dedupe: webhook redeliveries, and echoes of replies we sent ourselves
	// (the Send API response mid is stored on the outgoing row).
	var count int64
	database.DB.Model(&models.ChatMessage{}).
		Where("external_id = ?", ev.Message.MID).Count(&count)
	if count > 0 {
		return
	}

	// Echo = message sent by the page (admin replied from the FB/IG inbox
	// app, or another tool). The conversation partner is then the recipient.
	direction := "in"
	externalUserID := ev.Sender.ID
	if ev.Message.IsEcho {
		direction = "out"
		externalUserID = ev.Recipient.ID
	}
	if externalUserID == "" {
		return
	}

	conv, err := h.findOrCreateConversation(platform, externalUserID)
	if err != nil {
		log.Println("meta webhook: conversation error:", err)
		return
	}

	msg := models.ChatMessage{
		ConversationID: conv.ID,
		Direction:      direction,
		Type:           "text",
		Text:           ev.Message.Text,
		ExternalID:     ev.Message.MID,
	}
	for _, att := range ev.Message.Attachments {
		if att.Type == "image" && att.Payload.URL != "" {
			msg.Type = "image"
			// CDN URLs expire — persist a local copy, fall back to hotlink.
			if data, ct, err := h.Meta.DownloadMedia(att.Payload.URL); err == nil {
				if url, err := h.saveChatMedia(conv.ID, data, ct); err == nil {
					msg.ImageURL = url
				}
			}
			if msg.ImageURL == "" {
				msg.ImageURL = att.Payload.URL
			}
			break
		}
		if msg.Text == "" {
			msg.Type = "unsupported"
			msg.Text = "[" + att.Type + "]"
		}
	}

	if err := database.DB.Create(&msg).Error; err != nil {
		log.Println("meta webhook: save message failed:", err)
		return
	}

	preview := msg.Text
	if msg.Type == "image" && preview == "" {
		preview = "[รูปภาพ]"
	}
	touchConversation(conv, preview, direction)

	h.Hub.Broadcast(fiber.Map{"type": "message", "conversation_id": conv.ID, "message": msg})
}
