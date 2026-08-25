package copilot

import (
	"strings"

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

// RegisterRoutes wires POST /copilot/query. Mount behind
// middleware.RateLimit in router.go — this is the most abuse-prone endpoint
// since every call proxies to a paid LLM API.
func (h *Handler) RegisterRoutes(r fiber.Router) {
	r.Post("/copilot/query", h.Query)
}

func (h *Handler) Query(c *fiber.Ctx) error {
	var req QueryRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Format body request tidak valid")
	}

	req.Query = strings.TrimSpace(req.Query)
	if req.Query == "" {
		return response.BadRequest(c, "Query wajib diisi")
	}
	if len(req.Query) > 500 {
		return response.BadRequest(c, "Query terlalu panjang (maksimal 500 karakter)")
	}
	if req.StationID != "" {
		if _, err := uuid.Parse(req.StationID); err != nil {
			return response.BadRequest(c, "Format ID stasiun tidak valid (harus UUID)")
		}
	}

	resp, err := h.svc.Query(c.Context(), req)
	if err != nil {
		return response.Internal(c, "Gagal memproses query copilot")
	}
	return response.OKWithMessage(c, "Berhasil memproses query copilot", resp)
}
