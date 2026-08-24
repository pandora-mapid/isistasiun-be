package confidence

import (
	"github.com/gofiber/fiber/v2"

	"github.com/list-pandora/isi-stasiun-backend/internal/response"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r fiber.Router) {
	r.Get("/confidence-layer", h.List)
}

func (h *Handler) List(c *fiber.Ctx) error {
	stationID := c.Query("station_id", "")
	data, err := h.svc.List(c.Context(), stationID)
	if err != nil {
		return response.Internal(c, "failed to fetch confidence layer")
	}
	return response.OK(c, data)
}
