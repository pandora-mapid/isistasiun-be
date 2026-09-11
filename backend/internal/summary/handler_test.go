package summary

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"

	"github.com/list-pandora/isi-stasiun-backend/internal/response"
)

type stubService struct{ rows []StationSummaryResponse }

func (s *stubService) StationSummary(context.Context, string) ([]StationSummaryResponse, error) {
	return s.rows, nil
}

func TestStationSummaryRejectsInvalidStationID(t *testing.T) {
	app := fiber.New()
	NewHandler(&stubService{}).RegisterRoutes(app)

	res, err := app.Test(httptest.NewRequest("GET", "/analytics/station-summary/not-a-uuid", nil))
	require.NoError(t, err)
	require.Equal(t, fiber.StatusBadRequest, res.StatusCode)
}

func TestStationSummaryAllReturnsEnvelope(t *testing.T) {
	rate := 0.31
	app := fiber.New()
	NewHandler(&stubService{rows: []StationSummaryResponse{
		{
			StationID: "11111111-1111-1111-1111-111111111111", StationName: "Manggarai",
			Typology: "mixed", DayType: "weekday", PintuDicacah: 3,
			Gap:         MoneyRange{P10: 5_000_000, P50: 5_700_000, P90: 7_300_000},
			CaptureRate: &rate, Basis: "monte-carlo-simpul",
			Composition: []CategoryComposition{{Category: "apotek_kesehatan", DemandShare: 0.11, IsMissing: true}},
		},
	}}).RegisterRoutes(app)

	res, err := app.Test(httptest.NewRequest("GET", "/analytics/station-summary", nil))
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, res.StatusCode)

	var body response.Envelope
	require.NoError(t, json.NewDecoder(res.Body).Decode(&body))
	require.True(t, body.Success)

	rows, ok := body.Data.([]interface{})
	require.True(t, ok)
	require.Len(t, rows, 1)

	first := rows[0].(map[string]interface{})
	require.Equal(t, "Manggarai", first["station_name"])
	require.Equal(t, "monte-carlo-simpul", first["basis"])
	require.Equal(t, float64(5_700_000), first["gap"].(map[string]interface{})["p50"])
}

func TestStationSummaryEmptyIsArrayNotNull(t *testing.T) {
	app := fiber.New()
	NewHandler(&stubService{rows: make([]StationSummaryResponse, 0)}).RegisterRoutes(app)

	res, err := app.Test(httptest.NewRequest("GET", "/analytics/station-summary", nil))
	require.NoError(t, err)

	var body response.Envelope
	require.NoError(t, json.NewDecoder(res.Body).Decode(&body))
	require.Equal(t, []interface{}{}, body.Data)
}
