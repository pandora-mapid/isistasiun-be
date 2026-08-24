package survey

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

// RegisterRoutes wires the survey ingestion endpoints. These sit behind the
// RequireServiceKey middleware (see router.go) since submissions come from
// MAPID Apps survey activities, not end-user sessions.
func (h *Handler) RegisterRoutes(r fiber.Router) {
	r.Post("/survey/flow-observations", h.SubmitFlowObservation)
	r.Get("/survey/flow-observations", h.ListFlowObservations)
	r.Post("/survey/entry-conversion-observations", h.SubmitEntryConversion)
}

func (h *Handler) SubmitFlowObservation(c *fiber.Ctx) error {
	var req CreateFlowObservationRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}
	if req.StationID == "" || req.EntranceID == "" || req.SurveyorID == "" {
		return response.BadRequest(c, "station_id, entrance_id, surveyor_id are required")
	}

	id, err := h.svc.SubmitFlowObservation(c.Context(), req)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}
	return response.Created(c, fiber.Map{"id": id})
}

func (h *Handler) SubmitEntryConversion(c *fiber.Ctx) error {
	var req CreateEntryConversionRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}
	if req.StationID == "" || req.GeraiID == "" || req.SurveyorID == "" {
		return response.BadRequest(c, "station_id, gerai_id, surveyor_id are required")
	}

	id, err := h.svc.SubmitEntryConversion(c.Context(), req)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}
	return response.Created(c, fiber.Map{"id": id})
}

func (h *Handler) ListFlowObservations(c *fiber.Ctx) error {
	var q ListFlowObservationsQuery
	if err := c.QueryParser(&q); err != nil {
		return response.BadRequest(c, "invalid query params")
	}

	data, err := h.svc.ListFlowObservations(c.Context(), q.StationID, q.EntranceID, q.TimeSlot)
	if err != nil {
		return response.Internal(c, "failed to list flow observations")
	}
	return response.OK(c, data)
}
