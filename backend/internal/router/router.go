package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/list-pandora/isi-stasiun-backend/internal/analytics"
	"github.com/list-pandora/isi-stasiun-backend/internal/auth"
	"github.com/list-pandora/isi-stasiun-backend/internal/config"
	"github.com/list-pandora/isi-stasiun-backend/internal/confidence"
	"github.com/list-pandora/isi-stasiun-backend/internal/copilot"
	"github.com/list-pandora/isi-stasiun-backend/internal/middleware"
	"github.com/list-pandora/isi-stasiun-backend/internal/pipeline"
	"github.com/list-pandora/isi-stasiun-backend/internal/premium"
	"github.com/list-pandora/isi-stasiun-backend/internal/station"
	"github.com/list-pandora/isi-stasiun-backend/internal/survey"
	"github.com/list-pandora/isi-stasiun-backend/internal/transparency"
)

// Setup wires every module's handler onto /api/v1, matching the ownership
// split in BACKEND_TASK_DIVISION_3_PERSON.md. Each module keeps its own
// RegisterRoutes so merge conflicts stay confined to this one file.
func Setup(app *fiber.App, db *pgxpool.Pool, cfg *config.Config) {
	api := app.Group("/api/v1")

	// ---- Priyapta: public map/analytics/confidence (no auth — public tier) ----
	stationHandler := station.NewHandler(station.NewService(station.NewRepository(db)))
	stationHandler.RegisterRoutes(api)

	analyticsHandler := analytics.NewHandler(analytics.NewService(analytics.NewRepository(db)))
	analyticsHandler.RegisterRoutes(api)

	confidenceHandler := confidence.NewHandler(confidence.NewService(confidence.NewRepository(db)))
	confidenceHandler.RegisterRoutes(api)

	// ---- Priyapta: AI copilot routing (rate-limited, public) ----
	copilotClient := copilot.NewClient(cfg.AIServiceURL, cfg.AIServiceTimeoutSeconds)
	copilotHandler := copilot.NewHandler(copilot.NewService(copilotClient))
	copilotGroup := api.Group("", middleware.RateLimit(cfg.RateLimitRPM))
	copilotHandler.RegisterRoutes(copilotGroup)

	// ---- Arzaka: survey ingestion + pipeline callbacks (service-key protected) ----
	surveyHandler := survey.NewHandler(survey.NewService(survey.NewRepository(db)))
	serviceGroup := api.Group("", middleware.RequireServiceKey(cfg.PipelineServiceAPIKey))
	surveyHandler.RegisterRoutes(serviceGroup)

	pipelineHandler := pipeline.NewHandler(pipeline.NewService(pipeline.NewRepository(db)))
	pipelineHandler.RegisterRoutes(serviceGroup)

	// ---- Firaz: transparency (public — this is the point of the panel) ----
	transparencyHandler := transparency.NewHandler(transparency.NewService(transparency.NewRepository(db)))
	transparencyHandler.RegisterRoutes(api)

	// ---- Firaz: auth ----
	authHandler := auth.NewHandler(auth.NewService(auth.NewRepository(db), cfg.JWTSecret, cfg.JWTAccessTTLMins, cfg.JWTRefreshTTLHours))
	authHandler.RegisterRoutes(api)

	// ---- Firaz: premium (JWT + operator role required) ----
	premiumHandler := premium.NewHandler(db)
	premiumGroup := api.Group("", middleware.RequireAuth(cfg.JWTSecret), middleware.RequireRole(string(auth.RoleOperator), string(auth.RoleAdmin)))
	premiumHandler.RegisterRoutes(premiumGroup)

	// ---- Health check ----
	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
}
