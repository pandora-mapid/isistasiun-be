package router

import (
	"time"

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
	"github.com/list-pandora/isi-stasiun-backend/internal/rental"
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
	api.Use("/auth/login", middleware.RateLimit(cfg.AuthRateLimitRPM))
	api.Use("/auth/register", middleware.RateLimit(cfg.AuthRateLimitRPM))
	// /auth/upgrade is the one /auth/* route that needs an existing session —
	// it flips the caller's own account, so it must know who's calling.
	api.Use("/auth/upgrade", middleware.RequireAuth(cfg.JWTSecret))

	// Base map + basic analysis moved from public to "any logged-in account"
	// tier (2026-09-13 product decision — supersedes the old free-tier-forever
	// policy; see the vault ADR). RequireRole is deliberately absent here: a
	// bare "user" is enough, same as premium/operator/admin. /premium stays a
	// stricter gate below, and /transparency stays open on purpose — it is
	// the evidence panel judges are meant to see without an account.
	//
	// /confidence-layer, /analytics/rental-assets and /analytics/rent-flow-index
	// are deliberately NOT in this list, even though they're "basic analysis"
	// too: Beranda — the one screen that must stay public, it's the
	// login/register funnel — shares usePetaData() with Peta, and that hook
	// resolves every loader through a single Promise.all. Gating any one of
	// these three (the only loaders currently live against the real API
	// rather than mock/) made the ENTIRE public landing page's map fail for
	// anonymous visitors, not just the one field. Revisit once the frontend
	// stops sharing that hook between a public and a gated screen.
	basicAuth := middleware.RequireAuth(cfg.JWTSecret)
	api.Use("/stations", basicAuth)
	api.Use("/analytics/spending-gap", basicAuth)
	api.Use("/analytics/category-gap", basicAuth)
	api.Use("/analytics/event-potential", basicAuth)
	api.Use("/analytics/station-summary", basicAuth)

	api.Use("/premium",
		middleware.RequireAuth(cfg.JWTSecret),
		middleware.RequireRole(string(auth.RolePremium), string(auth.RoleOperator), string(auth.RoleAdmin)),
	)
	api.Use("/admin",
		middleware.RequireAuth(cfg.JWTSecret),
		middleware.RequireRole(string(auth.RoleAdmin)),
	)

	// ---- Priyapta: map/analytics/confidence (now behind basicAuth above) ----
	stationHandler := station.NewHandler(station.NewService(station.NewRepository(db)))
	stationHandler.RegisterRoutes(api)

	analyticsHandler := analytics.NewHandler(analytics.NewService(analytics.NewRepository(db)))
	analyticsHandler.RegisterRoutes(api)

	rentalHandler := rental.NewHandler(rental.NewService(rental.NewRepository(db)))
	rentalHandler.RegisterRoutes(api)

	confidenceHandler := confidence.NewHandler(confidence.NewService(confidence.NewRepository(db)))
	confidenceHandler.RegisterRoutes(api)

	// ---- Arzaka: station-level rollup for the Compare View (behind basicAuth, same as /analytics) ----
	// TTL-cached: rows only change when the batch pipeline re-runs, and the
	// FE reads the same two stations repeatedly (map dialog + Insight +
	// Beranda), so this is a pure win with no numbers changed.
	api.Use("/analytics/station-summary", middleware.ResponseCache(cfg.SummaryCacheTTLSeconds))
	summaryRepo := summary.NewRepository(db)
	summaryHandler := summary.NewHandler(summary.NewService(summaryRepo))
	summaryHandler.RegisterRoutes(api)

	// ---- Priyapta: AI copilot routing (rate-limited, public) ----
	copilotClient := copilot.NewClient(cfg.AIServiceURL, cfg.AIServiceTimeoutSeconds)
	copilotHandler := copilot.NewHandler(copilot.NewService(copilotClient))
	copilotHandler.RegisterRoutes(api)

	// ---- Arzaka: survey ingestion + pipeline callbacks (service-key protected) ----
	surveyHandler := survey.NewHandler(survey.NewService(survey.NewRepository(db)))
	surveyHandler.RegisterRoutes(api)

	// The pipeline callbacks own the write path into station_summary, which
	// the read handler above only ever reads — hence the shared repository.
	pipelineHandler := pipeline.NewHandler(pipeline.NewService(pipeline.NewRepository(db), summaryRepo))
	pipelineHandler.RegisterRoutes(api)

	// ---- Firaz: transparency (public — this is the point of the panel) ----
	transparencyHandler := transparency.NewHandler(transparency.NewService(transparency.NewRepository(db)))
	transparencyHandler.RegisterRoutes(api)

	// ---- Firaz: auth ----
	authHandler := auth.NewHandler(
		auth.NewService(auth.NewRepository(db), cfg.JWTSecret, cfg.JWTAccessTTLMins, cfg.JWTRefreshTTLHours),
		auth.CookieConfig{
			Secure: cfg.IsProduction(),
			Domain: cfg.AuthCookieDomain,
			TTL:    time.Duration(cfg.JWTRefreshTTLHours) * time.Hour,
		},
	)
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
