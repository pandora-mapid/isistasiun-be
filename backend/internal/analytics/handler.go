package analytics

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/list-pandora/isi-stasiun-backend/internal/response"
)

type Handler struct {
	svc analyticsService
}

type analyticsService interface {
	SpendingGap(ctx context.Context, stationID string) ([]SpendingGapResponse, error)
	CategoryGap(ctx context.Context, stationID string) ([]CategoryGapResponse, error)
	RentFlowIndex(ctx context.Context, stationID string) ([]RentFlowIndexResponse, error)
	EventPotential(ctx context.Context, stationID string) ([]EventPotentialResponse, error)
}

func NewHandler(svc analyticsService) *Handler {
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
		return response.Internal(c, "Gagal mengambil data spending gap")
	}
	return response.OKWithMessage(c, "Berhasil mengambil data spending gap", data)
}

func (h *Handler) SpendingGapByStation(c *fiber.Ctx) error {
	stationID := c.Params("station_id")
	if _, err := uuid.Parse(stationID); err != nil {
		return response.BadRequest(c, "Format ID stasiun tidak valid (harus UUID)")
	}

	data, err := h.svc.SpendingGap(c.Context(), stationID)
	if err != nil {
		return response.Internal(c, "Gagal mengambil data spending gap")
	}
	return response.OKWithMessage(c, "Berhasil mengambil data spending gap stasiun", data)
}

func (h *Handler) CategoryGap(c *fiber.Ctx) error {
	stationID := c.Query("station_id", "")
	if stationID != "" {
		if _, err := uuid.Parse(stationID); err != nil {
			return response.BadRequest(c, "Format ID stasiun tidak valid (harus UUID)")
		}
	}

	data, err := h.svc.CategoryGap(c.Context(), stationID)
	if err != nil {
		return response.Internal(c, "Gagal mengambil data category gap")
	}
	return response.OKWithMessage(c, "Berhasil mengambil data category gap", data)
}

func (h *Handler) RentFlowIndex(c *fiber.Ctx) error {
	stationID := c.Query("station_id", "")
	if stationID != "" {
		if _, err := uuid.Parse(stationID); err != nil {
			return response.BadRequest(c, "Format ID stasiun tidak valid (harus UUID)")
		}
	}

	data, err := h.svc.RentFlowIndex(c.Context(), stationID)
	if err != nil {
		return response.Internal(c, "Gagal mengambil data rent-flow index")
	}
	return response.OKWithMessage(c, "Berhasil mengambil data rent-flow index", data)
}

func (h *Handler) EventPotential(c *fiber.Ctx) error {
	stationID := c.Query("station_id", "")
	if stationID != "" {
		if _, err := uuid.Parse(stationID); err != nil {
			return response.BadRequest(c, "Format ID stasiun tidak valid (harus UUID)")
		}
	}

	data, err := h.svc.EventPotential(c.Context(), stationID)
	if err != nil {
		return response.Internal(c, "Gagal mengambil data event potential")
	}
	return response.OKWithMessage(c, "Berhasil mengambil data potensi event", data)
}
