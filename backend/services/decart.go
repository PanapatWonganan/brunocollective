package services

// Decart realtime virtual try-on ("ลองใส่สด"). The browser streams the
// shopper's webcam to Decart's lucy-vton model over WebRTC using the
// official JS SDK; the only server-side piece is minting short-lived client
// tokens so the permanent dct_* key never reaches the browser. Disabled
// when DECART_API_KEY is unset.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"brunocollective_inventory/config"
)

type DecartClient struct {
	apiKey string
	http   *http.Client
}

func NewDecartClient(cfg *config.Config) *DecartClient {
	return &DecartClient{apiKey: cfg.DecartAPIKey, http: &http.Client{Timeout: 20 * time.Second}}
}

func (d *DecartClient) Enabled() bool { return d != nil && d.apiKey != "" }

// ClientToken is what the browser needs to open a realtime session.
type ClientToken struct {
	APIKey    string `json:"apiKey"`
	ExpiresAt string `json:"expiresAt"`
}

// CreateClientToken mints an ephemeral token scoped to one model, the given
// web origins and a hard per-session streaming cap (seconds) — the cap is
// enforced by Decart itself, so a tab left open can't run up the bill.
func (d *DecartClient) CreateClientToken(ctx context.Context, model string, origins []string, maxSessionSeconds, ttlSeconds int) (*ClientToken, error) {
	if !d.Enabled() {
		return nil, errors.New("decart disabled")
	}
	if maxSessionSeconds < 10 {
		maxSessionSeconds = 10 // Decart minimum
	}
	body := map[string]any{
		"expiresIn":     ttlSeconds,
		"allowedModels": []string{model},
		"constraints":   map[string]any{"realtime": map[string]any{"maxSessionDuration": maxSessionSeconds}},
	}
	if len(origins) > 0 {
		body["allowedOrigins"] = origins
	}
	raw, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.decart.ai/v1/client/tokens", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", d.apiKey)
	req.Header.Set("content-type", "application/json")
	resp, err := d.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("decart token: http %d: %.200s", resp.StatusCode, data)
	}
	var tok ClientToken
	if err := json.Unmarshal(data, &tok); err != nil || tok.APIKey == "" {
		return nil, fmt.Errorf("decart token: bad response: %.200s", data)
	}
	return &tok, nil
}
