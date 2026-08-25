package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
)

func TestRateLimitRejectsRequestsOverLimit(t *testing.T) {
	app := fiber.New()
	app.Get("/limited", RateLimit(1), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})

	first, err := app.Test(httptest.NewRequest("GET", "/limited", nil))
	require.NoError(t, err)
	require.Equal(t, fiber.StatusNoContent, first.StatusCode)

	second, err := app.Test(httptest.NewRequest("GET", "/limited", nil))
	require.NoError(t, err)
	require.Equal(t, fiber.StatusTooManyRequests, second.StatusCode)
}
