package copilot

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

// RegisterRoutes wires POST /copilot/query. Mount behind
// middleware.RateLimit in router.go — this is the most abuse-prone endpoint
// since every call proxies to a paid LLM API.
func (h *Handler) RegisterRoutes(r fiber.Router) {
	r.Post("/copilot/query", h.Query)
}

func (h *Handler) Query(c *fiber.Ctx) error {
	var req QueryRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}
	if req.Query == "" {
		return response.BadRequest(c, "query is required")
	}

	resp, err := h.svc.Query(c.Context(), req)
	if err != nil {
		return response.Internal(c, "copilot query failed")
	}
	return response.OK(c, resp)
}
