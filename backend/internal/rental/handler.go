package rental

import (
	"context"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/list-pandora/isi-stasiun-backend/internal/response"
)

type rentalService interface {
	List(context.Context, ListParams) ([]AssetResponse, error)
}

type Handler struct {
	svc rentalService
}

func NewHandler(svc rentalService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r fiber.Router) {
	r.Get("/analytics/rental-assets", h.List)
}

func (h *Handler) List(c *fiber.Ctx) error {
	params := ListParams{
		StationID:   c.Query("station_id", ""),
		StationCode: strings.TrimSpace(c.Query("station_code", "")),
		Status:      strings.TrimSpace(c.Query("status", "")),
	}
	if params.StationID != "" {
		if _, err := uuid.Parse(params.StationID); err != nil {
			return response.BadRequest(c, "Format ID stasiun tidak valid (harus UUID)")
		}
	}
	if params.Status != "" && !validStatus(params.Status) {
		return response.BadRequest(c, "Status aset sewa tidak valid")
	}

	data, err := h.svc.List(c.Context(), params)
	if err != nil {
		return response.Internal(c, "Gagal mengambil data aset sewa")
	}
	return response.OKWithMessage(c, "Berhasil mengambil data aset sewa", data)
}

func validStatus(status string) bool {
	switch status {
	case "occupied", "available", "needs_verification":
		return true
	default:
		return false
	}
}
