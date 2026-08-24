package pipeline

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

// RegisterRoutes wires the batch-pipeline callback endpoints. These sit
// behind RequireServiceKey — only the Python pipeline container calls them.
func (h *Handler) RegisterRoutes(r fiber.Router) {
	r.Post("/pipeline/extractions/struk", h.StrukExtraction)
	r.Post("/pipeline/extractions/properti", h.PropertiExtraction)
	r.Post("/pipeline/extractions/gerai", h.GeraiExtraction)
	r.Post("/pipeline/simulations/monte-carlo", h.MonteCarloResult)
}

func (h *Handler) StrukExtraction(c *fiber.Ctx) error {
	var cb StrukExtractionCallback
	if err := c.BodyParser(&cb); err != nil {
		return response.BadRequest(c, "invalid payload")
	}
	if err := h.svc.IngestStrukExtraction(c.Context(), cb); err != nil {
		return response.Internal(c, "failed to persist struk extraction")
	}
	return response.OK(c, fiber.Map{"job_id": cb.JobID, "status": "ingested"})
}

func (h *Handler) PropertiExtraction(c *fiber.Ctx) error {
	var cb PropertiExtractionCallback
	if err := c.BodyParser(&cb); err != nil {
		return response.BadRequest(c, "invalid payload")
	}
	if err := h.svc.IngestPropertiExtraction(c.Context(), cb); err != nil {
		return response.Internal(c, "failed to persist properti extraction")
	}
	return response.OK(c, fiber.Map{"job_id": cb.JobID, "status": "ingested"})
}

func (h *Handler) GeraiExtraction(c *fiber.Ctx) error {
	var cb GeraiClassificationCallback
	if err := c.BodyParser(&cb); err != nil {
		return response.BadRequest(c, "invalid payload")
	}
	if err := h.svc.IngestGeraiClassification(c.Context(), cb); err != nil {
		return response.Internal(c, "failed to persist gerai classification")
	}
	return response.OK(c, fiber.Map{"job_id": cb.JobID, "status": "ingested"})
}

func (h *Handler) MonteCarloResult(c *fiber.Ctx) error {
	var cb MonteCarloResultCallback
	if err := c.BodyParser(&cb); err != nil {
		return response.BadRequest(c, "invalid payload")
	}
	if err := h.svc.IngestMonteCarloResult(c.Context(), cb); err != nil {
		return response.Internal(c, "failed to persist monte carlo result")
	}
	return response.OK(c, fiber.Map{"job_id": cb.JobID, "status": "ingested"})
}
