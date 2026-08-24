package transparency

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

func (h *Handler) RegisterRoutes(r fiber.Router) {
	r.Get("/transparency/struk/:id", h.GetStruk)
	r.Get("/transparency/gerai/:id", h.GetGerai)
	r.Get("/transparency/properti/:id", h.GetProperti)
	r.Get("/transparency/station/:id/records", h.ListByStation)
}

func (h *Handler) GetStruk(c *fiber.Ctx) error {
	rec, err := h.svc.GetStruk(c.Context(), c.Params("id"))
	return respondRecord(c, rec, err)
}

func (h *Handler) GetGerai(c *fiber.Ctx) error {
	rec, err := h.svc.GetGerai(c.Context(), c.Params("id"))
	return respondRecord(c, rec, err)
}

func (h *Handler) GetProperti(c *fiber.Ctx) error {
	rec, err := h.svc.GetProperti(c.Context(), c.Params("id"))
	return respondRecord(c, rec, err)
}

func (h *Handler) ListByStation(c *fiber.Ctx) error {
	data, err := h.svc.ListByStation(c.Context(), c.Params("id"))
	if err != nil {
		return response.Internal(c, "failed to list transparency records")
	}
	return response.OK(c, data)
}

func respondRecord(c *fiber.Ctx, rec *Record, err error) error {
	if errors.Is(err, ErrNotFound) {
		return response.NotFound(c, "transparency record not found")
	}
	if err != nil {
		return response.Internal(c, "failed to fetch transparency record")
	}
	return response.OK(c, rec)
}
