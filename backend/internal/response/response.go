package response

import "github.com/gofiber/fiber/v2"

// Envelope matches the standard API response shape agreed in
// BACKEND_TASK_DIVISION_3_PERSON.md: { success, data, error }.
type Envelope struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Error   *ErrorBody  `json:"error"`
}

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func OK(c *fiber.Ctx, data interface{}) error {
	return c.Status(fiber.StatusOK).JSON(Envelope{
		Success: true,
		Data:    data,
		Error:   nil,
	})
}

func Created(c *fiber.Ctx, data interface{}) error {
	return c.Status(fiber.StatusCreated).JSON(Envelope{
		Success: true,
		Data:    data,
		Error:   nil,
	})
}

func Fail(c *fiber.Ctx, status int, code, message string) error {
	return c.Status(status).JSON(Envelope{
		Success: false,
		Data:    nil,
		Error: &ErrorBody{
			Code:    code,
			Message: message,
		},
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
