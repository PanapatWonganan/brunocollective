package handlers

import (
	"fmt"
	"html"
	"log"
	"time"

	"brunocollective_inventory/config"
	"brunocollective_inventory/database"
	"brunocollective_inventory/models"
	"brunocollective_inventory/services"
)

// StartChatSLAWatcher alerts the Telegram group when chats have been waiting
// for a reply longer than cfg.ChatSLAMinutes (0 = disabled). One alert per
// waiting period: SlaAlertedAt is stamped on alert, and replying clears
// WaitingSince, so only a NEW unanswered inbound triggers the next alert.
func StartChatSLAWatcher(cfg *config.Config, telegram *services.TelegramNotifier) {
	if cfg.ChatSLAMinutes <= 0 {
		log.Println("Chat SLA alerts disabled (CHAT_SLA_MINUTES=0)")
		return
	}
	log.Printf("Chat SLA alerts enabled (threshold %d min)", cfg.ChatSLAMinutes)
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			checkChatSLA(cfg, telegram)
		}
	}()
}

func checkChatSLA(cfg *config.Config, telegram *services.TelegramNotifier) {
	cutoff := time.Now().Add(-time.Duration(cfg.ChatSLAMinutes) * time.Minute)

	var convs []models.Conversation
	database.DB.
		Where("status = ? AND last_direction = ?", "open", "in").
		Where("waiting_since IS NOT NULL AND waiting_since <= ?", cutoff).
		Where("sla_alerted_at IS NULL OR sla_alerted_at < waiting_since").
		Order("waiting_since ASC").
		Find(&convs)
	if len(convs) == 0 {
		return
	}

	now := time.Now()
	msg := fmt.Sprintf("⏰ <b>แชทรอตอบเกิน %d นาที (%d ราย)</b>\n", cfg.ChatSLAMinutes, len(convs))
	for i, conv := range convs {
		if i >= 10 {
			msg += fmt.Sprintf("…และอีก %d ราย\n", len(convs)-10)
			break
		}
		mins := int(now.Sub(*conv.WaitingSince).Minutes())
		wait := fmt.Sprintf("%d นาที", mins)
		if mins >= 60 {
			wait = fmt.Sprintf("%d ชม. %d นาที", mins/60, mins%60)
		}
		preview := []rune(conv.LastMessageText)
		if len(preview) > 40 {
			preview = append(preview[:40], '…')
		}
		msg += fmt.Sprintf("• [%s] %s — รอ %s\n   “%s”\n",
			chatPlatformShort(conv.Platform), html.EscapeString(conv.DisplayName),
			wait, html.EscapeString(string(preview)))
	}
	msg += "\nเปิดเมนู Chats ในระบบหลังบ้านเพื่อตอบได้เลย"

	telegram.SendText(msg)
	log.Printf("chat SLA: alerted %d waiting conversation(s)", len(convs))

	ids := make([]uint, len(convs))
	for i, c := range convs {
		ids[i] = c.ID
	}
	database.DB.Model(&models.Conversation{}).Where("id IN ?", ids).
		Update("sla_alerted_at", now)
}

func chatPlatformShort(p string) string {
	switch p {
	case "line":
		return "LINE"
	case "facebook":
		return "FB"
	case "instagram":
		return "IG"
	}
	return p
}
