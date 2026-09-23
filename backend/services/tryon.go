package services

// Virtual try-on — renders a shopper's photo wearing one of our garments with
// a Gemini image model (REST, no SDK). Disabled when GEMINI_API_KEY is unset;
// the storefront hides the button when the status endpoint reports that.
//
// Privacy: the customer photo only ever lives in memory for the duration of
// one request — it is sent to Google for generation and never written to disk
// or the database.

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"brunocollective_inventory/config"
)

// ErrTryOnBusy is returned when Google rate-limits us (HTTP 429) — the UI
// shows a "try again shortly" message rather than a hard failure.
var ErrTryOnBusy = errors.New("try-on: upstream rate limited")

// ErrTryOnNoImage is returned when the model answered without an image
// (safety block, refusal, or a text-only reply).
var ErrTryOnNoImage = errors.New("try-on: no image in response")

type TryOnClient struct {
	apiKey string
	model  string
	http   *http.Client
}

func NewTryOnClient(cfg *config.Config) *TryOnClient {
	return &TryOnClient{
		apiKey: cfg.GeminiAPIKey,
		model:  cfg.TryOnModel,
		http:   &http.Client{Timeout: 180 * time.Second}, // measured 17s–120s+ per generation
	}
}

func (t *TryOnClient) Enabled() bool { return t != nil && t.apiKey != "" }

// TryOnResult is the generated image.
type TryOnResult struct {
	Data []byte
	Mime string
}

// tryOnPrompt tells the model what to keep and what to change. Category
// steers where the item goes (a ring is not a shirt).
func tryOnPrompt(productName, category string) string {
	cat := strings.ToLower(category)
	var placement string
	switch {
	case strings.Contains(cat, "แหวน") || strings.Contains(cat, "ring"):
		placement = "Image 2 is a ring. Produce a natural close-up of the person's hand from image 1 wearing the ring on a finger, framed so the ring is clearly visible (crop in on the hand if image 1 shows more of the body); keep skin tone, lighting and background consistent with image 1."
	case strings.Contains(cat, "สร้อย") || strings.Contains(cat, "necklace") || strings.Contains(cat, "bracelet") ||
		strings.Contains(cat, "เครื่องประดับ") || strings.Contains(cat, "jewel"):
		placement = "Image 2 is a piece of jewellery. Show it worn where such a piece is normally worn (neck, wrist or hand), framed close enough that the piece is clearly visible (crop in if needed); keep the person's clothing unchanged."
	case strings.Contains(cat, "รองเท้า") || strings.Contains(cat, "shoe"):
		placement = "Image 2 is a pair of shoes. Show the person wearing them on their feet; keep the rest of the outfit unchanged."
	case strings.Contains(cat, "กระเป๋า") || strings.Contains(cat, "bag"):
		placement = "Image 2 is a bag. Show the person carrying or wearing it naturally; keep the outfit unchanged."
	default:
		placement = "Image 2 is a garment. Replace only the clothing the garment would cover; keep other clothing (e.g. trousers under a shirt) unless the garment replaces it."
	}
	return "Virtual try-on for the clothing brand Bruno Collective. " +
		"Image 1 (labelled CUSTOMER) is the shopper — this person is the ONLY person allowed in the output. " +
		"Image 2 (labelled PRODUCT) shows the item \"" + productName + "\" for sale; if a model or mannequin appears in image 2, ignore that person completely — take only the item itself. " +
		placement + " " +
		"Generate one photorealistic image of the CUSTOMER from image 1 wearing the PRODUCT from image 2. " +
		"Keep the customer's face, skin tone, body shape, hair, pose, background and lighting exactly as in image 1; never output the model or background from image 2. " +
		"Reproduce the product's colour, material, print, shape and fit faithfully — do not invent logos, text or design details. " +
		"Do not add any text, watermark or extra people. Output only the final image."
}

type geminiPart struct {
	Text       string            `json:"text,omitempty"`
	InlineData *geminiInlineData `json:"inlineData,omitempty"`
}
type geminiInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}
type geminiRequest struct {
	Contents []struct {
		Parts []geminiPart `json:"parts"`
	} `json:"contents"`
	GenerationConfig struct {
		ResponseModalities []string `json:"responseModalities"`
	} `json:"generationConfig"`
}
type geminiResponse struct {
	Candidates []struct {
		FinishReason string `json:"finishReason"`
		Content      struct {
			Parts []geminiPart `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	PromptFeedback *struct {
		BlockReason string `json:"blockReason"`
	} `json:"promptFeedback"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// Generate renders person wearing garment. Both images are passed inline
// (base64) — keep them modest (the handler normalises the photo to ~1280px).
func (t *TryOnClient) Generate(ctx context.Context, person []byte, personMime string, garment []byte, garmentMime string, productName, category string) (*TryOnResult, error) {
	if !t.Enabled() {
		return nil, errors.New("try-on disabled")
	}
	var req geminiRequest
	req.Contents = make([]struct {
		Parts []geminiPart `json:"parts"`
	}, 1)
	req.Contents[0].Parts = []geminiPart{
		{Text: "Image 1 — CUSTOMER (keep this person, their pose and background):"},
		{InlineData: &geminiInlineData{MimeType: personMime, Data: base64.StdEncoding.EncodeToString(person)}},
		{Text: "Image 2 — PRODUCT (take only the item; ignore any person shown):"},
		{InlineData: &geminiInlineData{MimeType: garmentMime, Data: base64.StdEncoding.EncodeToString(garment)}},
		{Text: tryOnPrompt(productName, category)},
	}
	req.GenerationConfig.ResponseModalities = []string{"IMAGE"}

	return t.run(ctx, req)
}

// Cutout turns a product photo into a clean catalogue image of the item
// alone (flat / ghost-mannequin on white, no model) — the ideal reference
// for both the photo try-on and Decart's live try-on. Product photos that
// show a model wearing the piece otherwise confuse the try-on model about
// which person to dress.
func (t *TryOnClient) Cutout(ctx context.Context, product []byte, productMime, productName, category string) (*TryOnResult, error) {
	if !t.Enabled() {
		return nil, errors.New("try-on disabled")
	}
	cat := strings.ToLower(category)
	item := "garment"
	switch {
	case strings.Contains(cat, "แหวน") || strings.Contains(cat, "ring"):
		item = "ring"
	case strings.Contains(cat, "เครื่องประดับ") || strings.Contains(cat, "jewel") || strings.Contains(cat, "สร้อย"):
		item = "piece of jewellery"
	case strings.Contains(cat, "รองเท้า") || strings.Contains(cat, "shoe"):
		item = "pair of shoes"
	case strings.Contains(cat, "กระเป๋า") || strings.Contains(cat, "bag"):
		item = "bag"
	}
	prompt := "Product photo of \"" + productName + "\". Output a clean e-commerce catalogue image of ONLY the " + item + " from this photo: " +
		"the item alone, front view, laid flat or ghost-mannequin style, centred on a plain white background. " +
		"No person, no body parts, no hands, no hanger, no props, no text. " +
		"Preserve the exact colour, fabric texture, print, logo placement, neckline, sleeve length and proportions. " +
		"If several colour variants are shown, output only the single most prominent one. Output only the image."
	var req geminiRequest
	req.Contents = make([]struct {
		Parts []geminiPart `json:"parts"`
	}, 1)
	req.Contents[0].Parts = []geminiPart{
		{InlineData: &geminiInlineData{MimeType: productMime, Data: base64.StdEncoding.EncodeToString(product)}},
		{Text: prompt},
	}
	req.GenerationConfig.ResponseModalities = []string{"IMAGE"}
	return t.run(ctx, req)
}

// run posts one generateContent request and returns the first image part.
func (t *TryOnClient) run(ctx context.Context, req geminiRequest) (*TryOnResult, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent", t.model)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", t.apiKey)

	resp, err := t.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 32<<20))

	if resp.StatusCode == http.StatusTooManyRequests {
		log.Printf("try-on: gemini 429: %.200s", raw)
		return nil, ErrTryOnBusy
	}
	var out geminiResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("try-on: bad response (%d): %.200s", resp.StatusCode, raw)
	}
	if out.Error != nil {
		return nil, fmt.Errorf("try-on: gemini %d: %s", out.Error.Code, out.Error.Message)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("try-on: gemini http %d", resp.StatusCode)
	}
	for _, cand := range out.Candidates {
		for _, p := range cand.Content.Parts {
			if p.InlineData != nil && p.InlineData.Data != "" {
				data, err := base64.StdEncoding.DecodeString(p.InlineData.Data)
				if err != nil {
					return nil, err
				}
				mime := p.InlineData.MimeType
				if mime == "" {
					mime = "image/png"
				}
				return &TryOnResult{Data: data, Mime: mime}, nil
			}
		}
	}
	reason := ""
	if out.PromptFeedback != nil {
		reason = out.PromptFeedback.BlockReason
	} else if len(out.Candidates) > 0 {
		reason = out.Candidates[0].FinishReason
	}
	log.Printf("try-on: no image (reason=%q)", reason)
	return nil, ErrTryOnNoImage
}
