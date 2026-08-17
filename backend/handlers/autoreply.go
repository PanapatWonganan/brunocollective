package handlers

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"brunocollective_inventory/database"
	"brunocollective_inventory/models"
	"brunocollective_inventory/services"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// AutoReplyHandler manages auto-reply rules and runs the matching engine for
// incoming FB/IG comment webhooks and (rules with ApplyToChats) inbound chat
// messages on LINE / FB Messenger / IG DM.
type AutoReplyHandler struct {
	Meta *services.MetaClient
	Line *services.LineClient
	Hub  *services.ChatHub
	AI   *services.AIClient
}

func NewAutoReplyHandler(meta *services.MetaClient, line *services.LineClient, hub *services.ChatHub, ai *services.AIClient) *AutoReplyHandler {
	return &AutoReplyHandler{Meta: meta, Line: line, Hub: hub, AI: ai}
}

// ── CRUD ──

func (h *AutoReplyHandler) List(c *fiber.Ctx) error {
	var rules []models.AutoReplyRule
	database.DB.Order("priority ASC, id ASC").Find(&rules)
	return c.JSON(rules)
}

func (h *AutoReplyHandler) Create(c *fiber.Ctx) error {
	var rule models.AutoReplyRule
	if err := c.BodyParser(&rule); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if err := validateRule(&rule); err != "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err})
	}
	rule.ID = 0
	rule.UsageCount = 0
	if err := database.DB.Create(&rule).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create rule"})
	}
	return c.Status(fiber.StatusCreated).JSON(rule)
}

func (h *AutoReplyHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var existing models.AutoReplyRule
	if err := database.DB.First(&existing, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "rule not found"})
	}
	var body models.AutoReplyRule
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if errMsg := validateRule(&body); errMsg != "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": errMsg})
	}
	database.DB.Model(&existing).Updates(map[string]interface{}{
		"name":               body.Name,
		"platform":           body.Platform,
		"keywords":           body.Keywords,
		"enabled":            body.Enabled,
		"priority":           body.Priority,
		"reply_text":         body.ReplyText,
		"private_reply_text": body.PrivateReplyText,
		"hide_comment":       body.HideComment,
		"apply_to_chats":     body.ApplyToChats,
	})
	return c.JSON(existing)
}

func (h *AutoReplyHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	database.DB.Delete(&models.AutoReplyRule{}, id)
	return c.JSON(fiber.Map{"message": "deleted"})
}

func (h *AutoReplyHandler) Toggle(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var rule models.AutoReplyRule
	if err := database.DB.First(&rule, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "rule not found"})
	}
	database.DB.Model(&rule).Update("enabled", !rule.Enabled)
	return c.JSON(rule)
}

// Logs returns the newest 100 engine runs.
func (h *AutoReplyHandler) Logs(c *fiber.Ctx) error {
	var logs []models.AutoReplyLog
	database.DB.Order("created_at DESC").Limit(100).Find(&logs)
	return c.JSON(logs)
}

func validateRule(rule *models.AutoReplyRule) string {
	if strings.TrimSpace(rule.Name) == "" {
		return "ต้องตั้งชื่อกฎ"
	}
	if rule.Platform != "all" && rule.Platform != "facebook" && rule.Platform != "instagram" && rule.Platform != "line" {
		return "platform ต้องเป็น all, facebook, instagram หรือ line"
	}
	// LINE has no public comments — a line-only rule is chat-only by nature.
	if rule.Platform == "line" && !rule.ApplyToChats {
		return "กฎสำหรับ LINE ใช้ได้กับแชทเท่านั้น — เปิด \"ตอบแชทอัตโนมัติ\" ด้วย"
	}
	if rule.ReplyText == "" && rule.PrivateReplyText == "" && !rule.HideComment {
		return "ต้องเลือกอย่างน้อย 1 การทำงาน (ตอบคอมเมนต์ / ส่ง DM / ซ่อนคอมเมนต์)"
	}
	if rule.ApplyToChats && rule.ReplyText == "" {
		return "ตอบแชทอัตโนมัติใช้ข้อความจากช่อง 1) — กรุณากรอกข้อความตอบ"
	}
	return ""
}

// ── Engine ──

// CommentEvent is a normalized incoming comment from either platform.
type CommentEvent struct {
	Platform  string
	CommentID string
	Text      string
	FromID    string
	FromName  string
}

// HandleComment matches the first enabled rule and executes its actions.
// Runs in its own goroutine (webhooks must answer fast) — errors land in
// the log table, not the webhook response.
func (h *AutoReplyHandler) HandleComment(ev CommentEvent) {
	// Dedupe webhook redeliveries: one log row per comment id.
	var count int64
	database.DB.Model(&models.AutoReplyLog{}).
		Where("comment_id = ?", ev.CommentID).Count(&count)
	if count > 0 {
		return
	}

	var rules []models.AutoReplyRule
	database.DB.Where("enabled = ?", true).
		Where("platform = ? OR platform = ?", ev.Platform, "all").
		Order("priority ASC, id ASC").
		Find(&rules)

	text := strings.ToLower(ev.Text)
	var matched *models.AutoReplyRule
	for i := range rules {
		if ruleMatches(&rules[i], text) {
			matched = &rules[i]
			break
		}
	}
	if matched == nil {
		return
	}

	logRow := models.AutoReplyLog{
		RuleID:      matched.ID,
		RuleName:    matched.Name,
		Platform:    ev.Platform,
		CommentID:   ev.CommentID,
		FromID:      ev.FromID,
		FromName:    ev.FromName,
		CommentText: ev.Text,
	}

	var actions []string
	var failures []string
	run := func(name string, fn func() error) {
		actions = append(actions, name)
		if err := fn(); err != nil {
			failures = append(failures, name+": "+err.Error())
			log.Printf("auto-reply %s failed for comment %s: %v", name, ev.CommentID, err)
		}
	}

	if matched.ReplyText != "" {
		replyText := renderTemplate(matched.ReplyText, ev.FromName)
		run("reply", func() error { return h.Meta.ReplyToComment(ev.Platform, ev.CommentID, replyText) })
	}
	if matched.PrivateReplyText != "" {
		dmText := renderTemplate(matched.PrivateReplyText, ev.FromName)
		run("private_reply", func() error { return h.Meta.PrivateReply(ev.CommentID, dmText) })
	}
	if matched.HideComment {
		run("hide", func() error { return h.Meta.HideComment(ev.Platform, ev.CommentID) })
	}

	logRow.Actions = strings.Join(actions, ",")
	switch {
	case len(failures) == 0:
		logRow.Status = "success"
	case len(failures) == len(actions):
		logRow.Status = "failed"
	default:
		logRow.Status = "partial"
	}
	if len(failures) > 0 {
		logRow.Error = failures[0]
	}
	database.DB.Create(&logRow)
	database.DB.Model(&models.AutoReplyRule{}).Where("id = ?", matched.ID).
		Update("usage_count", gorm.Expr("usage_count + 1"))
}

// chatAutoReplyCooldown keeps the same rule from replying twice in a row when
// a customer sends several matching messages back-to-back.
const chatAutoReplyCooldown = 10 * time.Minute

// HandleChatMessage runs auto-reply rules (ApplyToChats only) against an
// inbound chat message. Runs in a goroutine from the webhook handlers.
// The reply is recorded as a normal outbound ChatMessage, but the thread's
// waiting state is deliberately left untouched — a bot answer still deserves
// a human look, so the conversation stays in the "รอตอบ" queue.
func (h *AutoReplyHandler) HandleChatMessage(conv *models.Conversation, msg *models.ChatMessage) {
	if msg.Direction != "in" || msg.Type != "text" || strings.TrimSpace(msg.Text) == "" {
		return
	}

	var rules []models.AutoReplyRule
	database.DB.Where("enabled = ? AND apply_to_chats = ?", true, true).
		Where("platform = ? OR platform = ?", conv.Platform, "all").
		Order("priority ASC, id ASC").
		Find(&rules)

	text := strings.ToLower(msg.Text)
	var matched *models.AutoReplyRule
	for i := range rules {
		if rules[i].ReplyText != "" && ruleMatches(&rules[i], text) {
			matched = &rules[i]
			break
		}
	}
	if matched == nil {
		// No keyword rule — let the AI assistant take a shot (no-op when
		// disabled globally or for this thread).
		h.tryAIReply(conv, msg)
		return
	}

	// Cooldown: skip when this rule already replied to this person recently.
	var recent int64
	database.DB.Model(&models.AutoReplyLog{}).
		Where("rule_id = ? AND from_id = ? AND created_at > ?",
			matched.ID, conv.ExternalID, time.Now().Add(-chatAutoReplyCooldown)).
		Count(&recent)
	if recent > 0 {
		return
	}

	replyText := renderTemplate(matched.ReplyText, conv.DisplayName)

	var externalID string
	var sendErr error
	switch conv.Platform {
	case "line":
		sendErr = h.Line.PushText(conv.ExternalID, replyText)
	case "facebook", "instagram":
		externalID, sendErr = h.Meta.SendText(conv.ExternalID, replyText)
	default:
		return
	}

	logRow := models.AutoReplyLog{
		RuleID:      matched.ID,
		RuleName:    matched.Name,
		Platform:    conv.Platform,
		CommentID:   fmt.Sprintf("chat:%d", msg.ID),
		FromID:      conv.ExternalID,
		FromName:    conv.DisplayName,
		CommentText: msg.Text,
		Actions:     "chat_reply",
		Status:      "success",
	}
	if sendErr != nil {
		logRow.Status = "failed"
		logRow.Error = sendErr.Error()
		log.Printf("chat auto-reply failed for conversation %d: %v", conv.ID, sendErr)
		database.DB.Create(&logRow)
		return
	}

	// Save only after the platform accepted — same contract as admin replies.
	out := models.ChatMessage{
		ConversationID: conv.ID,
		Direction:      "out",
		Type:           "text",
		Text:           replyText,
		ExternalID:     externalID,
		Source:         "rule",
	}
	if err := database.DB.Create(&out).Error; err == nil {
		// Refresh the list preview but keep waiting_since / last_direction —
		// see the note in the function comment.
		database.DB.Model(conv).Updates(map[string]interface{}{
			"last_message_text": replyText,
			"last_message_at":   time.Now(),
		})
		h.Hub.Broadcast(fiber.Map{"type": "message", "conversation_id": conv.ID, "message": out})
	}

	database.DB.Create(&logRow)
	database.DB.Model(&models.AutoReplyRule{}).Where("id = ?", matched.ID).
		Update("usage_count", gorm.Expr("usage_count + 1"))
}

// ruleMatches reports whether the (lowercased) comment text hits any of the
// rule's keywords. No keywords = catch-all.
func ruleMatches(rule *models.AutoReplyRule, lowerText string) bool {
	if len(rule.Keywords) == 0 {
		return true
	}
	for _, kw := range rule.Keywords {
		kw = strings.ToLower(strings.TrimSpace(kw))
		if kw != "" && strings.Contains(lowerText, kw) {
			return true
		}
	}
	return false
}

// renderTemplate substitutes template variables ({name} = commenter).
func renderTemplate(text, name string) string {
	if name == "" {
		name = "คุณลูกค้า"
	}
	return strings.ReplaceAll(text, "{name}", name)
}
