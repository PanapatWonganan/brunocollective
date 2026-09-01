package services

// AI chat assistant — answers customer chat messages with Claude, grounded in
// the live shop context (catalog + stock) the caller passes in. Disabled when
// ANTHROPIC_API_KEY is unset, same graceful degradation as Telegram/LINE.

import (
	"context"
	"log"
	"strings"
	"time"

	"brunocollective_inventory/config"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// aiHandoffSentinel is what the model outputs when a human must take over.
const aiHandoffSentinel = "[HANDOFF]"

// aiSystemPrompt sets the assistant's role and hard boundaries. The live shop
// context (catalog/stock) is appended after it per request. Kept in Thai —
// replies go to shoppers as-is.
const aiSystemPrompt = `คุณคือผู้ช่วยตอบแชทของร้าน Bruno Collective (แบรนด์เสื้อผ้าไทย จัดส่งทั่วประเทศ)
หน้าที่: ตอบคำถามลูกค้าเรื่องสินค้า ราคา ไซส์ สี สต็อก และวิธีสั่งซื้อ อย่างสุภาพ เป็นกันเอง ลงท้ายว่า "ค่ะ"

กติกาสำคัญ:
- ตอบจากข้อมูลร้านด้านล่างเท่านั้น ห้ามเดาหรือแต่งข้อมูลที่ไม่มี
- ตอบสั้นกระชับ 1-4 ประโยค เหมือนแอดมินร้านตอบแชท ไม่ใช้หัวข้อหรือ bullet
- วิธีสั่งซื้อ: สั่งผ่านหน้าเว็บของร้าน หรือแจ้งรุ่น/ไซส์ในแชทได้เลย เดี๋ยวแอดมินสรุปยอดและส่งลิงก์ชำระเงินให้
- การชำระเงิน: โอนเข้าบัญชีธนาคารกสิกรไทย 231-1421-053 (บจก. บรูโน่ คอลเลคทีฟ) แล้วแนบสลิป
- ห้ามรับปากเรื่องส่วนลด ราคาพิเศษ หรือกำหนดเวลาจัดส่งแทนร้าน

ถ้าข้อความล่าสุดของลูกค้าเป็นเรื่องต่อไปนี้ ให้ตอบว่า [HANDOFF] เท่านั้น ห้ามตอบอย่างอื่น:
- ยืนยันยอดโอน ตรวจสลิป หรือสถานะการชำระเงิน
- สถานะจัดส่ง เลขพัสดุ หรือของยังไม่ถึง
- ขอส่วนลด ต่อรองราคา หรือโปรโมชั่นที่ไม่มีในข้อมูลร้าน
- เคลม เปลี่ยน คืนสินค้า หรือร้องเรียน
- ค่าส่งหรือระยะเวลาจัดส่ง (ไม่มีในข้อมูลร้าน)
- เรื่องอื่นใดที่ข้อมูลร้านด้านล่างตอบไม่ได้`

type AIClient struct {
	enabled bool
	client  anthropic.Client
	model   string
}

func NewAIClient(cfg *config.Config) *AIClient {
	enabled := cfg.AnthropicAPIKey != ""
	if enabled {
		log.Printf("AI chat assistant enabled (model %s)", cfg.AIModel)
	} else {
		log.Println("AI chat assistant disabled (ANTHROPIC_API_KEY not set)")
	}
	return &AIClient{
		enabled: enabled,
		client:  anthropic.NewClient(option.WithAPIKey(cfg.AnthropicAPIKey)),
		model:   cfg.AIModel,
	}
}

func (a *AIClient) Enabled() bool { return a.enabled }

// AIChatTurn is one prior message in the conversation, oldest first.
type AIChatTurn struct {
	Direction string // "in" = customer, "out" = shop
	Text      string
}

// AnswerChat asks Claude to answer the customer's latest message. Returns
// (reply, handoff, err): handoff true means the model declined to answer (per
// its instructions, or via a safety refusal) and a human should reply instead.
func (a *AIClient) AnswerChat(shopContext string, turns []AIChatTurn) (string, bool, error) {
	if !a.enabled {
		return "", true, nil
	}

	msgs := make([]anthropic.MessageParam, 0, len(turns))
	for _, t := range turns {
		text := strings.TrimSpace(t.Text)
		if text == "" {
			continue
		}
		if t.Direction == "out" {
			msgs = append(msgs, anthropic.NewAssistantMessage(anthropic.NewTextBlock(text)))
		} else {
			msgs = append(msgs, anthropic.NewUserMessage(anthropic.NewTextBlock(text)))
		}
	}
	// The API requires the first message to be a user turn.
	for len(msgs) > 0 && msgs[0].Role != anthropic.MessageParamRoleUser {
		msgs = msgs[1:]
	}
	if len(msgs) == 0 {
		return "", true, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	resp, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(a.model),
		MaxTokens: 1024,
		// Chat FAQ answers are simple — low effort keeps latency and cost down.
		OutputConfig: anthropic.OutputConfigParam{Effort: anthropic.OutputConfigEffortLow},
		System: []anthropic.TextBlockParam{
			{Text: aiSystemPrompt + "\n\n" + shopContext},
		},
		Messages: msgs,
	})
	if err != nil {
		return "", true, err
	}

	// Safety classifiers can decline a request (stop_reason "refusal") —
	// treat it like any other handoff: the human admin answers.
	if resp.StopReason == "refusal" {
		return "", true, nil
	}

	var sb strings.Builder
	for _, block := range resp.Content {
		if block.Type == "text" {
			sb.WriteString(block.Text)
		}
	}
	reply := strings.TrimSpace(sb.String())
	if reply == "" || strings.Contains(reply, aiHandoffSentinel) {
		return "", true, nil
	}
	return reply, false, nil
}
