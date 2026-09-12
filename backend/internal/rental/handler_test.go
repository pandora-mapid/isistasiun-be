package rental

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
)

type stubService struct{}

func (*stubService) List(context.Context, ListParams) ([]AssetResponse, error) {
	return []AssetResponse{}, nil
}

func TestRejectsInvalidFilters(t *testing.T) {
	app := fiber.New()
	NewHandler(&stubService{}).RegisterRoutes(app)

	for _, path := range []string{
		"/analytics/rental-assets?station_id=not-a-uuid",
		"/analytics/rental-assets?status=unknown",
	} {
		res, err := app.Test(httptest.NewRequest("GET", path, nil))
		require.NoError(t, err)
		require.Equal(t, fiber.StatusBadRequest, res.StatusCode)
	}
}

func TestListReturnsOK(t *testing.T) {
	app := fiber.New()
	NewHandler(&stubService{}).RegisterRoutes(app)

	res, err := app.Test(httptest.NewRequest("GET", "/analytics/rental-assets?station_code=MRI", nil))
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, res.StatusCode)
}
