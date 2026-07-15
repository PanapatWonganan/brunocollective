package handlers

import (
	"strconv"

	"brunocollective_inventory/config"
	"brunocollective_inventory/database"
	"brunocollective_inventory/models"
	"brunocollective_inventory/services"

	"github.com/gofiber/fiber/v2"
)

// ChatHandler serves the admin chat inbox: conversation list, message
// history, replies (pushed to the platform), read state, and customer
// linking.
type ChatHandler struct {
	Config *config.Config
	Line   *services.LineClient
	Meta   *services.MetaClient
	Hub    *services.ChatHub
}

func NewChatHandler(cfg *config.Config, line *services.LineClient, meta *services.MetaClient, hub *services.ChatHub) *ChatHandler {
	return &ChatHandler{Config: cfg, Line: line, Meta: meta, Hub: hub}
}

// List returns all conversations, most recently active first.
func (h *ChatHandler) List(c *fiber.Ctx) error {
	var convs []models.Conversation
	database.DB.Preload("Customer").Order("last_message_at DESC").Find(&convs)
	return c.JSON(convs)
}

// Messages returns a conversation's messages oldest-first.
func (h *ChatHandler) Messages(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var conv models.Conversation
	if err := database.DB.Preload("Customer").First(&conv, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "conversation not found"})
	}
	var messages []models.ChatMessage
	database.DB.Where("conversation_id = ?", conv.ID).Order("created_at ASC").Find(&messages)
	return c.JSON(fiber.Map{"conversation": conv, "messages": messages})
}

// Reply pushes a text message to the shopper on their platform, then records
// it. The message is only saved when the platform accepted it.
func (h *ChatHandler) Reply(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var body struct {
		Text string `json:"text"`
	}
	if err := c.BodyParser(&body); err != nil || body.Text == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "text is required"})
	}

	var conv models.Conversation
	if err := database.DB.First(&conv, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "conversation not found"})
	}

	// externalMsgID (Meta only) is stored so the echo event that follows the
	// send is deduped instead of double-inserting the reply.
	var externalMsgID string
	switch conv.Platform {
	case "line":
		if err := h.Line.PushText(conv.ExternalID, body.Text); err != nil {
			return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": err.Error()})
		}
	case "facebook", "instagram":
		mid, err := h.Meta.SendText(conv.ExternalID, body.Text)
		if err != nil {
			return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": err.Error()})
		}
		externalMsgID = mid
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "platform ยังไม่รองรับการตอบกลับ: " + conv.Platform})
	}

	msg := models.ChatMessage{
		ConversationID: conv.ID,
		Direction:      "out",
		Type:           "text",
		Text:           body.Text,
		ExternalID:     externalMsgID,
	}
	database.DB.Create(&msg)
	// A reply also reopens a "done" thread — you're clearly talking again.
	database.DB.Model(&conv).Update("status", "open")
	touchConversation(&conv, body.Text, "out")

	h.Hub.Broadcast(fiber.Map{"type": "message", "conversation_id": conv.ID, "message": msg})
	return c.Status(fiber.StatusCreated).JSON(msg)
}

// MarkRead zeroes the unread counter.
func (h *ChatHandler) MarkRead(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	if err := database.DB.Model(&models.Conversation{}).Where("id = ?", id).
		Update("unread_count", 0).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update"})
	}
	return c.JSON(fiber.Map{"message": "ok"})
}

// Summary returns lightweight counters for the sidebar badge: threads
// waiting for a reply and total unread messages.
func (h *ChatHandler) Summary(c *fiber.Ctx) error {
	var waiting int64
	database.DB.Model(&models.Conversation{}).
		Where("status = ? AND last_direction = ?", "open", "in").
		Count(&waiting)
	var unread int64
	database.DB.Model(&models.Conversation{}).
		Select("COALESCE(SUM(unread_count), 0)").Scan(&unread)
	return c.JSON(fiber.Map{"waiting": waiting, "unread": unread})
}

// UpdateStatus flips a thread between open and done.
func (h *ChatHandler) UpdateStatus(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := c.BodyParser(&body); err != nil || (body.Status != "open" && body.Status != "done") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "status must be open or done"})
	}
	updates := map[string]interface{}{"status": body.Status}
	if body.Status == "done" {
		// Closing a thread also clears the waiting indicator and unread.
		updates["waiting_since"] = nil
		updates["unread_count"] = 0
	}
	if err := database.DB.Model(&models.Conversation{}).Where("id = ?", id).
		Updates(updates).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update"})
	}
	var conv models.Conversation
	database.DB.Preload("Customer").First(&conv, id)
	return c.JSON(conv)
}

// UpdateTags replaces a thread's label set.
func (h *ChatHandler) UpdateTags(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var body struct {
		Tags []string `json:"tags"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if err := database.DB.Model(&models.Conversation{}).Where("id = ?", id).
		Update("tags", models.StringSlice(body.Tags)).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update"})
	}
	var conv models.Conversation
	database.DB.Preload("Customer").First(&conv, id)
	return c.JSON(conv)
}

// ── Canned replies (ข้อความสำเร็จรูป) ──

func (h *ChatHandler) CannedList(c *fiber.Ctx) error {
	var replies []models.CannedReply
	database.DB.Order("title ASC").Find(&replies)
	return c.JSON(replies)
}

func (h *ChatHandler) CannedCreate(c *fiber.Ctx) error {
	var reply models.CannedReply
	if err := c.BodyParser(&reply); err != nil || reply.Title == "" || reply.Text == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "title and text are required"})
	}
	reply.ID = 0
	if err := database.DB.Create(&reply).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create"})
	}
	return c.Status(fiber.StatusCreated).JSON(reply)
}

func (h *ChatHandler) CannedUpdate(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var existing models.CannedReply
	if err := database.DB.First(&existing, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}
	var body models.CannedReply
	if err := c.BodyParser(&body); err != nil || body.Title == "" || body.Text == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "title and text are required"})
	}
	database.DB.Model(&existing).Updates(map[string]interface{}{"title": body.Title, "text": body.Text})
	return c.JSON(existing)
}

func (h *ChatHandler) CannedDelete(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	database.DB.Delete(&models.CannedReply{}, id)
	return c.JSON(fiber.Map{"message": "deleted"})
}

// LinkCustomer attaches (or detaches, with customer_id 0/null) a customer
// record to the conversation.
func (h *ChatHandler) LinkCustomer(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var body struct {
		CustomerID *uint `json:"customer_id"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	var conv models.Conversation
	if err := database.DB.First(&conv, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "conversation not found"})
	}

	if body.CustomerID != nil && *body.CustomerID != 0 {
		var customer models.Customer
		if err := database.DB.First(&customer, *body.CustomerID).Error; err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "customer not found"})
		}
		database.DB.Model(&conv).Update("customer_id", *body.CustomerID)
	} else {
		database.DB.Model(&conv).Update("customer_id", nil)
	}

	database.DB.Preload("Customer").First(&conv, conv.ID)
	return c.JSON(conv)
}
