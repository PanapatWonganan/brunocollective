package handlers

import (
	"log"
	"strconv"
	"strings"

	"brunocollective_inventory/database"
	"brunocollective_inventory/models"
	"brunocollective_inventory/services"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// AutoReplyHandler manages comment auto-reply rules and runs the matching
// engine for incoming FB/IG comment webhooks.
type AutoReplyHandler struct {
	Meta *services.MetaClient
}

func NewAutoReplyHandler(meta *services.MetaClient) *AutoReplyHandler {
	return &AutoReplyHandler{Meta: meta}
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
	if rule.Platform != "all" && rule.Platform != "facebook" && rule.Platform != "instagram" {
		return "platform ต้องเป็น all, facebook หรือ instagram"
	}
	if rule.ReplyText == "" && rule.PrivateReplyText == "" && !rule.HideComment {
		return "ต้องเลือกอย่างน้อย 1 การทำงาน (ตอบคอมเมนต์ / ส่ง DM / ซ่อนคอมเมนต์)"
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
