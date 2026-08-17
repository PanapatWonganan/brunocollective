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
	// AI chat assistant (Claude). Empty API key = disabled, same graceful
	// degradation as Telegram/LINE.
	AnthropicAPIKey string
	AIModel         string
}

func Load() *Config {
	return &Config{
		Port:              getEnv("PORT", "8080"),
		DBPath:            getEnv("DB_PATH", "inventory.db"),
		JWTSecret:         getEnv("JWT_SECRET", "change-me-in-production"),
		UploadDir:         getEnv("UPLOAD_DIR", "./uploads"),
		TelegramBotToken:  getEnv("TELEGRAM_BOT_TOKEN", ""),
		TelegramChatID:    getEnv("TELEGRAM_CHAT_ID", ""),
		BaseURL:           getEnv("BASE_URL", "http://localhost:8080"),
		LineChannelSecret: getEnv("LINE_CHANNEL_SECRET", ""),
		LineChannelToken:  getEnv("LINE_CHANNEL_ACCESS_TOKEN", ""),
		MetaAppSecret:     getEnv("META_APP_SECRET", ""),
		MetaVerifyToken:   getEnv("META_VERIFY_TOKEN", ""),
		MetaPageToken:     getEnv("META_PAGE_ACCESS_TOKEN", ""),
		ChatSLAMinutes:    getEnvInt("CHAT_SLA_MINUTES", 10),
		AnthropicAPIKey:   getEnv("ANTHROPIC_API_KEY", ""),
		AIModel:           getEnv("AI_MODEL", "claude-opus-5"),
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
