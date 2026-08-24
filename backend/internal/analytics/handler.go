package analytics

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
	r.Get("/analytics/spending-gap", h.SpendingGapAll)
	r.Get("/analytics/spending-gap/:station_id", h.SpendingGapByStation)
	r.Get("/analytics/category-gap", h.CategoryGap)
	r.Get("/analytics/rent-flow-index", h.RentFlowIndex)
	r.Get("/analytics/event-potential", h.EventPotential)
}

func (h *Handler) SpendingGapAll(c *fiber.Ctx) error {
	data, err := h.svc.SpendingGap(c.Context(), "")
	if err != nil {
		return response.Internal(c, "failed to fetch spending gap")
	}
	return response.OK(c, data)
}

func (h *Handler) SpendingGapByStation(c *fiber.Ctx) error {
	stationID := c.Params("station_id")
	data, err := h.svc.SpendingGap(c.Context(), stationID)
	if err != nil {
		return response.Internal(c, "failed to fetch spending gap")
	}
	return response.OK(c, data)
}

func (h *Handler) CategoryGap(c *fiber.Ctx) error {
	stationID := c.Query("station_id", "")
	data, err := h.svc.CategoryGap(c.Context(), stationID)
	if err != nil {
		return response.Internal(c, "failed to fetch category gap")
	}
	return response.OK(c, data)
}

func (h *Handler) RentFlowIndex(c *fiber.Ctx) error {
	stationID := c.Query("station_id", "")
	data, err := h.svc.RentFlowIndex(c.Context(), stationID)
	if err != nil {
		return response.Internal(c, "failed to fetch rent-flow index")
	}
	return response.OK(c, data)
}

func (h *Handler) EventPotential(c *fiber.Ctx) error {
	stationID := c.Query("station_id", "")
	data, err := h.svc.EventPotential(c.Context(), stationID)
	if err != nil {
		return response.Internal(c, "failed to fetch event potential")
	}
	return response.OK(c, data)
}
