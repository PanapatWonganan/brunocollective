package middleware

import (
	"strings"

	"brunocollective_inventory/config"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// parseBearerToken extracts and verifies the JWT from the Authorization
// header, returning its claims.
func parseBearerToken(c *fiber.Ctx, cfg *config.Config) (jwt.MapClaims, error) {
	auth := c.Get("Authorization")
	if auth == "" {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "missing authorization header")
	}

	tokenStr := strings.TrimPrefix(auth, "Bearer ")
	if tokenStr == auth {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "invalid authorization format")
	}

	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fiber.ErrUnauthorized
		}
		return []byte(cfg.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "invalid or expired token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "invalid token claims")
	}
	return claims, nil
}

// JWTAuth guards the admin API. Storefront member tokens (role=member) are
// valid JWTs signed with the same secret, so they must be rejected explicitly
// — only admin tokens (which carry user_id and no role) pass.
func JWTAuth(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, err := parseBearerToken(c, cfg)
		if err != nil {
			fe := err.(*fiber.Error)
			return c.Status(fe.Code).JSON(fiber.Map{"error": fe.Message})
		}

		if role, _ := claims["role"].(string); role == "member" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
		}
		if claims["user_id"] == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token claims"})
		}

		c.Locals("user_id", claims["user_id"])
		c.Locals("username", claims["username"])
		return c.Next()
	}
}

// MemberAuth guards the storefront member routes. Only tokens with
// role=member pass; the customer ID lands in c.Locals("customer_id").
func MemberAuth(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, err := parseBearerToken(c, cfg)
		if err != nil {
			fe := err.(*fiber.Error)
			return c.Status(fe.Code).JSON(fiber.Map{"error": fe.Message})
		}

		role, _ := claims["role"].(string)
		id, ok := claims["customer_id"].(float64)
		if role != "member" || !ok || id <= 0 {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid member token"})
		}

		c.Locals("customer_id", uint(id))
		return c.Next()
	}
}
