package main

import (
	"log"
	"os"

	"brunocollective_inventory/config"
	"brunocollective_inventory/database"
	"brunocollective_inventory/handlers"
	"brunocollective_inventory/middleware"

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

	// Public routes
	authHandler := handlers.NewAuthHandler(cfg)
	app.Post("/api/login", authHandler.Login)

	// Protected routes
	api := app.Group("/api", middleware.JWTAuth(cfg))

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

	// Orders
	orderHandler := handlers.NewOrderHandler(cfg)
	api.Get("/orders", orderHandler.List)
	api.Get("/orders/:id", orderHandler.Get)
	api.Post("/orders", orderHandler.Create)
	api.Put("/orders/:id/status", orderHandler.UpdateStatus)
	api.Post("/orders/:id/slip", orderHandler.UploadSlip)
	api.Delete("/orders/:id", orderHandler.Delete)

	log.Printf("Server starting on port %s", cfg.Port)
	log.Fatal(app.Listen(":" + cfg.Port))
}
