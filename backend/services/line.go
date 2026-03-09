package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"brunocollective_inventory/config"
	"brunocollective_inventory/database"
	"brunocollective_inventory/models"
)

type LineNotifier struct {
	channelToken string
	groupID      string
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

	msg := fmt.Sprintf(""+
		"\U0001F4E6 New Order #%d\n"+
		"-------------------------------\n"+
		"Customer: %s\n"+
		"Total: %.2f THB\n"+
		"Items:\n%s\n"+
		"-------------------------------\n"+
		"Stock Remaining:\n%s",
		order.ID,
		order.Customer.Name,
		order.TotalAmount,
		items,
		stock,
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

	msg := fmt.Sprintf(""+
		"%s Order #%d Status Updated\n"+
		"-------------------------------\n"+
		"Customer: %s\n"+
		"Status: %s\n"+
		"Total: %.2f THB\n"+
		"-------------------------------\n"+
		"Stock Remaining:\n%s",
		emoji,
		order.ID,
		order.Customer.Name,
		strings.ToUpper(string(newStatus)),
		order.TotalAmount,
		stock,
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

	go l.pushMessage(msg)
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
