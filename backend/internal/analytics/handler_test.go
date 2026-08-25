package analytics

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"

	"github.com/list-pandora/isi-stasiun-backend/internal/response"
)

type stubAnalyticsService struct{}

func (*stubAnalyticsService) SpendingGap(context.Context, string) ([]SpendingGapResponse, error) {
	return make([]SpendingGapResponse, 0), nil
}

func (*stubAnalyticsService) CategoryGap(context.Context, string) ([]CategoryGapResponse, error) {
	return make([]CategoryGapResponse, 0), nil
}

func (*stubAnalyticsService) RentFlowIndex(context.Context, string) ([]RentFlowIndexResponse, error) {
	return make([]RentFlowIndexResponse, 0), nil
}

func (*stubAnalyticsService) EventPotential(context.Context, string) ([]EventPotentialResponse, error) {
	return make([]EventPotentialResponse, 0), nil
}

func TestAnalyticsRejectsInvalidStationID(t *testing.T) {
	app := fiber.New()
	NewHandler(&stubAnalyticsService{}).RegisterRoutes(app)

	paths := []string{
		"/analytics/spending-gap/not-a-uuid",
		"/analytics/category-gap?station_id=not-a-uuid",
		"/analytics/rent-flow-index?station_id=not-a-uuid",
		"/analytics/event-potential?station_id=not-a-uuid",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			res, err := app.Test(httptest.NewRequest("GET", path, nil))
			require.NoError(t, err)
			require.Equal(t, fiber.StatusBadRequest, res.StatusCode)
		})
	}
}

func TestSpendingGapAllReturnsEmptyArray(t *testing.T) {
	app := fiber.New()
	NewHandler(&stubAnalyticsService{}).RegisterRoutes(app)

	res, err := app.Test(httptest.NewRequest("GET", "/analytics/spending-gap", nil))
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, res.StatusCode)

	var body response.Envelope
	require.NoError(t, json.NewDecoder(res.Body).Decode(&body))
	require.Equal(t, []interface{}{}, body.Data)
}

func TestOptionalStationFilter(t *testing.T) {
	query, args := withOptionalStationFilter("SELECT * FROM result", "")
	require.Equal(t, "SELECT * FROM result", query)
	require.Empty(t, args)

	query, args = withOptionalStationFilter("SELECT * FROM result", "station-id")
	require.Equal(t, "SELECT * FROM result WHERE station_id = $1", query)
	require.Equal(t, []any{"station-id"}, args)
}
