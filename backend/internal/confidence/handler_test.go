package confidence

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"

	"github.com/list-pandora/isi-stasiun-backend/internal/response"
)

type stubConfidenceService struct{}

func (*stubConfidenceService) List(context.Context, string) ([]LayerEntry, error) {
	return make([]LayerEntry, 0), nil
}

func TestConfidenceRejectsInvalidStationID(t *testing.T) {
	app := fiber.New()
	NewHandler(&stubConfidenceService{}).RegisterRoutes(app)

	res, err := app.Test(httptest.NewRequest("GET", "/confidence-layer?station_id=invalid", nil))
	require.NoError(t, err)
	require.Equal(t, fiber.StatusBadRequest, res.StatusCode)
}

func TestConfidenceReturnsEmptyArray(t *testing.T) {
	app := fiber.New()
	NewHandler(&stubConfidenceService{}).RegisterRoutes(app)

	res, err := app.Test(httptest.NewRequest("GET", "/confidence-layer", nil))
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, res.StatusCode)

	var body response.Envelope
	require.NoError(t, json.NewDecoder(res.Body).Decode(&body))
	require.Equal(t, []interface{}{}, body.Data)
}
