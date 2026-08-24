package station

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/list-pandora/isi-stasiun-backend/internal/response"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes wires GET /stations, /stations/:id, /stations/:id/entrances.
func (h *Handler) RegisterRoutes(r fiber.Router) {
	r.Get("/stations", h.ListStations)
	r.Get("/stations/:id", h.GetStation)
	r.Get("/stations/:id/entrances", h.ListEntrances)
}

func (h *Handler) ListStations(c *fiber.Ctx) error {
	var q ListStationsQuery
	if err := c.QueryParser(&q); err != nil {
		return response.BadRequest(c, "invalid query params")
	}

	stations, err := h.svc.ListStations(c.Context(), q.AreaType, q.Operator)
	if err != nil {
		return response.Internal(c, "failed to list stations")
	}
	return response.OK(c, stations)
}

func (h *Handler) GetStation(c *fiber.Ctx) error {
	id := c.Params("id")
	st, err := h.svc.GetStation(c.Context(), id)
	if errors.Is(err, ErrNotFound) {
		return response.NotFound(c, "station not found")
	}
	if err != nil {
		return response.Internal(c, "failed to get station")
	}
	return response.OK(c, st)
}

func (h *Handler) ListEntrances(c *fiber.Ctx) error {
	id := c.Params("id")
	entrances, err := h.svc.ListEntrances(c.Context(), id)
	if errors.Is(err, ErrNotFound) {
		return response.NotFound(c, "station not found")
	}
	if err != nil {
		return response.Internal(c, "failed to list entrances")
	}
	return response.OK(c, entrances)
}
