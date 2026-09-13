package pipeline

import (
	"context"

	"github.com/gofiber/fiber/v2"

	"github.com/list-pandora/isi-stasiun-backend/internal/response"
	"github.com/list-pandora/isi-stasiun-backend/internal/summary"
	"github.com/list-pandora/isi-stasiun-backend/internal/validate"
)

type Handler struct {
	svc pipelineService
}

type pipelineService interface {
	IngestStrukExtraction(ctx context.Context, cb StrukExtractionCallback) error
	IngestPropertiExtraction(ctx context.Context, cb PropertiExtractionCallback) error
	IngestGeraiClassification(ctx context.Context, cb GeraiClassificationCallback) error
	IngestMonteCarloResult(ctx context.Context, cb MonteCarloResultCallback) error
	IngestStationSummary(ctx context.Context, in summary.StationSummaryWrite) error
}

func NewHandler(svc pipelineService) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes wires the batch-pipeline callback endpoints. These sit
// behind RequireServiceKey — only the Python pipeline container calls them.
func (h *Handler) RegisterRoutes(r fiber.Router) {
	r.Post("/pipeline/extractions/struk", h.StrukExtraction)
	r.Post("/pipeline/extractions/properti", h.PropertiExtraction)
	r.Post("/pipeline/extractions/gerai", h.GeraiExtraction)
	r.Post("/pipeline/simulations/monte-carlo", h.MonteCarloResult)
	// Separate from the one above on purpose: that callback is per
	// (station, time_slot) and carries only P10/P90, while this one is a
	// whole-station rollup with P50, a simulated gap, peak and composition.
	// Folding the two into one payload would make half the fields meaningless
	// on every call.
	r.Post("/pipeline/simulations/station-summary", h.StationSummaryResult)
}

func (h *Handler) StrukExtraction(c *fiber.Ctx) error {
	var cb StrukExtractionCallback
	if err := c.BodyParser(&cb); err != nil {
		return response.BadRequest(c, "invalid payload")
	}
	if msg := validate.Struct(&cb); msg != "" {
		return response.BadRequest(c, msg)
	}
	// An ambiguous receipt is still recorded (it counts toward the coverage
	// rate) but carries no amount; a non-ambiguous one must have a real total.
	if !cb.IsAmbiguous && cb.FinalAmount <= 0 {
		return response.BadRequest(c, "final_amount must be positive unless is_ambiguous is true")
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
	if msg := validate.Struct(&cb); msg != "" {
		return response.BadRequest(c, msg)
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
	if msg := validate.Struct(&cb); msg != "" {
		return response.BadRequest(c, msg)
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
	if msg := validate.Struct(&cb); msg != "" {
		return response.BadRequest(c, msg)
	}
	if err := h.svc.IngestMonteCarloResult(c.Context(), cb); err != nil {
		return response.Internal(c, "failed to persist monte carlo result")
	}
	return response.OK(c, fiber.Map{"job_id": cb.JobID, "status": "ingested"})
}

func (h *Handler) StationSummaryResult(c *fiber.Ctx) error {
	var in summary.StationSummaryWrite
	if err := c.BodyParser(&in); err != nil {
		return response.BadRequest(c, "invalid payload")
	}
	if msg := validate.Struct(&in); msg != "" {
		return response.BadRequest(c, msg)
	}
	if err := in.Validate(); err != nil {
		return response.BadRequest(c, err.Error())
	}
	if err := h.svc.IngestStationSummary(c.Context(), in); err != nil {
		return response.Internal(c, "failed to persist station summary")
	}
	return response.OK(c, fiber.Map{"job_id": in.JobID, "status": "ingested"})
}
