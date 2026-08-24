package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/list-pandora/isi-stasiun-backend/internal/config"
	"github.com/list-pandora/isi-stasiun-backend/internal/database"
	"github.com/list-pandora/isi-stasiun-backend/internal/middleware"
	"github.com/list-pandora/isi-stasiun-backend/internal/response"
	"github.com/list-pandora/isi-stasiun-backend/internal/router"
)

func main() {
	cfg := config.Load()

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	app := fiber.New(fiber.Config{
		AppName:      "Isi Stasiun API",
		ErrorHandler: fiberErrorHandler,
	})

	app.Use(recover.New())
	app.Use(middleware.RequestLogger())
	app.Use(middleware.CORS(cfg.CORSAllowedOrigins))

	router.Setup(app, db, cfg)

	log.Printf("Isi Stasiun API listening on :%s [%s]", cfg.AppPort, cfg.AppEnv)
	if err := app.Listen(":" + cfg.AppPort); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func fiberErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}
	return response.Fail(c, code, "ERROR", err.Error())
}
