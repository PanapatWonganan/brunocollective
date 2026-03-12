package main

import (
	"log"
	"os"
	"time"

	"brunocollective_inventory/config"
	"brunocollective_inventory/database"
	"brunocollective_inventory/handlers"
	"brunocollective_inventory/middleware"
	"brunocollective_inventory/services"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func main() {
	cfg := config.Load()

	// Ensure upload directory exists
	if err := os.MkdirAll(cfg.UploadDir, 0755); err != nil {
		log.Fatal("Failed to create upload directory:", err)
	}

	database.Connect(cfg)

	app := fiber.New(fiber.Config{
		BodyLimit: 10 * 1024 * 1024, // 10MB limit for slip uploads
	})

	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "http://localhost:5173, http://localhost:3000",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	// Serve uploaded files
	app.Static("/uploads", cfg.UploadDir)

	// LINE Webhook (public - called by LINE platform)
	app.Post("/webhook/line", func(c *fiber.Ctx) error {
		rawBody := string(c.Body())
		log.Printf("LINE Webhook received: %s", rawBody)

		var body struct {
			Events []struct {
				Type   string `json:"type"`
				Source struct {
					Type    string `json:"type"`
					GroupID string `json:"groupId"`
				} `json:"source"`
			} `json:"events"`
		}
		if err := c.BodyParser(&body); err != nil {
			log.Printf("LINE Webhook parse error: %v", err)
			return c.SendStatus(200)
		}
		log.Printf("LINE Webhook events count: %d", len(body.Events))
		for _, event := range body.Events {
			log.Printf("LINE event type=%s source_type=%s groupID=%s", event.Type, event.Source.Type, event.Source.GroupID)
			if event.Source.GroupID != "" {
				log.Printf("========================================")
				log.Printf("LINE GROUP ID: %s", event.Source.GroupID)
				log.Printf("========================================")
			}
		}
		return c.SendStatus(200)
	})

	// Public routes
	authHandler := handlers.NewAuthHandler(cfg)
	app.Post("/api/login", authHandler.Login)

	// Protected routes
	api := app.Group("/api", middleware.JWTAuth(cfg))

	// Change Password
	api.Put("/change-password", authHandler.ChangePassword)

	// Dashboard
	dashboardHandler := handlers.NewDashboardHandler()
	api.Get("/dashboard", dashboardHandler.Stats)
	api.Get("/dashboard/charts", dashboardHandler.Charts)

	// Products
	productHandler := handlers.NewProductHandler()
	api.Get("/products", productHandler.List)
	api.Get("/products/:id", productHandler.Get)
	api.Post("/products", productHandler.Create)
	api.Put("/products/:id", productHandler.Update)
	api.Delete("/products/:id", productHandler.Delete)

	// Customers
	customerHandler := handlers.NewCustomerHandler()
	api.Get("/customers", customerHandler.List)
	api.Get("/customers/:id", customerHandler.Get)
	api.Post("/customers", customerHandler.Create)
	api.Put("/customers/:id", customerHandler.Update)
	api.Delete("/customers/:id", customerHandler.Delete)

	// LINE Notifier
	lineNotifier := services.NewLineNotifier(cfg)

	// Orders
	orderHandler := handlers.NewOrderHandler(cfg, lineNotifier)
	api.Get("/orders", orderHandler.List)
	api.Get("/orders/:id", orderHandler.Get)
	api.Post("/orders", orderHandler.Create)
	api.Put("/orders/:id/status", orderHandler.UpdateStatus)
	api.Post("/orders/:id/slip", orderHandler.UploadSlip)
	api.Delete("/orders/:id", orderHandler.Delete)

	// Daily summary scheduler (8:00 AM Bangkok time)
	go func() {
		loc, _ := time.LoadLocation("Asia/Bangkok")
		for {
			now := time.Now().In(loc)
			next := time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, loc)
			if now.After(next) {
				next = next.Add(24 * time.Hour)
			}
			time.Sleep(time.Until(next))
			log.Println("Sending daily summary...")
			lineNotifier.SendDailySummary()
		}
	}()

	// Manual trigger for daily summary (protected)
	api.Post("/daily-summary", func(c *fiber.Ctx) error {
		go lineNotifier.SendDailySummary()
		return c.JSON(fiber.Map{"message": "daily summary sent"})
	})

	log.Printf("Server starting on port %s", cfg.Port)
	log.Fatal(app.Listen(":" + cfg.Port))
}
