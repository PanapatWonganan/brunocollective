package config

import "os"

type Config struct {
	Port             string
	DBPath           string
	JWTSecret        string
	UploadDir        string
	TelegramBotToken string
	TelegramChatID   string
	BaseURL          string
}

func Load() *Config {
	return &Config{
		Port:             getEnv("PORT", "8080"),
		DBPath:           getEnv("DB_PATH", "inventory.db"),
		JWTSecret:        getEnv("JWT_SECRET", "change-me-in-production"),
		UploadDir:        getEnv("UPLOAD_DIR", "./uploads"),
		TelegramBotToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
		TelegramChatID:   getEnv("TELEGRAM_CHAT_ID", ""),
		BaseURL:          getEnv("BASE_URL", "http://localhost:8080"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
