package services

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"brunocollective_inventory/config"
)

const metaGraphBase = "https://graph.facebook.com/v21.0"

// MetaClient talks to the Meta Graph API for Facebook Messenger and
// Instagram DM. One Facebook app + one page access token serve both
// platforms (IG messages route through the linked page). Disabled cleanly
// when the app secret / page token are not configured.
type MetaClient struct {
	appSecret   string
	verifyToken string
	pageToken   string
	enabled     bool
	http        *http.Client
}

func NewMetaClient(cfg *config.Config) *MetaClient {
	enabled := cfg.MetaAppSecret != "" && cfg.MetaPageToken != ""
	if enabled {
		log.Println("Meta chat (Facebook/Instagram) enabled")
	} else {
		log.Println("Meta chat disabled (META_APP_SECRET or META_PAGE_ACCESS_TOKEN not set)")
	}
	return &MetaClient{
		appSecret:   cfg.MetaAppSecret,
		verifyToken: cfg.MetaVerifyToken,
		pageToken:   cfg.MetaPageToken,
		enabled:     enabled,
		http:        &http.Client{Timeout: 15 * time.Second},
	}
}

func (m *MetaClient) Enabled() bool { return m.enabled }

// VerifyToken is the owner-chosen string Meta echoes back during the
// webhook subscribe handshake (GET hub.verify_token).
func (m *MetaClient) VerifyToken() string { return m.verifyToken }

// VerifySignature checks X-Hub-Signature-256 ("sha256=<hex hmac>") against
// the raw body, keyed with the app secret.
func (m *MetaClient) VerifySignature(body []byte, header string) bool {
	if !m.enabled {
		return false
	}
	sig := strings.TrimPrefix(header, "sha256=")
	if sig == header {
		return false
	}
	mac := hmac.New(sha256.New, []byte(m.appSecret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(sig))
}

// SendText pushes a text reply to a PSID (Facebook) or IGSID (Instagram)
// via the Send API and returns Meta's message id (used to dedupe the echo
// event that follows).
func (m *MetaClient) SendText(recipientID, text string) (string, error) {
	if !m.enabled {
		return "", fmt.Errorf("Facebook/Instagram ยังไม่ได้ตั้งค่า (META_APP_SECRET / META_PAGE_ACCESS_TOKEN)")
	}
	payload := map[string]interface{}{
		"recipient":      map[string]string{"id": recipientID},
		"messaging_type": "RESPONSE",
		"message":        map[string]string{"text": text},
	}
	body, _ := json.Marshal(payload)

	url := metaGraphBase + "/me/messages?access_token=" + m.pageToken
	resp, err := m.http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("Meta send failed (%d): %s", resp.StatusCode, string(respBody))
	}
	var result struct {
		MessageID string `json:"message_id"`
	}
	_ = json.Unmarshal(respBody, &result)
	return result.MessageID, nil
}

// MetaProfile is the subset of profile fields we keep. `name` works for
// both PSIDs and IGSIDs; IG additionally has username.
type MetaProfile struct {
	Name       string `json:"name"`
	Username   string `json:"username"`
	ProfilePic string `json:"profile_pic"`
}

// GetProfile fetches display info for a PSID/IGSID. The direct profile
// endpoint requires Advanced Access ("Business Asset User Profile Access"
// via App Review) — without it, it returns error 100/33 for real customers.
// As a fallback we ask the page conversations endpoint, which returns the
// participant's real name with standard access (no avatar, but a name).
func (m *MetaClient) GetProfile(userID string) (*MetaProfile, error) {
	if !m.enabled {
		return nil, fmt.Errorf("Meta not configured")
	}
	url := fmt.Sprintf("%s/%s?fields=name,username,profile_pic&access_token=%s", metaGraphBase, userID, m.pageToken)
	resp, err := m.http.Get(url)
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode < 300 {
			var p MetaProfile
			if derr := json.NewDecoder(resp.Body).Decode(&p); derr == nil {
				return &p, nil
			}
		}
	}
	// Fallback: find the user among the page's conversation participants.
	for _, platform := range []string{"", "instagram"} {
		if name := m.participantName(userID, platform); name != "" {
			return &MetaProfile{Name: name}, nil
		}
	}
	return nil, fmt.Errorf("Meta profile unavailable for %s", userID)
}

type conversationsPage struct {
	Data []struct {
		Participants struct {
			Data []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"data"`
		} `json:"participants"`
	} `json:"data"`
	Paging struct {
		Next string `json:"next"`
	} `json:"paging"`
}

// participantName resolves one user's display name via the conversations
// endpoint (platform "" = messenger, or "instagram").
func (m *MetaClient) participantName(userID, platform string) string {
	u := fmt.Sprintf("%s/me/conversations?fields=participants&user_id=%s&access_token=%s", metaGraphBase, userID, m.pageToken)
	if platform != "" {
		u += "&platform=" + platform
	}
	resp, err := m.http.Get(u)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return ""
	}
	var out conversationsPage
	if json.NewDecoder(resp.Body).Decode(&out) != nil {
		return ""
	}
	for _, conv := range out.Data {
		for _, p := range conv.Participants.Data {
			if p.ID == userID && p.Name != "" {
				return p.Name
			}
		}
	}
	return ""
}

// ListConversationNames pages through every page conversation and returns a
// userID -> display name map. platform "" = messenger, or "instagram".
// Used by cmd/refreshprofiles to backfill names in a handful of requests
// instead of one per thread.
func (m *MetaClient) ListConversationNames(platform string) (map[string]string, error) {
	if !m.enabled {
		return nil, fmt.Errorf("Meta not configured")
	}
	names := make(map[string]string)
	url := fmt.Sprintf("%s/me/conversations?fields=participants&limit=100&access_token=%s", metaGraphBase, m.pageToken)
	if platform != "" {
		url += "&platform=" + platform
	}
	for url != "" {
		resp, err := m.http.Get(url)
		if err != nil {
			return names, err
		}
		if resp.StatusCode >= 300 {
			resp.Body.Close()
			return names, fmt.Errorf("Meta conversations failed (%d)", resp.StatusCode)
		}
		var out conversationsPage
		err = json.NewDecoder(resp.Body).Decode(&out)
		resp.Body.Close()
		if err != nil {
			return names, err
		}
		for _, conv := range out.Data {
			for _, p := range conv.Participants.Data {
				if p.Name != "" {
					names[p.ID] = p.Name
				}
			}
		}
		url = out.Paging.Next
	}
	return names, nil
}

// postForm POSTs url-encoded params to a Graph API path and returns an
// error with Meta's message on non-2xx.
func (m *MetaClient) postForm(path string, params map[string]string) error {
	if !m.enabled {
		return fmt.Errorf("Facebook/Instagram ยังไม่ได้ตั้งค่า (META_APP_SECRET / META_PAGE_ACCESS_TOKEN)")
	}
	form := make([]string, 0, len(params)+1)
	for k, v := range params {
		form = append(form, k+"="+urlQueryEscape(v))
	}
	form = append(form, "access_token="+urlQueryEscape(m.pageToken))
	body := strings.Join(form, "&")

	resp, err := m.http.Post(metaGraphBase+path, "application/x-www-form-urlencoded", strings.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("Meta API failed (%d): %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// ReplyToComment posts a public reply under a comment. Facebook uses
// /{id}/comments, Instagram uses /{id}/replies.
func (m *MetaClient) ReplyToComment(platform, commentID, text string) error {
	path := "/" + commentID + "/comments"
	if platform == "instagram" {
		path = "/" + commentID + "/replies"
	}
	return m.postForm(path, map[string]string{"message": text})
}

// HideComment hides a comment from other users (the commenter and page
// still see it). Facebook: is_hidden; Instagram: hide.
func (m *MetaClient) HideComment(platform, commentID string) error {
	param := "is_hidden"
	if platform == "instagram" {
		param = "hide"
	}
	return m.postForm("/"+commentID, map[string]string{param: "true"})
}

// PrivateReply DMs the commenter (allowed once per comment, both
// platforms) via the Send API's comment_id recipient.
func (m *MetaClient) PrivateReply(commentID, text string) error {
	if !m.enabled {
		return fmt.Errorf("Facebook/Instagram ยังไม่ได้ตั้งค่า (META_APP_SECRET / META_PAGE_ACCESS_TOKEN)")
	}
	payload := map[string]interface{}{
		"recipient": map[string]string{"comment_id": commentID},
		"message":   map[string]string{"text": text},
	}
	body, _ := json.Marshal(payload)
	url := metaGraphBase + "/me/messages?access_token=" + m.pageToken
	resp, err := m.http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("Meta private reply failed (%d): %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// urlQueryEscape percent-encodes a form value.
func urlQueryEscape(s string) string {
	return url.QueryEscape(s)
}

// DownloadMedia fetches an attachment from Meta's CDN URL (those URLs
// expire, so media is persisted locally on receipt).
func (m *MetaClient) DownloadMedia(url string) ([]byte, string, error) {
	resp, err := m.http.Get(url)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("media download failed (%d)", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 20*1024*1024))
	if err != nil {
		return nil, "", err
	}
	return data, resp.Header.Get("Content-Type"), nil
}
