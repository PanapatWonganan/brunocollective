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
	"github.com/gofiber/websocket/v2"
	"github.com/golang-jwt/jwt/v5"
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

	// Chat inbox platforms + realtime hub for the admin UI
	lineClient := services.NewLineClient(cfg)
	metaClient := services.NewMetaClient(cfg)
	chatHub := services.NewChatHub()

	// Public routes
	authHandler := handlers.NewAuthHandler(cfg)
	app.Post("/api/login", authHandler.Login)

	// Public storefront routes (no auth) — product browsing + customer checkout
	shopHandler := handlers.NewShopHandler(cfg, telegramNotifier)
	app.Get("/api/shop/products", shopHandler.Products)
	// "suggest" is a literal path — must be registered before ":id".
	app.Get("/api/shop/products/suggest", shopHandler.Suggest)
	app.Get("/api/shop/products/:id", shopHandler.Product)
	app.Post("/api/shop/orders", shopHandler.Checkout)
	app.Get("/api/shop/site-images", shopHandler.SiteImages)

	// Public coupon preview — the storefront checks a code before checkout.
	couponHandler := handlers.NewCouponHandler()
	app.Post("/api/shop/coupons/validate", couponHandler.Validate)

	// Storefront membership — register/login by phone; members get a flat 5%
	// discount on every order, stacked separately from coupons.
	memberHandler := handlers.NewMemberHandler(cfg)
	app.Post("/api/shop/members/register", memberHandler.Register)
	app.Post("/api/shop/members/login", memberHandler.Login)
	app.Post("/api/shop/members/check", memberHandler.Check)
	app.Get("/api/shop/members/me", middleware.MemberAuth(cfg), memberHandler.Me)
	app.Put("/api/shop/members/me", middleware.MemberAuth(cfg), memberHandler.UpdateMe)
	app.Get("/api/shop/members/me/orders", middleware.MemberAuth(cfg), memberHandler.MyOrders)

	// Public sale/landing pages — rendered by the storefront at /s/{slug}.
	salePageHandler := handlers.NewSalePageHandler(cfg, telegramNotifier)
	app.Get("/api/shop/sale-pages/:slug", salePageHandler.PublicGet)
	app.Post("/api/shop/sale-pages/:slug/order", salePageHandler.PublicOrder)

	// Public payment page — token-scoped order summary + slip upload for
	// orders created from the chat inbox ("สร้างออเดอร์จากแชท"). The token is
	// unguessable and the response carries no phone/address.
	orderHandler := handlers.NewOrderHandler(cfg, telegramNotifier)
	app.Get("/api/pay/:token", orderHandler.PayGet)
	app.Post("/api/pay/:token/slip", orderHandler.PayUploadSlip)

	// AI chat assistant (Claude) — answers inbox questions from live stock
	// when no keyword rule matches. Disabled without ANTHROPIC_API_KEY.
	aiClient := services.NewAIClient(cfg)

	// Auto-reply engine — FB/IG comments (via the Meta webhook) plus keyword
	// replies + AI answers in LINE/FB/IG chats.
	autoReplyHandler := handlers.NewAutoReplyHandler(metaClient, lineClient, chatHub, aiClient)

	// Platform webhooks (no auth — verified by platform signature).
	webhookHandler := handlers.NewWebhookHandler(cfg, lineClient, metaClient, chatHub, autoReplyHandler)
	app.Post("/api/webhooks/line", webhookHandler.LineWebhook)
	// One Meta endpoint serves both Facebook Messenger and Instagram DM.
	app.Get("/api/webhooks/meta", webhookHandler.MetaWebhookVerify)
	app.Post("/api/webhooks/meta", webhookHandler.MetaWebhook)

	// Admin chat WebSocket — JWT passed as ?token= because browsers can't set
	// headers on WebSocket connects.
	app.Use("/api/ws/chat", func(c *fiber.Ctx) error {
		if !websocket.IsWebSocketUpgrade(c) {
			return fiber.ErrUpgradeRequired
		}
		tokenStr := c.Query("token")
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.ErrUnauthorized
			}
			return []byte(cfg.JWTSecret), nil
		})
		if err != nil || !token.Valid {
			return fiber.ErrUnauthorized
		}
		// Storefront member tokens share the signing secret — keep them out
		// of the admin chat socket.
		if claims, ok := token.Claims.(jwt.MapClaims); !ok || claims["user_id"] == nil {
			return fiber.ErrUnauthorized
		}
		return c.Next()
	})
	app.Get("/api/ws/chat", websocket.New(func(conn *websocket.Conn) {
		chatHub.Register(conn)
		defer func() {
			chatHub.Unregister(conn)
			conn.Close()
		}()
		// Read loop only detects disconnects — admins receive, they don't send.
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))

	// Protected routes
	api := app.Group("/api", middleware.JWTAuth(cfg))

	// Change Password
	api.Put("/change-password", authHandler.ChangePassword)

	// Dashboard
	dashboardHandler := handlers.NewDashboardHandler()
	api.Get("/dashboard", dashboardHandler.Stats)
	api.Get("/dashboard/charts", dashboardHandler.Charts)
	api.Get("/notifications", dashboardHandler.Notifications)

	// Analytics (dashboard tabs: overview / inventory / customers / products)
	analyticsHandler := handlers.NewAnalyticsHandler()
	api.Get("/analytics/overview", analyticsHandler.Overview)
	api.Get("/analytics/inventory", analyticsHandler.Inventory)
	api.Get("/analytics/customers", analyticsHandler.Customers)
	api.Get("/analytics/products", analyticsHandler.Products)
	api.Get("/analytics/chats", analyticsHandler.Chats)

	// Products
	productHandler := handlers.NewProductHandler(cfg)
	api.Get("/products", productHandler.List)
	// NOTE: must be registered before /products/:id so "reorder" isn't parsed as an id.
	api.Put("/products/reorder", productHandler.Reorder)
	api.Get("/products/:id", productHandler.Get)
	api.Post("/products", productHandler.Create)
	api.Put("/products/:id", productHandler.Update)
	api.Delete("/products/:id", productHandler.Delete)
	api.Post("/products/:id/merge", productHandler.Merge)
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
	api.Post("/customers/:id/member", customerHandler.ToggleMember)

	// Orders (orderHandler constructed above, with the public payment routes)
	api.Get("/orders", orderHandler.List)
	// Accounting export — must be registered before "/orders/:id" so the literal
	// path isn't swallowed by the :id param route.
	api.Get("/orders/export.csv", orderHandler.ExportCSV)
	api.Get("/orders/:id", orderHandler.Get)
	api.Post("/orders", orderHandler.Create)
	api.Put("/orders/:id/status", orderHandler.UpdateStatus)
	api.Post("/orders/:id/slip", orderHandler.UploadSlip)
	// Follow-up discount for a stale unpaid order (single-use coupon, applied
	// immediately so the /pay page updates without the customer typing a code).
	api.Post("/orders/:id/followup-discount", orderHandler.FollowupDiscount)
	api.Delete("/orders/:id", orderHandler.Delete)

	// Coupons — literal routes before "/coupons/:id" so they aren't swallowed
	// by the param route (same reason as orders/export.csv above).
	api.Get("/coupons", couponHandler.List)
	api.Post("/coupons", couponHandler.Create)
	api.Post("/coupons/validate", couponHandler.Validate)
	api.Get("/coupons/:id", couponHandler.Get)
	api.Put("/coupons/:id", couponHandler.Update)
	api.Delete("/coupons/:id", couponHandler.Delete)
	api.Post("/coupons/:id/toggle", couponHandler.Toggle)
	api.Get("/coupons/:id/redemptions", couponHandler.Redemptions)

	// Sale pages (funnel builder) — "upload" before ":id" so it isn't swallowed.
	api.Get("/sale-pages", salePageHandler.List)
	api.Post("/sale-pages", salePageHandler.Create)
	api.Post("/sale-pages/upload", salePageHandler.UploadImage)
	api.Get("/sale-pages/:id", salePageHandler.Get)
	api.Put("/sale-pages/:id", salePageHandler.Update)
	api.Delete("/sale-pages/:id", salePageHandler.Delete)
	api.Post("/sale-pages/:id/duplicate", salePageHandler.Duplicate)
	api.Post("/sale-pages/:id/toggle", salePageHandler.TogglePublish)

	// Chat inbox (LINE + Facebook + Instagram). "summary" before ":id" so it
	// isn't swallowed by the param route.
	chatHandler := handlers.NewChatHandler(cfg, lineClient, metaClient, chatHub)
	api.Get("/chats", chatHandler.List)
	api.Get("/chats/summary", chatHandler.Summary)
	// "ดีลค้าง" — conversations with unpaid chat-created orders. Literal path,
	// so it must be registered before the ":id" routes below.
	api.Get("/chats/followups", orderHandler.ChatFollowups)
	api.Get("/chats/:id/messages", chatHandler.Messages)
	api.Post("/chats/:id/reply", chatHandler.Reply)
	// Create an order for the conversation's linked customer and get back a
	// payment link (/pay/{token}) to push into the chat.
	api.Post("/chats/:id/order", orderHandler.CreateFromChat)
	api.Post("/chats/:id/read", chatHandler.MarkRead)
	api.Put("/chats/:id/status", chatHandler.UpdateStatus)
	api.Put("/chats/:id/tags", chatHandler.UpdateTags)
	api.Put("/chats/:id/customer", chatHandler.LinkCustomer)
	// Per-thread AI assistant on/off toggle.
	api.Post("/chats/:id/ai", chatHandler.ToggleAI)
	// Manual "answered elsewhere" — LINE OA app replies never reach webhooks.
	api.Post("/chats/:id/answered", chatHandler.MarkAnswered)

	// Comment auto-reply rules — "logs" before ":id" so it isn't swallowed.
	api.Get("/auto-replies", autoReplyHandler.List)
	api.Get("/auto-replies/logs", autoReplyHandler.Logs)
	api.Post("/auto-replies", autoReplyHandler.Create)
	api.Put("/auto-replies/:id", autoReplyHandler.Update)
	api.Delete("/auto-replies/:id", autoReplyHandler.Delete)
	api.Post("/auto-replies/:id/toggle", autoReplyHandler.Toggle)

	// LINE broadcast by RFM segment (แคมเปญหาลูกค้าตามกลุ่ม)
	broadcastHandler := handlers.NewBroadcastHandler(lineClient)
	api.Get("/broadcasts", broadcastHandler.List)
	api.Get("/broadcasts/audience", broadcastHandler.Audience)
	api.Post("/broadcasts", broadcastHandler.Send)

	// Canned replies (ข้อความสำเร็จรูปในแชท)
	api.Get("/canned-replies", chatHandler.CannedList)
	api.Post("/canned-replies", chatHandler.CannedCreate)
	api.Put("/canned-replies/:id", chatHandler.CannedUpdate)
	api.Delete("/canned-replies/:id", chatHandler.CannedDelete)

	// Receipts (ใบเสร็จรับเงิน) — running number, persisted history
	receiptHandler := handlers.NewReceiptHandler()
	api.Get("/receipts", receiptHandler.List)
	api.Get("/orders/:id/receipt", receiptHandler.Get)
	api.Post("/orders/:id/receipt", receiptHandler.Issue)

	// Chat SLA watcher — Telegram alert when chats wait too long for a reply.
	handlers.StartChatSLAWatcher(cfg, telegramNotifier)

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
