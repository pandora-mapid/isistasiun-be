package docs

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestRegisterEnabled(t *testing.T) {
	app := fiber.New()
	Register(app, true)

	uiResponse, err := app.Test(httptest.NewRequest("GET", "/docs", nil))
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, uiResponse.StatusCode)
	require.Contains(t, uiResponse.Header.Get("Content-Type"), "text/html")
	uiBody, err := io.ReadAll(uiResponse.Body)
	require.NoError(t, err)
	require.Contains(t, string(uiBody), "swagger-ui-dist@5.17.14")
	require.Contains(t, string(uiBody), "/docs/openapi.yaml")

	specResponse, err := app.Test(httptest.NewRequest("GET", "/docs/openapi.yaml", nil))
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, specResponse.StatusCode)
	require.Contains(t, specResponse.Header.Get("Content-Type"), "application/yaml")

	specBody, err := io.ReadAll(specResponse.Body)
	require.NoError(t, err)
	var spec struct {
		OpenAPI string                 `yaml:"openapi"`
		Paths   map[string]interface{} `yaml:"paths"`
	}
	require.NoError(t, yaml.Unmarshal(specBody, &spec))
	require.Equal(t, "3.0.3", spec.OpenAPI)
	require.Len(t, spec.Paths, 29)
	require.Contains(t, spec.Paths, "/healthz")
	require.Contains(t, spec.Paths, "/api/v1/copilot/query")
	require.Contains(t, spec.Paths, "/api/v1/analytics/station-summary")
	require.Contains(t, spec.Paths, "/api/v1/analytics/rental-assets")
	require.Contains(t, spec.Paths, "/api/v1/auth/logout")

	operations := 0
	for _, pathValue := range spec.Paths {
		path, ok := pathValue.(map[string]interface{})
		require.True(t, ok)
		for method := range path {
			switch method {
			case "get", "post", "put", "patch", "delete":
				operations++
			}
		}
	}
	require.Equal(t, 30, operations)
}

func TestRegisterDisabled(t *testing.T) {
	app := fiber.New()
	Register(app, false)

	for _, path := range []string{"/docs", "/docs/openapi.yaml"} {
		response, err := app.Test(httptest.NewRequest("GET", path, nil))
		require.NoError(t, err)
		require.Equal(t, fiber.StatusNotFound, response.StatusCode)
	}
}

func TestSpecContainsSecuritySchemes(t *testing.T) {
	spec := string(openAPISpec)
	require.True(t, strings.Contains(spec, "ServiceKeyAuth:"))
	require.True(t, strings.Contains(spec, "BearerAuth:"))
	require.True(t, strings.Contains(spec, "RefreshCookie:"))
}
