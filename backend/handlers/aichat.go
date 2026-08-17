package handlers

// AI reply path for the chat inbox: when no keyword auto-reply rule matches an
// inbound message, the AI assistant answers from the live catalog — or stays
// silent (handoff) when a human should reply. Like keyword replies, an AI
// reply deliberately does NOT clear waiting_since: the thread stays in the
// "รอตอบ" queue so the admin still glances at every conversation.

import (
	"fmt"
	"log"
	"strings"
	"time"

	"brunocollective_inventory/database"
	"brunocollective_inventory/models"
	"brunocollective_inventory/services"

	"github.com/gofiber/fiber/v2"
)

// buildAIShopContext renders the live catalog (prices + per-variant stock)
// into compact Thai text for the AI system prompt.
func buildAIShopContext() string {
	var products []models.Product
	database.DB.Preload("Variants").Order("display_order ASC, id ASC").Limit(150).Find(&products)

	var sb strings.Builder
	sb.WriteString("## ข้อมูลร้าน: สินค้าและสต็อกล่าสุด\n")
	if len(products) == 0 {
		sb.WriteString("(ยังไม่มีสินค้าในระบบ)\n")
		return sb.String()
	}
	for _, p := range products {
		sb.WriteString(fmt.Sprintf("- %s — ฿%.0f", p.Name, p.Price))
		if p.Category != "" {
			sb.WriteString(" (" + p.Category + ")")
		}
		if len(p.Variants) > 0 {
			parts := make([]string, 0, len(p.Variants))
			for _, v := range p.Variants {
				label := strings.TrimSpace(strings.Trim(v.Size+"/"+v.Color, "/"))
				if label == "" {
					label = "One size"
				}
				if v.Stock > 0 {
					parts = append(parts, fmt.Sprintf("%s เหลือ %d", label, v.Stock))
				} else {
					parts = append(parts, label+" หมด")
				}
			}
			sb.WriteString(" — " + strings.Join(parts, ", "))
		} else if p.Stock > 0 {
			sb.WriteString(fmt.Sprintf(" — เหลือ %d", p.Stock))
		} else {
			sb.WriteString(" — สินค้าหมด")
		}
		sb.WriteString("\n")
	}
	sb.WriteString("\nสินค้าที่เขียนว่า \"หมด\" ให้แจ้งลูกค้าว่าหมดชั่วคราว สอบถามรอบเข้าใหม่กับแอดมินได้\n")
	return sb.String()
}

// tryAIReply answers an inbound chat message with the AI assistant. Called
// from HandleChatMessage when no keyword rule matched; runs in the webhook's
// goroutine, so it can afford the model latency.
func (h *AutoReplyHandler) tryAIReply(conv *models.Conversation, msg *models.ChatMessage) {
	if h.AI == nil || !h.AI.Enabled() || conv.AiDisabled {
		return
	}

	// Conversation history (oldest first) — the newest inbound is included.
	var recent []models.ChatMessage
	database.DB.Where("conversation_id = ?", conv.ID).
		Order("id DESC").Limit(12).Find(&recent)
	turns := make([]services.AIChatTurn, 0, len(recent))
	for i := len(recent) - 1; i >= 0; i-- {
		m := recent[i]
		if m.Type != "text" {
			continue
		}
		turns = append(turns, services.AIChatTurn{Direction: m.Direction, Text: m.Text})
	}

	reply, handoff, err := h.AI.AnswerChat(buildAIShopContext(), turns)

	logRow := models.AutoReplyLog{
		RuleName:    "AI Assistant",
		Platform:    conv.Platform,
		CommentID:   fmt.Sprintf("chat-ai:%d", msg.ID),
		FromID:      conv.ExternalID,
		FromName:    conv.DisplayName,
		CommentText: msg.Text,
	}

	if err != nil {
		logRow.Actions = "ai_reply"
		logRow.Status = "failed"
		logRow.Error = err.Error()
		log.Printf("ai reply failed for conversation %d: %v", conv.ID, err)
		database.DB.Create(&logRow)
		return
	}
	if handoff {
		// The model deferred to a human — record it so the owner can see what
		// the bot chose not to answer; the thread is already in "รอตอบ".
		logRow.Actions = "ai_handoff"
		logRow.Status = "success"
		database.DB.Create(&logRow)
		return
	}

	// Staleness guard: if the customer sent another message while the model
	// was generating, skip — that newer message's own AI pass will answer
	// with the fuller history.
	var newestInbound uint
	database.DB.Model(&models.ChatMessage{}).
		Where("conversation_id = ? AND direction = ?", conv.ID, "in").
		Select("COALESCE(MAX(id), 0)").Scan(&newestInbound)
	if newestInbound != msg.ID {
		return
	}

	var externalID string
	var sendErr error
	switch conv.Platform {
	case "line":
		sendErr = h.Line.PushText(conv.ExternalID, reply)
	case "facebook", "instagram":
		externalID, sendErr = h.Meta.SendText(conv.ExternalID, reply)
	default:
		return
	}

	logRow.Actions = "ai_reply"
	if sendErr != nil {
		logRow.Status = "failed"
		logRow.Error = sendErr.Error()
		log.Printf("ai reply push failed for conversation %d: %v", conv.ID, sendErr)
		database.DB.Create(&logRow)
		return
	}
	logRow.Status = "success"

	out := models.ChatMessage{
		ConversationID: conv.ID,
		Direction:      "out",
		Type:           "text",
		Text:           reply,
		ExternalID:     externalID,
		Source:         "ai",
	}
	if err := database.DB.Create(&out).Error; err == nil {
		// Refresh the preview; keep waiting_since so a human still follows up.
		database.DB.Model(&models.Conversation{}).Where("id = ?", conv.ID).
			Updates(map[string]interface{}{"last_message_text": reply, "last_message_at": time.Now()})
		h.Hub.Broadcast(fiber.Map{"type": "message", "conversation_id": conv.ID, "message": out})
	}
	database.DB.Create(&logRow)
}
