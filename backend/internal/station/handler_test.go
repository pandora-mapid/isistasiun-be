package station

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"

	"github.com/list-pandora/isi-stasiun-backend/internal/response"
)

type stubStationService struct {
	listCalled bool
}

func (s *stubStationService) ListStations(context.Context, string, string) ([]StationResponse, error) {
	s.listCalled = true
	return make([]StationResponse, 0), nil
}

func (*stubStationService) GetStation(context.Context, string) (*StationResponse, error) {
	return nil, ErrNotFound
}

func (*stubStationService) ListEntrances(context.Context, string) ([]EntranceResponse, error) {
	return make([]EntranceResponse, 0), nil
}

func TestListStationsRejectsInvalidAreaType(t *testing.T) {
	svc := &stubStationService{}
	app := fiber.New()
	NewHandler(svc).RegisterRoutes(app)

	res, err := app.Test(httptest.NewRequest("GET", "/stations?area_type=airport", nil))
	require.NoError(t, err)
	require.Equal(t, fiber.StatusBadRequest, res.StatusCode)
	require.False(t, svc.listCalled)
}

func TestListStationsReturnsEmptyArray(t *testing.T) {
	svc := &stubStationService{}
	app := fiber.New()
	NewHandler(svc).RegisterRoutes(app)

	res, err := app.Test(httptest.NewRequest("GET", "/stations?area_type=mixed", nil))
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, res.StatusCode)

	var body response.Envelope
	require.NoError(t, json.NewDecoder(res.Body).Decode(&body))
	require.Equal(t, []interface{}{}, body.Data)
	require.True(t, svc.listCalled)
}
