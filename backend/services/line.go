package services

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"brunocollective_inventory/config"
)

// LineClient talks to the LINE Messaging API for the chat inbox. Disabled
// (all calls no-op or error cleanly) when the channel secret/token are not
// configured — same pattern as TelegramNotifier.
type LineClient struct {
	channelSecret string
	channelToken  string
	enabled       bool
	http          *http.Client
}

func NewLineClient(cfg *config.Config) *LineClient {
	enabled := cfg.LineChannelSecret != "" && cfg.LineChannelToken != ""
	if enabled {
		log.Println("LINE chat enabled")
	} else {
		log.Println("LINE chat disabled (LINE_CHANNEL_SECRET or LINE_CHANNEL_ACCESS_TOKEN not set)")
	}
	return &LineClient{
		channelSecret: cfg.LineChannelSecret,
		channelToken:  cfg.LineChannelToken,
		enabled:       enabled,
		http:          &http.Client{Timeout: 15 * time.Second},
	}
}

func (l *LineClient) Enabled() bool { return l.enabled }

// VerifySignature checks the X-Line-Signature header against the raw body
// (base64-encoded HMAC-SHA256 keyed with the channel secret).
func (l *LineClient) VerifySignature(body []byte, signature string) bool {
	if !l.enabled {
		return false
	}
	mac := hmac.New(sha256.New, []byte(l.channelSecret))
	mac.Write(body)
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// PushText sends a text message to a LINE user (Push API — used for admin
// replies, which happen outside the webhook reply-token window).
func (l *LineClient) PushText(userID, text string) error {
	if !l.enabled {
		return fmt.Errorf("LINE ยังไม่ได้ตั้งค่า (LINE_CHANNEL_SECRET / LINE_CHANNEL_ACCESS_TOKEN)")
	}
	payload := map[string]interface{}{
		"to": userID,
		"messages": []map[string]string{
			{"type": "text", "text": text},
		},
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", "https://api.line.me/v2/bot/message/push", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+l.channelToken)

	resp, err := l.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("LINE push failed (%d): %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// LineProfile is the subset of the LINE profile API response we keep.
type LineProfile struct {
	DisplayName string `json:"displayName"`
	PictureURL  string `json:"pictureUrl"`
}

// GetProfile fetches a LINE user's display name and avatar.
func (l *LineClient) GetProfile(userID string) (*LineProfile, error) {
	if !l.enabled {
		return nil, fmt.Errorf("LINE not configured")
	}
	req, err := http.NewRequest("GET", "https://api.line.me/v2/bot/profile/"+userID, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+l.channelToken)

	resp, err := l.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("LINE profile failed (%d)", resp.StatusCode)
	}
	var p LineProfile
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return nil, err
	}
	return &p, nil
}

// GetMessageContent downloads media (image/video/audio) attached to a
// message. Caller owns saving the bytes.
func (l *LineClient) GetMessageContent(messageID string) ([]byte, string, error) {
	if !l.enabled {
		return nil, "", fmt.Errorf("LINE not configured")
	}
	url := fmt.Sprintf("https://api-data.line.me/v2/bot/message/%s/content", messageID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Authorization", "Bearer "+l.channelToken)

	resp, err := l.http.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("LINE content failed (%d)", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 20*1024*1024))
	if err != nil {
		return nil, "", err
	}
	return data, resp.Header.Get("Content-Type"), nil
}
