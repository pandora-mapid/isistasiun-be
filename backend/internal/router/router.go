package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/list-pandora/isi-stasiun-backend/internal/analytics"
	"github.com/list-pandora/isi-stasiun-backend/internal/auth"
	"github.com/list-pandora/isi-stasiun-backend/internal/confidence"
	"github.com/list-pandora/isi-stasiun-backend/internal/config"
	"github.com/list-pandora/isi-stasiun-backend/internal/copilot"
	"github.com/list-pandora/isi-stasiun-backend/internal/docs"
	"github.com/list-pandora/isi-stasiun-backend/internal/middleware"
	"github.com/list-pandora/isi-stasiun-backend/internal/pipeline"
	"github.com/list-pandora/isi-stasiun-backend/internal/premium"
	"github.com/list-pandora/isi-stasiun-backend/internal/response"
	"github.com/list-pandora/isi-stasiun-backend/internal/station"
	"github.com/list-pandora/isi-stasiun-backend/internal/summary"
	"github.com/list-pandora/isi-stasiun-backend/internal/survey"
	"github.com/list-pandora/isi-stasiun-backend/internal/transparency"
)

// Setup wires every module's handler onto /api/v1, matching the ownership
// split in BACKEND_TASK_DIVISION_3_PERSON.md. Each module keeps its own
// RegisterRoutes so merge conflicts stay confined to this one file.
//
// Middleware is mounted with Use(<path prefix>, ...) rather than
// Group("", ...). That distinction is load-bearing: Group("") compiles to
// Use("/api/v1", ...), which applies to *every route registered after it* —
// so the service-key guard silently swallowed transparency, auth and premium
// and every one of them answered 401. Binding each guard to the path it
// actually protects makes the tier independent of registration order.
// router_test.go pins this behaviour.
func Setup(app *fiber.App, db *pgxpool.Pool, cfg *config.Config) {
	docs.Register(app, !cfg.IsProduction())

	api := app.Group("/api/v1")

	// Guards first: Fiber walks the stack in registration order, so a Use must
	// be mounted before the routes it protects.
	serviceKey := middleware.RequireServiceKey(cfg.PipelineServiceAPIKey)
	api.Use("/survey", serviceKey)   // Arzaka: survey ingestion
	api.Use("/pipeline", serviceKey) // Arzaka: batch pipeline callbacks

	api.Use("/copilot", middleware.RateLimit(cfg.RateLimitRPM)) // most abuse-prone: every call proxies a paid LLM

	api.Use("/premium",
		middleware.RequireAuth(cfg.JWTSecret),
		middleware.RequireRole(string(auth.RoleOperator), string(auth.RoleAdmin)),
	)

	// ---- Priyapta: public map/analytics/confidence (no auth — public tier) ----
	stationHandler := station.NewHandler(station.NewService(station.NewRepository(db)))
	stationHandler.RegisterRoutes(api)

	analyticsHandler := analytics.NewHandler(analytics.NewService(analytics.NewRepository(db)))
	analyticsHandler.RegisterRoutes(api)

	confidenceHandler := confidence.NewHandler(confidence.NewService(confidence.NewRepository(db)))
	confidenceHandler.RegisterRoutes(api)

	// ---- Arzaka: station-level rollup for the Compare View (public tier) ----
	summaryHandler := summary.NewHandler(summary.NewService(summary.NewRepository(db)))
	summaryHandler.RegisterRoutes(api)

	// ---- Priyapta: AI copilot routing (rate-limited, public) ----
	copilotClient := copilot.NewClient(cfg.AIServiceURL, cfg.AIServiceTimeoutSeconds)
	copilotHandler := copilot.NewHandler(copilot.NewService(copilotClient))
	copilotHandler.RegisterRoutes(api)

	// ---- Arzaka: survey ingestion + pipeline callbacks (service-key protected) ----
	surveyHandler := survey.NewHandler(survey.NewService(survey.NewRepository(db)))
	surveyHandler.RegisterRoutes(api)

	pipelineHandler := pipeline.NewHandler(pipeline.NewService(pipeline.NewRepository(db)))
	pipelineHandler.RegisterRoutes(api)

	// ---- Firaz: transparency (public — this is the point of the panel) ----
	transparencyHandler := transparency.NewHandler(transparency.NewService(transparency.NewRepository(db)))
	transparencyHandler.RegisterRoutes(api)

	// ---- Firaz: auth ----
	authHandler := auth.NewHandler(auth.NewService(auth.NewRepository(db), cfg.JWTSecret, cfg.JWTAccessTTLMins, cfg.JWTRefreshTTLHours))
	authHandler.RegisterRoutes(api)

	// ---- Firaz: premium (JWT + operator role required, see Use above) ----
	premiumHandler := premium.NewHandler(premium.NewService(premium.NewRepository(db)))
	premiumHandler.RegisterRoutes(api)

	// ---- Health check ----
	app.Get("/healthz", func(c *fiber.Ctx) error {
		return response.OKWithMessage(c, "Server Isi Stasiun berjalan normal", fiber.Map{
			"app":     "Isi Stasiun Backend",
			"version": "1.0.0",
		})
	})
}
