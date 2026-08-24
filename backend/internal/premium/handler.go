package premium

import (
	"github.com/gofiber/fiber/v2"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/list-pandora/isi-stasiun-backend/internal/response"
)

type Handler struct {
	db *pgxpool.Pool
}

func NewHandler(db *pgxpool.Pool) *Handler {
	return &Handler{db: db}
}

// RegisterRoutes wires premium routes. Mount these behind
// middleware.RequireAuth + middleware.RequireRole("operator") in router.go —
// this is the paid tier per section 4.1 (operator/pengelola kawasan).
func (h *Handler) RegisterRoutes(r fiber.Router) {
	r.Get("/premium/deep-analysis/:station_id", h.DeepAnalysis)
}

func (h *Handler) DeepAnalysis(c *fiber.Ctx) error {
	stationID := c.Params("station_id")

	// TODO(Firaz): join spending_gap_estimates + rent_flow_index + category_gap
	// at per-entrance/per-plot granularity, beyond what the free analytics
	// endpoints expose. Placeholder shape below so frontend can integrate now.
	return response.OK(c, fiber.Map{
		"station_id": stationID,
		"note":       "deep-analysis not yet implemented",
	})
}
