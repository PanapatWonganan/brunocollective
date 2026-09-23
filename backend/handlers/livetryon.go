package handlers

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"brunocollective_inventory/config"
	"brunocollective_inventory/database"
	"brunocollective_inventory/middleware"
	"brunocollective_inventory/models"
	"brunocollective_inventory/services"

	"github.com/gofiber/fiber/v2"
)

// LiveTryOnHandler backs the storefront "ลองใส่สด" (live webcam try-on).
// The heavy lifting happens browser↔Decart over WebRTC; this handler only
// (1) reports whether the feature is on and the caller's budget, and
// (2) mints a client token per session, counting sessions per day (per IP
// for guests, per customer for members) and embedding the per-session
// streaming cap so Decart cuts the stream itself when time is up.
type LiveTryOnHandler struct {
	Config *config.Config
	Client *services.DecartClient

	mu     sync.Mutex
	counts map[string]tryOnCount
}

func NewLiveTryOnHandler(cfg *config.Config, client *services.DecartClient) *LiveTryOnHandler {
	return &LiveTryOnHandler{Config: cfg, Client: client, counts: map[string]tryOnCount{}}
}

func (h *LiveTryOnHandler) enabled() bool { return h.Client != nil && h.Client.Enabled() }

// budget resolves the caller's quota key, sessions/day and seconds/session.
func (h *LiveTryOnHandler) budget(c *fiber.Ctx) (key string, sessions, seconds int, member bool) {
	if id := middleware.OptionalMemberID(c, h.Config); id > 0 {
		return fmt.Sprintf("member:%d", id), h.Config.LiveTryOnMemberSessions, h.Config.LiveTryOnMemberSeconds, true
	}
	ip := strings.TrimSpace(strings.Split(c.Get("X-Forwarded-For"), ",")[0])
	if ip == "" {
		ip = c.IP()
	}
	return "ip:" + ip, h.Config.LiveTryOnDailySessions, h.Config.LiveTryOnSessionSeconds, false
}

func (h *LiveTryOnHandler) remaining(key string, limit int) int {
	if limit <= 0 {
		return 1 << 30
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	cnt, ok := h.counts[key]
	if !ok || cnt.day != bangkokDay() {
		return limit
	}
	if left := limit - cnt.n; left > 0 {
		return left
	}
	return 0
}

func (h *LiveTryOnHandler) consume(key string, limit int) bool {
	if limit <= 0 {
		return true
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	day := bangkokDay()
	cnt := h.counts[key]
	if cnt.day != day {
		cnt = tryOnCount{day: day}
	}
	if cnt.n >= limit {
		return false
	}
	cnt.n++
	h.counts[key] = cnt
	if len(h.counts) > 5000 {
		for k, v := range h.counts {
			if v.day != day {
				delete(h.counts, k)
			}
		}
	}
	return true
}

func (h *LiveTryOnHandler) refund(key string, limit int) {
	if limit <= 0 {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if cnt, ok := h.counts[key]; ok && cnt.n > 0 && cnt.day == bangkokDay() {
		cnt.n--
		h.counts[key] = cnt
	}
}

// Status — GET /api/shop/live-tryon.
func (h *LiveTryOnHandler) Status(c *fiber.Ctx) error {
	if !h.enabled() {
		return c.JSON(fiber.Map{"enabled": false})
	}
	key, sessions, seconds, member := h.budget(c)
	remaining := h.remaining(key, sessions)
	if h.Config.LiveTryOnMembersOnly && !member {
		remaining = 0
	}
	return c.JSON(fiber.Map{
		"enabled":         true,
		"members_only":    h.Config.LiveTryOnMembersOnly,
		"remaining":       remaining,
		"limit":           sessions,
		"session_seconds": seconds,
		"member":          member,
		"model":           h.Config.LiveTryOnModel,
	})
}

// Token — POST /api/shop/live-tryon/token: consume one session from the
// daily budget and return a Decart client token for it.
func (h *LiveTryOnHandler) Token(c *fiber.Ctx) error {
	if !h.enabled() {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "ฟีเจอร์ลองใส่สดยังไม่เปิดใช้งาน"})
	}
	key, sessions, seconds, member := h.budget(c)
	if h.Config.LiveTryOnMembersOnly && !member {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "ลองใส่สดสำหรับสมาชิกเท่านั้น — สมัครสมาชิกฟรีแล้วลองได้เลย", "members_only": true})
	}
	productID, _ := strconv.Atoi(c.FormValue("product_id", c.Query("product_id")))
	if !h.consume(key, sessions) {
		msg := "วันนี้ลองใส่สดครบจำนวนแล้ว พรุ่งนี้ลองใหม่ได้อีกครั้ง"
		if !member {
			msg = "วันนี้ลองใส่สดครบจำนวนแล้ว — สมัครสมาชิกเพื่อลองได้มากขึ้น หรือกลับมาใหม่พรุ่งนี้"
		}
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": msg, "remaining": 0})
	}

	var origins []string
	for _, o := range strings.Split(h.Config.LiveTryOnAllowedOrigins, ",") {
		if o = strings.TrimSpace(o); o != "" {
			origins = append(origins, o)
		}
	}
	// Token TTL only needs to cover connection setup; the WebRTC session
	// itself survives token expiry and is bounded by maxSessionDuration.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	tok, err := h.Client.CreateClientToken(ctx, h.Config.LiveTryOnModel, origins, seconds, 120)
	if err != nil {
		h.refund(key, sessions)
		log.Printf("live-tryon: token failed (%s): %v", key, err)
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "ระบบลองใส่สดขัดข้องชั่วคราว กรุณาลองใหม่อีกครั้ง"})
	}
	sess := models.LiveTryOnSession{QuotaKey: key, ProductID: uint(productID), CapSeconds: seconds}
	if member {
		if id := middleware.OptionalMemberID(c, h.Config); id > 0 {
			sess.CustomerID = &id
		}
	}
	if err := database.DB.Create(&sess).Error; err != nil {
		log.Printf("live-tryon: session row: %v", err)
	}
	log.Printf("live-tryon: session %d for %s (%ds cap, %d left today)", sess.ID, key, seconds, h.remaining(key, sessions))
	return c.JSON(fiber.Map{
		"api_key":         tok.APIKey,
		"expires_at":      tok.ExpiresAt,
		"model":           h.Config.LiveTryOnModel,
		"session_seconds": seconds,
		"remaining":       h.remaining(key, sessions),
		"session_id":      sess.ID,
	})
}

// End — POST /api/shop/live-tryon/sessions/:id/end {seconds, reason}: the
// browser reports how many seconds the SDK says were generated. Accepted
// once per session, clamped to the cap (+5s grace for connection setup).
func (h *LiveTryOnHandler) End(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	var body struct {
		Seconds float64 `json:"seconds"`
		Reason  string  `json:"reason"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	var sess models.LiveTryOnSession
	if err := database.DB.First(&sess, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}
	// Only the caller who opened the session may close it.
	key, _, _, _ := h.budget(c)
	if sess.QuotaKey != key || sess.Reported {
		return c.SendStatus(fiber.StatusNoContent)
	}
	secs := body.Seconds
	if secs < 0 {
		secs = 0
	}
	if max := float64(sess.CapSeconds + 5); secs > max {
		secs = max
	}
	if len(body.Reason) > 32 {
		body.Reason = body.Reason[:32]
	}
	now := time.Now()
	database.DB.Model(&sess).Updates(map[string]interface{}{
		"billed_seconds": secs, "reason": body.Reason, "reported": true, "ended_at": now,
	})
	log.Printf("live-tryon: session %d ended (%s) %.1fs ≈ $%.2f", sess.ID, body.Reason, secs, secs*liveTryOnUSDPerSecond)
	return c.SendStatus(fiber.StatusNoContent)
}

// liveTryOnUSDPerSecond is Decart's published realtime VTON rate (720p).
const liveTryOnUSDPerSecond = 0.02

// Usage — GET /api/live-tryon/usage?days=30 (admin): sessions and streamed
// seconds per day with a cost estimate, for the "is it worth it" question.
// Unreported sessions (tab killed before the end ping) are assumed to have
// run to the cap.
func (h *LiveTryOnHandler) Usage(c *fiber.Ctx) error {
	days, _ := strconv.Atoi(c.Query("days", "30"))
	if days <= 0 || days > 365 {
		days = 30
	}
	since := time.Now().AddDate(0, 0, -days)
	var rows []models.LiveTryOnSession
	database.DB.Where("created_at >= ?", since).Order("created_at ASC").Find(&rows)

	loc, err := time.LoadLocation("Asia/Bangkok")
	if err != nil {
		loc = time.FixedZone("ICT", 7*3600)
	}
	type day struct {
		Date     string  `json:"date"`
		Sessions int     `json:"sessions"`
		Seconds  float64 `json:"seconds"`
		USD      float64 `json:"usd"`
	}
	byDay := map[string]*day{}
	order := []string{}
	var totalSess int
	var totalSec float64
	members := map[uint]bool{}
	for _, r := range rows {
		d := r.CreatedAt.In(loc).Format("2006-01-02")
		if _, ok := byDay[d]; !ok {
			byDay[d] = &day{Date: d}
			order = append(order, d)
		}
		secs := r.BilledSeconds
		if !r.Reported {
			secs = float64(r.CapSeconds)
		}
		byDay[d].Sessions++
		byDay[d].Seconds += secs
		byDay[d].USD = byDay[d].Seconds * liveTryOnUSDPerSecond
		totalSess++
		totalSec += secs
		if r.CustomerID != nil {
			members[*r.CustomerID] = true
		}
	}
	out := make([]day, 0, len(order))
	for _, d := range order {
		out = append(out, *byDay[d])
	}
	return c.JSON(fiber.Map{
		"days":           days,
		"usd_per_second": liveTryOnUSDPerSecond,
		"total_sessions": totalSess,
		"total_seconds":  totalSec,
		"total_usd":      totalSec * liveTryOnUSDPerSecond,
		"unique_members": len(members),
		"avg_seconds": func() float64 {
			if totalSess == 0 {
				return 0
			}
			return totalSec / float64(totalSess)
		}(),
		"per_day":          out,
		"session_cap":      h.Config.LiveTryOnMemberSeconds,
		"sessions_per_day": h.Config.LiveTryOnMemberSessions,
		"members_only":     h.Config.LiveTryOnMembersOnly,
	})
}
