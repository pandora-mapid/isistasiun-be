package response

import "github.com/gofiber/fiber/v2"

// Envelope matches the standardized JSON response shape:
// { success, message, data }.
type Envelope struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func OK(c *fiber.Ctx, data interface{}) error {
	return c.Status(fiber.StatusOK).JSON(Envelope{
		Success: true,
		Message: "Berhasil mengambil data",
		Data:    data,
	})
}

func OKWithMessage(c *fiber.Ctx, message string, data interface{}) error {
	return c.Status(fiber.StatusOK).JSON(Envelope{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func Created(c *fiber.Ctx, data interface{}) error {
	return c.Status(fiber.StatusCreated).JSON(Envelope{
		Success: true,
		Message: "Berhasil membuat data",
		Data:    data,
	})
}

func CreatedWithMessage(c *fiber.Ctx, message string, data interface{}) error {
	return c.Status(fiber.StatusCreated).JSON(Envelope{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func Fail(c *fiber.Ctx, status int, code, message string) error {
	return c.Status(status).JSON(Envelope{
		Success: false,
		Message: message,
		Data:    nil,
	})
}

// Common shortcuts
func BadRequest(c *fiber.Ctx, message string) error {
	return Fail(c, fiber.StatusBadRequest, "BAD_REQUEST", message)
}

func Unauthorized(c *fiber.Ctx, message string) error {
	return Fail(c, fiber.StatusUnauthorized, "UNAUTHORIZED", message)
}

func Forbidden(c *fiber.Ctx, message string) error {
	return Fail(c, fiber.StatusForbidden, "FORBIDDEN", message)
}

func NotFound(c *fiber.Ctx, message string) error {
	return Fail(c, fiber.StatusNotFound, "NOT_FOUND", message)
}

func Internal(c *fiber.Ctx, message string) error {
	return Fail(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", message)
}

func TooManyRequests(c *fiber.Ctx, message string) error {
	return Fail(c, fiber.StatusTooManyRequests, "RATE_LIMITED", message)
}
