package handlers

import (
	"log"
	"sort"
	"strings"
	"time"

	"brunocollective_inventory/database"
	"brunocollective_inventory/models"
	"brunocollective_inventory/services"

	"github.com/gofiber/fiber/v2"
)

// BroadcastHandler sends campaign messages to LINE chat contacts, grouped by
// the same RFM segments the analytics dashboard uses. LINE only, deliberately:
// Meta blocks promotional pushes outside the 24-hour messaging window, so
// FB/IG contacts are excluded. Note LINE OA message quota applies — pushes
// beyond the plan's free allowance are billed by LINE.
type BroadcastHandler struct {
	Line *services.LineClient
}

func NewBroadcastHandler(line *services.LineClient) *BroadcastHandler {
	return &BroadcastHandler{Line: line}
}

// broadcastRecipient is one reachable LINE contact with its audience bucket.
type broadcastRecipient struct {
	ConversationID uint    `json:"conversation_id"`
	ExternalID     string  `json:"-"`
	CustomerID     uint    `json:"customer_id"` // 0 = conversation not linked
	Name           string  `json:"name"`
	Segment        string  `json:"segment"`
	Orders         int     `json:"orders"`
	Spent          float64 `json:"spent"`
}

// Audience display order: best customers first, then win-back targets, then
// the two special buckets for contacts without purchase history.
var broadcastSegmentOrder = []string{
	"VIP", "ขาประจำ", "ลูกค้าใหม่", "ทั่วไป", "กำลังจะหาย", "หายไปแล้ว",
	"ยังไม่เคยซื้อ", "ยังไม่ผูกลูกค้า",
}

// broadcastAudience lists every LINE conversation bucketed by the linked
// customer's RFM segment (same segmentOf as the analytics dashboard).
// Unlinked conversations and customers without orders get their own buckets.
func broadcastAudience() []broadcastRecipient {
	var convs []models.Conversation
	database.DB.Preload("Customer").Where("platform = ?", "line").Find(&convs)
	if len(convs) == 0 {
		return nil
	}

	type orderRow struct {
		CustomerID  uint
		TotalAmount float64
		CreatedAt   time.Time
	}
	var rows []orderRow
	database.DB.Model(&models.Order{}).
		Where("status != ?", models.StatusCancelled).
		Select("customer_id, total_amount, created_at").
		Scan(&rows)

	type agg struct {
		Orders int
		Spent  float64
		Last   time.Time
	}
	aggByID := make(map[uint]*agg)
	for _, r := range rows {
		a, ok := aggByID[r.CustomerID]
		if !ok {
			a = &agg{}
			aggByID[r.CustomerID] = a
		}
		a.Orders++
		a.Spent += r.TotalAmount
		if r.CreatedAt.After(a.Last) {
			a.Last = r.CreatedAt
		}
	}

	now := time.Now()
	out := make([]broadcastRecipient, 0, len(convs))
	for _, cv := range convs {
		r := broadcastRecipient{ConversationID: cv.ID, ExternalID: cv.ExternalID, Name: cv.DisplayName}
		if cv.CustomerID == nil {
			r.Segment = "ยังไม่ผูกลูกค้า"
		} else {
			r.CustomerID = *cv.CustomerID
			if cv.Customer != nil && cv.Customer.Name != "" {
				r.Name = cv.Customer.Name
			}
			if a, ok := aggByID[*cv.CustomerID]; ok {
				r.Orders = a.Orders
				r.Spent = a.Spent
				r.Segment = segmentOf(int(now.Sub(a.Last).Hours()/24), a.Orders)
			} else {
				r.Segment = "ยังไม่เคยซื้อ"
			}
		}
		out = append(out, r)
	}
	return out
}

// Audience handles GET /api/broadcasts/audience — segment buckets with their
// reachable LINE contacts (big spenders first inside each bucket).
func (h *BroadcastHandler) Audience(c *fiber.Ctx) error {
	recipients := broadcastAudience()
	bySeg := make(map[string][]broadcastRecipient)
	for _, r := range recipients {
		bySeg[r.Segment] = append(bySeg[r.Segment], r)
	}

	segments := make([]fiber.Map, 0, len(bySeg))
	for _, seg := range broadcastSegmentOrder {
		list := bySeg[seg]
		if len(list) == 0 {
			continue
		}
		sort.Slice(list, func(a, b int) bool { return list[a].Spent > list[b].Spent })
		segments = append(segments, fiber.Map{"segment": seg, "count": len(list), "recipients": list})
	}
	return c.JSON(fiber.Map{"line_enabled": h.Line.Enabled(), "segments": segments})
}

type broadcastRequest struct {
	Segments []string `json:"segments"`
	Message  string   `json:"message"`
}

// Send handles POST /api/broadcasts. Creates the campaign row and pushes in a
// background goroutine (Sent/Failed tick up for the UI to poll); responds 201
// immediately with the pending row.
func (h *BroadcastHandler) Send(c *fiber.Ctx) error {
	if !h.Line.Enabled() {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ยังไม่ได้ตั้งค่า LINE (LINE_CHANNEL_ACCESS_TOKEN)"})
	}

	var req broadcastRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if strings.TrimSpace(req.Message) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "กรุณาพิมพ์ข้อความที่จะส่ง"})
	}
	if len(req.Segments) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "เลือกกลุ่มเป้าหมายอย่างน้อย 1 กลุ่ม"})
	}

	selected := make(map[string]bool, len(req.Segments))
	for _, s := range req.Segments {
		selected[s] = true
	}
	var targets []broadcastRecipient
	for _, r := range broadcastAudience() {
		if selected[r.Segment] {
			targets = append(targets, r)
		}
	}
	if len(targets) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ไม่มีผู้รับในกลุ่มที่เลือก"})
	}

	bc := models.Broadcast{
		Message:  req.Message,
		Segments: models.StringSlice(req.Segments),
		Total:    len(targets),
		Status:   "sending",
	}
	if err := database.DB.Create(&bc).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create broadcast"})
	}

	go h.runBroadcast(bc.ID, req.Message, targets)

	return c.Status(fiber.StatusCreated).JSON(bc)
}

func (h *BroadcastHandler) runBroadcast(id uint, message string, targets []broadcastRecipient) {
	sent, failed := 0, 0
	for _, t := range targets {
		text := renderTemplate(message, t.Name)
		if err := h.Line.PushText(t.ExternalID, text); err != nil {
			failed++
			log.Printf("broadcast %d: push to conversation %d failed: %v", id, t.ConversationID, err)
		} else {
			sent++
			database.DB.Create(&models.ChatMessage{
				ConversationID: t.ConversationID,
				Direction:      "out",
				Type:           "text",
				Text:           text,
				Source:         "broadcast",
			})
			// Refresh the thread preview only — a broadcast must not clear a
			// waiting-for-reply state.
			database.DB.Model(&models.Conversation{}).Where("id = ?", t.ConversationID).
				Updates(map[string]interface{}{"last_message_text": text, "last_message_at": time.Now()})
		}
		database.DB.Model(&models.Broadcast{}).Where("id = ?", id).
			Updates(map[string]interface{}{"sent": sent, "failed": failed})
		time.Sleep(150 * time.Millisecond) // gentle pacing for the LINE API
	}
	database.DB.Model(&models.Broadcast{}).Where("id = ?", id).Update("status", "done")
	log.Printf("broadcast %d done: %d sent, %d failed", id, sent, failed)
}

// List handles GET /api/broadcasts — newest campaigns first.
func (h *BroadcastHandler) List(c *fiber.Ctx) error {
	var list []models.Broadcast
	database.DB.Order("created_at DESC").Limit(50).Find(&list)
	return c.JSON(list)
}
