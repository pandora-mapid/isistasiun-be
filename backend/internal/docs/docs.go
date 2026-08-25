package docs

import (
	_ "embed"

	"github.com/gofiber/fiber/v2"
)

//go:embed openapi.yaml
var openAPISpec []byte

const swaggerUI = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Isi Stasiun API Docs</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5.17.14/swagger-ui.css">
  <style>body { margin: 0; background: #fafafa; }</style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5.17.14/swagger-ui-bundle.js"></script>
  <script>
    SwaggerUIBundle({
      url: "/docs/openapi.yaml",
      dom_id: "#swagger-ui",
      deepLinking: true,
      displayRequestDuration: true,
      persistAuthorization: true,
      tryItOutEnabled: true
    });
  </script>
</body>
</html>`

// Register exposes interactive API documentation when enabled.
func Register(app *fiber.App, enabled bool) {
	if !enabled {
		return
	}

	app.Get("/docs", func(c *fiber.Ctx) error {
		c.Type("html", "utf-8")
		return c.SendString(swaggerUI)
	})
	app.Get("/docs/openapi.yaml", func(c *fiber.Ctx) error {
		c.Set(fiber.HeaderContentType, "application/yaml; charset=utf-8")
		return c.Send(openAPISpec)
	})
}
