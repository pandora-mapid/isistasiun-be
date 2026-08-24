package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"

	"github.com/list-pandora/isi-stasiun-backend/internal/response"
)

// RateLimit throttles per-IP requests. Used mainly to protect the AI
// copilot route and public endpoints from abuse.
func RateLimit(requestsPerMinute int) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        requestsPerMinute,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return response.TooManyRequests(c, "too many requests, slow down")
		},
	})
}
