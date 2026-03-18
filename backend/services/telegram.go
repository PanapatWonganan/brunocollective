package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"brunocollective_inventory/config"
	"brunocollective_inventory/database"
	"brunocollective_inventory/models"
)

type TelegramNotifier struct {
	botToken string
	chatID   string
	baseURL  string
	enabled  bool
}

func NewTelegramNotifier(cfg *config.Config) *TelegramNotifier {
	enabled := cfg.TelegramBotToken != "" && cfg.TelegramChatID != ""
	if enabled {
		log.Println("Telegram notifications enabled")
	} else {
		log.Println("Telegram notifications disabled (TELEGRAM_BOT_TOKEN or TELEGRAM_CHAT_ID not set)")
	}
	return &TelegramNotifier{
		botToken: cfg.TelegramBotToken,
		chatID:   cfg.TelegramChatID,
		baseURL:  cfg.BaseURL,
		enabled:  enabled,
	}
}

// NotifyNewOrder sends a notification when a new order is created
func (t *TelegramNotifier) NotifyNewOrder(order *models.Order) {
	if !t.enabled {
		return
	}

	items := formatOrderItems(order.Items)
	stock := getStockSummary(order.Items)
	today := getTodaySummary()

	msg := fmt.Sprintf(""+
		"\U0001F4E6 <b>New Order #%d</b>\n"+
		"-------------------------------\n"+
		"Customer: %s\n"+
		"Total: %.2f THB\n"+
		"Items:\n%s\n"+
		"-------------------------------\n"+
		"Stock Remaining:\n%s\n"+
		"-------------------------------\n"+
		"%s",
		order.ID,
		html.EscapeString(order.Customer.Name),
		order.TotalAmount,
		items,
		stock,
		today,
	)

	if order.SlipImage != "" {
		imageURL := fmt.Sprintf("%s/uploads/%s", t.baseURL, order.SlipImage)
		go t.sendPhoto(msg, imageURL)
	} else {
		go t.sendMessage(msg)
	}
}

// NotifyStatusChange sends a notification when order status changes
func (t *TelegramNotifier) NotifyStatusChange(order *models.Order, newStatus models.OrderStatus) {
	if !t.enabled {
		return
	}

	statusEmoji := map[models.OrderStatus]string{
		models.StatusPending:   "\U0001F7E1",
		models.StatusConfirmed: "\U0001F535",
		models.StatusShipped:   "\U0001F69A",
		models.StatusDelivered: "\u2705",
		models.StatusCancelled: "\u274C",
	}

	emoji := statusEmoji[newStatus]
	if emoji == "" {
		emoji = "\U0001F504"
	}

	stock := getStockSummary(order.Items)
	today := getTodaySummary()

	msg := fmt.Sprintf(""+
		"%s <b>Order #%d Status Updated</b>\n"+
		"-------------------------------\n"+
		"Customer: %s\n"+
		"Status: %s\n"+
		"Total: %.2f THB\n"+
		"-------------------------------\n"+
		"Stock Remaining:\n%s\n"+
		"-------------------------------\n"+
		"%s",
		emoji,
		order.ID,
		html.EscapeString(order.Customer.Name),
		strings.ToUpper(string(newStatus)),
		order.TotalAmount,
		stock,
		today,
	)

	go t.sendMessage(msg)
}

// NotifySlipUploaded sends a notification when payment slip is uploaded
func (t *TelegramNotifier) NotifySlipUploaded(order *models.Order) {
	if !t.enabled {
		return
	}

	msg := fmt.Sprintf(""+
		"\U0001F4B3 <b>Payment Slip Uploaded</b>\n"+
		"-------------------------------\n"+
		"Order #%d\n"+
		"Customer: %s\n"+
		"Total: %.2f THB",
		order.ID,
		html.EscapeString(order.Customer.Name),
		order.TotalAmount,
	)

	if order.SlipImage != "" {
		imageURL := fmt.Sprintf("%s/uploads/%s", t.baseURL, order.SlipImage)
		go t.sendPhoto(msg, imageURL)
	} else {
		go t.sendMessage(msg)
	}
}

// SendDailySummary sends a daily order summary to the Telegram chat
func (t *TelegramNotifier) SendDailySummary() {
	if !t.enabled {
		return
	}

	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, now.Location())
	endOfDay := time.Date(now.Year(), now.Month(), now.Day()-1, 23, 59, 59, 0, now.Location())

	var orders []models.Order
	database.DB.Preload("Customer").Preload("Items.Product").
		Where("created_at BETWEEN ? AND ?", startOfDay, endOfDay).
		Find(&orders)

	totalOrders := len(orders)
	var totalRevenue float64
	statusCount := map[models.OrderStatus]int{}

	for _, order := range orders {
		totalRevenue += order.TotalAmount
		statusCount[order.Status]++
	}

	dateStr := startOfDay.Format("02/01/2006")

	msg := fmt.Sprintf(""+
		"\U0001F4CA <b>Daily Summary (%s)</b>\n"+
		"===============================\n"+
		"Total Orders: %d\n"+
		"Total Revenue: %.2f THB\n"+
		"-------------------------------\n"+
		"Status Breakdown:\n"+
		"  \U0001F7E1 Pending: %d\n"+
		"  \U0001F535 Confirmed: %d\n"+
		"  \U0001F69A Shipped: %d\n"+
		"  \u2705 Delivered: %d\n"+
		"  \u274C Cancelled: %d\n"+
		"===============================",
		dateStr,
		totalOrders,
		totalRevenue,
		statusCount[models.StatusPending],
		statusCount[models.StatusConfirmed],
		statusCount[models.StatusShipped],
		statusCount[models.StatusDelivered],
		statusCount[models.StatusCancelled],
	)

	t.sendMessage(msg)
}

func (t *TelegramNotifier) sendMessage(text string) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.botToken)

	body := map[string]interface{}{
		"chat_id":    t.chatID,
		"text":       text,
		"parse_mode": "HTML",
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		log.Printf("Telegram notify marshal error: %v", err)
		return
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Printf("Telegram notify send error: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		log.Printf("Telegram notify response: %d - %s", resp.StatusCode, string(respBody))
	}
}

func (t *TelegramNotifier) sendPhoto(caption string, imageURL string) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendPhoto", t.botToken)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	writer.WriteField("chat_id", t.chatID)
	writer.WriteField("caption", caption)
	writer.WriteField("parse_mode", "HTML")
	writer.WriteField("photo", imageURL)
	writer.Close()

	resp, err := http.Post(url, writer.FormDataContentType(), &buf)
	if err != nil {
		log.Printf("Telegram photo send error: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		log.Printf("Telegram photo response: %d - %s", resp.StatusCode, string(respBody))
	}
}

func getTodaySummary() string {
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	var orders []models.Order
	database.DB.Preload("Items").
		Where("created_at >= ? AND status != ?", startOfDay, models.StatusCancelled).
		Find(&orders)

	totalOrders := len(orders)
	var totalRevenue float64
	var totalItems int

	for _, order := range orders {
		totalRevenue += order.TotalAmount
		for _, item := range order.Items {
			totalItems += item.Quantity
		}
	}

	return fmt.Sprintf(
		"\U0001F4C8 Today: %d orders | %d items | %.2f THB",
		totalOrders, totalItems, totalRevenue,
	)
}

func formatOrderItems(items []models.OrderItem) string {
	var lines []string
	for _, item := range items {
		name := "Unknown"
		if item.Product.Name != "" {
			name = item.Product.Name
		}
		lines = append(lines, fmt.Sprintf("  - %s x%d (%.2f)", html.EscapeString(name), item.Quantity, item.Price*float64(item.Quantity)))
	}
	return strings.Join(lines, "\n")
}

func getStockSummary(orderItems []models.OrderItem) string {
	var lines []string
	seen := make(map[uint]bool)

	for _, item := range orderItems {
		if seen[item.ProductID] {
			continue
		}
		seen[item.ProductID] = true

		var product models.Product
		if err := database.DB.First(&product, item.ProductID).Error; err != nil {
			continue
		}

		warning := ""
		if product.Stock <= 5 {
			warning = " \u26A0\uFE0F LOW"
		}
		if product.Stock == 0 {
			warning = " \U0001F6A8 OUT OF STOCK"
		}

		lines = append(lines, fmt.Sprintf("  - %s: %d%s", html.EscapeString(product.Name), product.Stock, warning))
	}

	if len(lines) == 0 {
		return "  (no products)"
	}
	return strings.Join(lines, "\n")
}
