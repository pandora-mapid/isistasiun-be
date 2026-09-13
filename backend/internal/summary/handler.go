package summary

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/list-pandora/isi-stasiun-backend/internal/response"
)

type Handler struct {
	svc summaryService
}

type summaryService interface {
	StationSummary(ctx context.Context, stationID string) ([]StationSummaryResponse, error)
}

func NewHandler(svc summaryService) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes wires the two station-summary reads. Public tier — the
// Compare View is free (../Context/04-VALUE-PROP-AND-MONETIZATION.md §3);
// only multi-simpul / cross-koridor is premium, and that is a different
// endpoint. router_test.go pins that these answer without a token.
func (h *Handler) RegisterRoutes(r fiber.Router) {
	r.Get("/analytics/station-summary", h.StationSummaryAll)
	r.Get("/analytics/station-summary/:station_id", h.StationSummaryByStation)
}

func (h *Handler) StationSummaryAll(c *fiber.Ctx) error {
	data, err := h.svc.StationSummary(c.Context(), "")
	if err != nil {
		return response.Internal(c, "Gagal mengambil ringkasan simpul")
	}
	return response.OKWithMessage(c, "Berhasil mengambil ringkasan simpul", data)
}

func (h *Handler) StationSummaryByStation(c *fiber.Ctx) error {
	stationID := c.Params("station_id")
	if _, err := uuid.Parse(stationID); err != nil {
		return response.BadRequest(c, "Format ID stasiun tidak valid (harus UUID)")
	}

	data, err := h.svc.StationSummary(c.Context(), stationID)
	if err != nil {
		return response.Internal(c, "Gagal mengambil ringkasan simpul")
	}
	return response.OKWithMessage(c, "Berhasil mengambil ringkasan simpul stasiun", data)
}
