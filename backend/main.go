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

	// Telegram Notifier
	telegramNotifier := services.NewTelegramNotifier(cfg)

	// Public routes
	authHandler := handlers.NewAuthHandler(cfg)
	app.Post("/api/login", authHandler.Login)

	// Public storefront routes (no auth) — product browsing + customer checkout
	shopHandler := handlers.NewShopHandler(cfg, telegramNotifier)
	app.Get("/api/shop/products", shopHandler.Products)
	app.Get("/api/shop/products/:id", shopHandler.Product)
	app.Post("/api/shop/orders", shopHandler.Checkout)
	app.Get("/api/shop/site-images", shopHandler.SiteImages)

	// Protected routes
	api := app.Group("/api", middleware.JWTAuth(cfg))

	// Change Password
	api.Put("/change-password", authHandler.ChangePassword)

	// Dashboard
	dashboardHandler := handlers.NewDashboardHandler()
	api.Get("/dashboard", dashboardHandler.Stats)
	api.Get("/dashboard/charts", dashboardHandler.Charts)

	// Products
	productHandler := handlers.NewProductHandler(cfg)
	api.Get("/products", productHandler.List)
	api.Get("/products/:id", productHandler.Get)
	api.Post("/products", productHandler.Create)
	api.Put("/products/:id", productHandler.Update)
	api.Delete("/products/:id", productHandler.Delete)
	api.Post("/products/:id/images", productHandler.UploadImages)
	api.Delete("/products/:id/images", productHandler.DeleteImage)

	// Site Images (editable storefront hero/lookbook/journal)
	siteImageHandler := handlers.NewSiteImageHandler(cfg)
	api.Get("/site-images", siteImageHandler.List)
	api.Post("/site-images/:key/image", siteImageHandler.UploadImage)
	api.Put("/site-images/:key", siteImageHandler.UpdateCaptions)

	// Customers
	customerHandler := handlers.NewCustomerHandler()
	api.Get("/customers", customerHandler.List)
	api.Get("/customers/:id", customerHandler.Get)
	api.Post("/customers", customerHandler.Create)
	api.Put("/customers/:id", customerHandler.Update)
	api.Delete("/customers/:id", customerHandler.Delete)

	// Orders
	orderHandler := handlers.NewOrderHandler(cfg, telegramNotifier)
	api.Get("/orders", orderHandler.List)
	// Accounting export — must be registered before "/orders/:id" so the literal
	// path isn't swallowed by the :id param route.
	api.Get("/orders/export.csv", orderHandler.ExportCSV)
	api.Get("/orders/:id", orderHandler.Get)
	api.Post("/orders", orderHandler.Create)
	api.Put("/orders/:id/status", orderHandler.UpdateStatus)
	api.Post("/orders/:id/slip", orderHandler.UploadSlip)
	api.Delete("/orders/:id", orderHandler.Delete)

	// Receipts (ใบเสร็จรับเงิน) — running number, persisted history
	receiptHandler := handlers.NewReceiptHandler()
	api.Get("/receipts", receiptHandler.List)
	api.Get("/orders/:id/receipt", receiptHandler.Get)
	api.Post("/orders/:id/receipt", receiptHandler.Issue)

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
			telegramNotifier.SendDailySummary()
		}
	}()

	// Manual trigger for daily summary (protected)
	api.Post("/daily-summary", func(c *fiber.Ctx) error {
		go telegramNotifier.SendDailySummary()
		return c.JSON(fiber.Map{"message": "daily summary sent"})
	})

	log.Printf("Server starting on port %s", cfg.Port)
	log.Fatal(app.Listen(":" + cfg.Port))
}
