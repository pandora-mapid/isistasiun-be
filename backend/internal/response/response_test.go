package response_test

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"

	"github.com/list-pandora/isi-stasiun-backend/internal/response"
)

func TestResponse_OK(t *testing.T) {
	app := fiber.New()
	app.Get("/test-ok", func(c *fiber.Ctx) error {
		return response.OK(c, fiber.Map{"key": "value"})
	})

	req := httptest.NewRequest("GET", "/test-ok", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var env response.Envelope
	err = json.Unmarshal(body, &env)
	assert.NoError(t, err)
	assert.True(t, env.Success)
	assert.Equal(t, "Berhasil mengambil data", env.Message)
	assert.NotNil(t, env.Data)
}

func TestResponse_BadRequest(t *testing.T) {
	app := fiber.New()
	app.Get("/test-bad", func(c *fiber.Ctx) error {
		return response.BadRequest(c, "invalid parameter")
	})

	req := httptest.NewRequest("GET", "/test-bad", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var env response.Envelope
	err = json.Unmarshal(body, &env)
	assert.NoError(t, err)
	assert.False(t, env.Success)
	assert.Equal(t, "invalid parameter", env.Message)
	assert.Nil(t, env.Data)
}
