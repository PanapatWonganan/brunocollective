package handlers

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"brunocollective_inventory/config"
	"brunocollective_inventory/middleware"
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
	return c.JSON(fiber.Map{
		"enabled":         true,
		"remaining":       h.remaining(key, sessions),
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
	log.Printf("live-tryon: session for %s (%ds cap, %d left today)", key, seconds, h.remaining(key, sessions))
	return c.JSON(fiber.Map{
		"api_key":         tok.APIKey,
		"expires_at":      tok.ExpiresAt,
		"model":           h.Config.LiveTryOnModel,
		"session_seconds": seconds,
		"remaining":       h.remaining(key, sessions),
	})
}
