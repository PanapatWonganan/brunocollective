package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"brunocollective_inventory/config"
	"brunocollective_inventory/database"
	"brunocollective_inventory/models"
)

type LineNotifier struct {
	channelToken string
	groupID      string
	baseURL      string
	enabled      bool
}

func NewLineNotifier(cfg *config.Config) *LineNotifier {
	enabled := cfg.LineChannelToken != "" && cfg.LineGroupID != ""
	if enabled {
		log.Println("LINE notifications enabled")
	} else {
		log.Println("LINE notifications disabled (LINE_CHANNEL_TOKEN or LINE_GROUP_ID not set)")
	}
	return &LineNotifier{
		channelToken: cfg.LineChannelToken,
		groupID:      cfg.LineGroupID,
		baseURL:      cfg.BaseURL,
		enabled:      enabled,
	}
}

// NotifyNewOrder sends a notification when a new order is created
func (l *LineNotifier) NotifyNewOrder(order *models.Order) {
	if !l.enabled {
		return
	}

	items := formatOrderItems(order.Items)
	stock := getStockSummary(order.Items)

	today := getTodaySummary()

	msg := fmt.Sprintf(""+
		"\U0001F4E6 New Order #%d\n"+
		"-------------------------------\n"+
		"Customer: %s\n"+
		"Total: %.2f THB\n"+
		"Items:\n%s\n"+
		"-------------------------------\n"+
		"Stock Remaining:\n%s\n"+
		"-------------------------------\n"+
		"%s",
		order.ID,
		order.Customer.Name,
		order.TotalAmount,
		items,
		stock,
		today,
	)

	go l.pushMessage(msg)
}

// NotifyStatusChange sends a notification when order status changes
func (l *LineNotifier) NotifyStatusChange(order *models.Order, newStatus models.OrderStatus) {
	if !l.enabled {
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
		"%s Order #%d Status Updated\n"+
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
		order.Customer.Name,
		strings.ToUpper(string(newStatus)),
		order.TotalAmount,
		stock,
		today,
	)

	go l.pushMessage(msg)
}

// NotifySlipUploaded sends a notification when payment slip is uploaded
func (l *LineNotifier) NotifySlipUploaded(order *models.Order) {
	if !l.enabled {
		return
	}

	msg := fmt.Sprintf(""+
		"\U0001F4B3 Payment Slip Uploaded\n"+
		"-------------------------------\n"+
		"Order #%d\n"+
		"Customer: %s\n"+
		"Total: %.2f THB",
		order.ID,
		order.Customer.Name,
		order.TotalAmount,
	)

	if order.SlipImage != "" {
		imageURL := fmt.Sprintf("%s/uploads/%s", l.baseURL, order.SlipImage)
		go l.pushMessageWithImage(msg, imageURL)
	} else {
		go l.pushMessage(msg)
	}
}

// SendDailySummary sends a daily order summary to the LINE group
func (l *LineNotifier) SendDailySummary() {
	if !l.enabled {
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
		"\U0001F4CA Daily Summary (%s)\n"+
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

	l.pushMessage(msg)
}

func (l *LineNotifier) pushMessageWithImage(text string, imageURL string) {
	body := map[string]interface{}{
		"to": l.groupID,
		"messages": []interface{}{
			map[string]string{"type": "text", "text": text},
			map[string]string{
				"type":               "image",
				"originalContentUrl": imageURL,
				"previewImageUrl":    imageURL,
			},
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		log.Printf("LINE notify marshal error: %v", err)
		return
	}

	req, err := http.NewRequest("POST", "https://api.line.me/v2/bot/message/push", bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Printf("LINE notify request error: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+l.channelToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("LINE notify send error: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("LINE notify response: %d", resp.StatusCode)
	}
}

func (l *LineNotifier) pushMessage(text string) {
	body := map[string]interface{}{
		"to": l.groupID,
		"messages": []map[string]string{
			{"type": "text", "text": text},
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		log.Printf("LINE notify marshal error: %v", err)
		return
	}

	req, err := http.NewRequest("POST", "https://api.line.me/v2/bot/message/push", bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Printf("LINE notify request error: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+l.channelToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("LINE notify send error: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("LINE notify response: %d", resp.StatusCode)
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
		lines = append(lines, fmt.Sprintf("  - %s x%d (%.2f)", name, item.Quantity, item.Price*float64(item.Quantity)))
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

		lines = append(lines, fmt.Sprintf("  - %s: %d%s", product.Name, product.Stock, warning))
	}

	if len(lines) == 0 {
		return "  (no products)"
	}
	return strings.Join(lines, "\n")
}
