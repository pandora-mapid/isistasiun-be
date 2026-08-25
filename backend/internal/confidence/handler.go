package confidence

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/list-pandora/isi-stasiun-backend/internal/response"
)

type Handler struct {
	svc confidenceService
}

type confidenceService interface {
	List(ctx context.Context, stationID string) ([]LayerEntry, error)
}

func NewHandler(svc confidenceService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r fiber.Router) {
	r.Get("/confidence-layer", h.List)
}

func (h *Handler) List(c *fiber.Ctx) error {
	stationID := c.Query("station_id", "")
	if stationID != "" {
		if _, err := uuid.Parse(stationID); err != nil {
			return response.BadRequest(c, "Format ID stasiun tidak valid (harus UUID)")
		}
	}

	data, err := h.svc.List(c.Context(), stationID)
	if err != nil {
		return response.Internal(c, "Gagal mengambil confidence layer")
	}
	return response.OKWithMessage(c, "Berhasil mengambil data confidence layer", data)
}
