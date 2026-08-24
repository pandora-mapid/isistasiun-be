package middleware

import (
	"crypto/subtle"

	"github.com/gofiber/fiber/v2"

	"github.com/list-pandora/isi-stasiun-backend/internal/response"
)

// RequireServiceKey protects service-to-service endpoints (pipeline callbacks,
// survey ingestion from MAPID Apps) with a shared API key instead of user JWT.
// Owned by Arzaka per BACKEND_TASK_DIVISION_3_PERSON.md.
func RequireServiceKey(expectedKey string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		got := c.Get("X-Service-Key")
		if got == "" || subtle.ConstantTimeCompare([]byte(got), []byte(expectedKey)) != 1 {
			return response.Unauthorized(c, "invalid or missing service key")
		}
		return c.Next()
	}
}
