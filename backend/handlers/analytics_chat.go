package handlers

// Chat → sales analytics: which channel closes best, how long a chat takes to
// turn into an order, and whether reply speed moves revenue. Chat-created
// orders carry Order.ConversationID (attribution) and every order carries
// Channel, so this is computed straight from existing rows. Datetime math is
// done in Go, not SQL (SQLite MIN/MAX on datetime columns lose the type).

import (
	"sort"
	"strconv"
	"time"

	"brunocollective_inventory/database"
	"brunocollective_inventory/models"

	"github.com/gofiber/fiber/v2"
)

// Chats handles GET /api/analytics/chats?days=30 — the dashboard "แชท" tab.
func (h *AnalyticsHandler) Chats(c *fiber.Ctx) error {
	days, err := strconv.Atoi(c.Query("days", "30"))
	if err != nil || days < 1 || days > 365 {
		days = 30
	}
	since := time.Now().AddDate(0, 0, -days)

	// ── Orders in period (non-cancelled) ──
	type orderRow struct {
		ID             uint
		ConversationID *uint
		Channel        string
		TotalAmount    float64
		CreatedAt      time.Time
	}
	var orders []orderRow
	database.DB.Model(&models.Order{}).
		Where("status != ? AND created_at >= ?", models.StatusCancelled, since).
		Select("id, conversation_id, channel, total_amount, created_at").
		Scan(&orders)

	// Revenue split by channel.
	type chStat struct {
		Orders  int
		Revenue float64
	}
	chStats := make(map[string]*chStat)
	for _, o := range orders {
		ch := o.Channel
		if ch == "" {
			ch = "unknown"
		}
		s := chStats[ch]
		if s == nil {
			s = &chStat{}
			chStats[ch] = s
		}
		s.Orders++
		s.Revenue += o.TotalAmount
	}
	channels := make([]fiber.Map, 0, len(chStats))
	for ch, s := range chStats {
		aov := 0.0
		if s.Orders > 0 {
			aov = s.Revenue / float64(s.Orders)
		}
		channels = append(channels, fiber.Map{
			"channel": ch, "orders": s.Orders, "revenue": s.Revenue, "aov": aov,
		})
	}
	sort.Slice(channels, func(a, b int) bool {
		return channels[a]["revenue"].(float64) > channels[b]["revenue"].(float64)
	})

	// ── Chat attribution: orders created from a conversation ──
	convOrderRevenue := make(map[uint]float64)
	convFirstOrderAt := make(map[uint]time.Time)
	chatOrderCount := 0
	chatRevenue := 0.0
	for _, o := range orders {
		if o.ConversationID == nil {
			continue
		}
		cid := *o.ConversationID
		chatOrderCount++
		chatRevenue += o.TotalAmount
		convOrderRevenue[cid] += o.TotalAmount
		if t, ok := convFirstOrderAt[cid]; !ok || o.CreatedAt.Before(t) {
			convFirstOrderAt[cid] = o.CreatedAt
		}
	}

	// ── Conversations + messages in period ──
	var convs []models.Conversation
	database.DB.Find(&convs)
	convByID := make(map[uint]models.Conversation, len(convs))
	for _, cv := range convs {
		convByID[cv.ID] = cv
	}

	type msgRow struct {
		ConversationID uint
		Direction      string
		CreatedAt      time.Time
	}
	var msgs []msgRow
	database.DB.Model(&models.ChatMessage{}).
		Where("created_at >= ?", since).
		Order("conversation_id ASC, id ASC").
		Select("conversation_id, direction, created_at").
		Scan(&msgs)

	// Active conversations (≥1 inbound this period) and first-response gaps:
	// each unanswered inbound run → next outbound = one response sample.
	activeConvs := make(map[uint]bool)
	type respAgg struct {
		total time.Duration
		n     int
	}
	resp := make(map[uint]*respAgg)
	pendingIn := make(map[uint]time.Time)
	for _, m := range msgs {
		switch m.Direction {
		case "in":
			activeConvs[m.ConversationID] = true
			if _, waiting := pendingIn[m.ConversationID]; !waiting {
				pendingIn[m.ConversationID] = m.CreatedAt
			}
		case "out":
			if start, waiting := pendingIn[m.ConversationID]; waiting {
				a := resp[m.ConversationID]
				if a == nil {
					a = &respAgg{}
					resp[m.ConversationID] = a
				}
				a.total += m.CreatedAt.Sub(start)
				a.n++
				delete(pendingIn, m.ConversationID)
			}
		}
	}

	chatConversion := 0.0
	if len(activeConvs) > 0 {
		chatConversion = float64(len(convFirstOrderAt)) / float64(len(activeConvs)) * 100
	}

	// ── Time to close: first chat contact → first order of that conversation ──
	ttcBuckets := map[string]int{"within_1h": 0, "within_1d": 0, "within_3d": 0, "over_3d": 0}
	ttcHours := make([]float64, 0, len(convFirstOrderAt))
	for cid, orderAt := range convFirstOrderAt {
		cv, ok := convByID[cid]
		if !ok {
			continue
		}
		hrs := orderAt.Sub(cv.CreatedAt).Hours()
		if hrs < 0 {
			hrs = 0
		}
		ttcHours = append(ttcHours, hrs)
		switch {
		case hrs <= 1:
			ttcBuckets["within_1h"]++
		case hrs <= 24:
			ttcBuckets["within_1d"]++
		case hrs <= 72:
			ttcBuckets["within_3d"]++
		default:
			ttcBuckets["over_3d"]++
		}
	}
	medianTTC := 0.0
	if len(ttcHours) > 0 {
		sort.Float64s(ttcHours)
		medianTTC = ttcHours[len(ttcHours)/2]
	}

	// ── Reply speed vs conversion: bucket conversations by their average
	// first-response time, then compare order rates between buckets ──
	type speedBucket struct {
		Key           string
		Label         string
		Conversations int
		WithOrder     int
		Revenue       float64
	}
	speedBuckets := []*speedBucket{
		{Key: "fast", Label: "≤ 10 นาที"},
		{Key: "medium", Label: "10-60 นาที"},
		{Key: "slow", Label: "1-24 ชม."},
		{Key: "very_slow", Label: "> 24 ชม."},
	}
	respMinutes := make([]float64, 0, len(resp))
	for cid, a := range resp {
		avgMin := a.total.Minutes() / float64(a.n)
		respMinutes = append(respMinutes, avgMin)
		var b *speedBucket
		switch {
		case avgMin <= 10:
			b = speedBuckets[0]
		case avgMin <= 60:
			b = speedBuckets[1]
		case avgMin <= 1440:
			b = speedBuckets[2]
		default:
			b = speedBuckets[3]
		}
		b.Conversations++
		if rev, ok := convOrderRevenue[cid]; ok {
			b.WithOrder++
			b.Revenue += rev
		}
	}
	medianResp := 0.0
	if len(respMinutes) > 0 {
		sort.Float64s(respMinutes)
		medianResp = respMinutes[len(respMinutes)/2]
	}
	speedOut := make([]fiber.Map, 0, len(speedBuckets))
	for _, b := range speedBuckets {
		conv := 0.0
		if b.Conversations > 0 {
			conv = float64(b.WithOrder) / float64(b.Conversations) * 100
		}
		speedOut = append(speedOut, fiber.Map{
			"key": b.Key, "label": b.Label,
			"conversations": b.Conversations, "with_order": b.WithOrder,
			"conversion_pct": conv, "revenue": b.Revenue,
		})
	}

	// ── Bot activity this period (successful runs) ──
	type botRow struct {
		Actions string
		N       int
	}
	var botRows []botRow
	database.DB.Model(&models.AutoReplyLog{}).
		Where("created_at >= ? AND status = ?", since, "success").
		Select("actions, COUNT(*) as n").
		Group("actions").
		Scan(&botRows)
	bot := fiber.Map{"rule_replies": 0, "ai_replies": 0, "ai_handoffs": 0}
	for _, r := range botRows {
		switch r.Actions {
		case "chat_reply":
			bot["rule_replies"] = r.N
		case "ai_reply":
			bot["ai_replies"] = r.N
		case "ai_handoff":
			bot["ai_handoffs"] = r.N
		}
	}

	// ── Open deals snapshot (all time): chat orders still awaiting payment ──
	type pendAgg struct {
		N     int
		Value float64
	}
	var pend pendAgg
	database.DB.Model(&models.Order{}).
		Where("conversation_id IS NOT NULL AND status = ? AND (slip_image = '' OR slip_image IS NULL)",
			models.StatusPending).
		Select("COUNT(*) as n, COALESCE(SUM(total_amount), 0) as value").
		Scan(&pend)

	return c.JSON(fiber.Map{
		"days":                 days,
		"channels":             channels,
		"active_conversations": len(activeConvs),
		"chat_orders":          chatOrderCount,
		"chat_revenue":         chatRevenue,
		"chat_conversion_pct":  chatConversion,
		"median_response_min":  medianResp,
		"response_buckets":     speedOut,
		"time_to_close": fiber.Map{
			"median_hours": medianTTC,
			"buckets":      ttcBuckets,
			"total":        len(ttcHours),
		},
		"bot":           bot,
		"pending_deals": fiber.Map{"count": pend.N, "value": pend.Value},
	})
}
