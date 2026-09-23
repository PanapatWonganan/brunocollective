package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port             string
	DBPath           string
	JWTSecret        string
	UploadDir        string
	TelegramBotToken string
	TelegramChatID   string
	BaseURL          string
	// LINE Messaging API (chat inbox). Both empty = LINE chat disabled,
	// same graceful-degradation pattern as Telegram.
	LineChannelSecret string
	LineChannelToken  string
	// Meta (Facebook Messenger + Instagram DM). One app serves both — the
	// page access token sends replies for the FB page and its linked IG
	// account. VerifyToken is an owner-chosen string echoed during the
	// webhook subscribe handshake.
	MetaAppSecret   string
	MetaVerifyToken string
	MetaPageToken   string
	// ChatSLAMinutes: alert Telegram when a chat has waited this long for a
	// reply. 0 disables the watcher.
	ChatSLAMinutes int
	// ChatWaitingExpireHours: a thread that has waited this long with no new
	// inbound message is assumed answered outside the system (LINE OA app —
	// LINE sends no echo events) and drops out of the "รอตอบ" queue on its
	// own. 0 disables the auto-expiry.
	ChatWaitingExpireHours int
	// AI chat assistant (Claude). Empty API key = disabled, same graceful
	// degradation as Telegram/LINE.
	AnthropicAPIKey string
	AIModel         string
	// Virtual try-on (Gemini image model). Empty API key = feature hidden on
	// the storefront. Limits are generations per day: per IP for guests, per
	// customer for logged-in members. Each generation is billed by Google.
	GeminiAPIKey          string
	TryOnMembersOnly      bool // guests see the button but must sign up first
	TryOnModel            string
	TryOnDailyLimit       int
	TryOnMemberDailyLimit int
	// Live (webcam) try-on via Decart's realtime lucy-vton model. Empty key
	// = hidden. Billed per streamed second (~$0.02/s), so sessions are
	// capped: seconds per session and sessions per day, guest vs member.
	DecartAPIKey            string
	LiveTryOnMembersOnly    bool // guests see the button but must sign up first
	LiveTryOnModel          string
	LiveTryOnSessionSeconds int
	LiveTryOnMemberSeconds  int
	LiveTryOnDailySessions  int
	LiveTryOnMemberSessions int
	LiveTryOnAllowedOrigins string // comma-separated web origins the token may be used from
}

func Load() *Config {
	return &Config{
		Port:                    getEnv("PORT", "8080"),
		DBPath:                  getEnv("DB_PATH", "inventory.db"),
		JWTSecret:               getEnv("JWT_SECRET", "change-me-in-production"),
		UploadDir:               getEnv("UPLOAD_DIR", "./uploads"),
		TelegramBotToken:        getEnv("TELEGRAM_BOT_TOKEN", ""),
		TelegramChatID:          getEnv("TELEGRAM_CHAT_ID", ""),
		BaseURL:                 getEnv("BASE_URL", "http://localhost:8080"),
		LineChannelSecret:       getEnv("LINE_CHANNEL_SECRET", ""),
		LineChannelToken:        getEnv("LINE_CHANNEL_ACCESS_TOKEN", ""),
		MetaAppSecret:           getEnv("META_APP_SECRET", ""),
		MetaVerifyToken:         getEnv("META_VERIFY_TOKEN", ""),
		MetaPageToken:           getEnv("META_PAGE_ACCESS_TOKEN", ""),
		ChatSLAMinutes:          getEnvInt("CHAT_SLA_MINUTES", 10),
		ChatWaitingExpireHours:  getEnvInt("CHAT_WAITING_EXPIRE_HOURS", 12),
		AnthropicAPIKey:         getEnv("ANTHROPIC_API_KEY", ""),
		AIModel:                 getEnv("AI_MODEL", "claude-opus-5"),
		GeminiAPIKey:            getEnv("GEMINI_API_KEY", ""),
		TryOnMembersOnly:        getEnv("TRYON_MEMBERS_ONLY", "true") != "false",
		TryOnModel:              getEnv("TRYON_MODEL", "gemini-3.1-flash-image"),
		TryOnDailyLimit:         getEnvInt("TRYON_DAILY_LIMIT", 5),
		TryOnMemberDailyLimit:   getEnvInt("TRYON_MEMBER_DAILY_LIMIT", 20),
		DecartAPIKey:            getEnv("DECART_API_KEY", ""),
		LiveTryOnMembersOnly:    getEnv("LIVE_TRYON_MEMBERS_ONLY", "true") != "false",
		LiveTryOnModel:          getEnv("LIVE_TRYON_MODEL", "lucy-vton-latest"),
		LiveTryOnSessionSeconds: getEnvInt("LIVE_TRYON_SESSION_SECONDS", 20),
		LiveTryOnMemberSeconds:  getEnvInt("LIVE_TRYON_MEMBER_SECONDS", 20),
		LiveTryOnDailySessions:  getEnvInt("LIVE_TRYON_DAILY_SESSIONS", 2),
		LiveTryOnMemberSessions: getEnvInt("LIVE_TRYON_MEMBER_SESSIONS", 4),
		LiveTryOnAllowedOrigins: getEnv("LIVE_TRYON_ALLOWED_ORIGINS", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
