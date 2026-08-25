package station

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

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
		return response.BadRequest(c, "Format query parameter tidak valid")
	}

	stations, err := h.svc.ListStations(c.Context(), q.AreaType, q.Operator)
	if err != nil {
		return response.Internal(c, "Gagal mengambil daftar stasiun")
	}
	return response.OKWithMessage(c, "Berhasil mengambil daftar stasiun", stations)
}

func (h *Handler) GetStation(c *fiber.Ctx) error {
	id := c.Params("id")
	if _, err := uuid.Parse(id); err != nil {
		return response.BadRequest(c, "Format ID stasiun tidak valid (harus UUID)")
	}

	st, err := h.svc.GetStation(c.Context(), id)
	if errors.Is(err, ErrNotFound) {
		return response.NotFound(c, "Stasiun tidak ditemukan")
	}
	if err != nil {
		return response.Internal(c, "Gagal mengambil detail stasiun")
	}
	return response.OKWithMessage(c, "Berhasil mengambil detail stasiun", st)
}

func (h *Handler) ListEntrances(c *fiber.Ctx) error {
	id := c.Params("id")
	if _, err := uuid.Parse(id); err != nil {
		return response.BadRequest(c, "Format ID stasiun tidak valid (harus UUID)")
	}

	entrances, err := h.svc.ListEntrances(c.Context(), id)
	if errors.Is(err, ErrNotFound) {
		return response.NotFound(c, "Stasiun tidak ditemukan")
	}
	if err != nil {
		return response.Internal(c, "Gagal mengambil daftar pintu stasiun")
	}
	return response.OKWithMessage(c, "Berhasil mengambil daftar pintu stasiun", entrances)
}
