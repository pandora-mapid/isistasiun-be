package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

func RequestLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		status := c.Response().StatusCode()

		// TODO: swap for structured logger (zerolog/zap) before prod.
		println(
			c.Method(), c.Path(), status,
			time.Since(start).String(),
		)
		return err
	}
}
