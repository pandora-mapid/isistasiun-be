package survey

import (
	"context"
	"fmt"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) SubmitFlowObservation(ctx context.Context, req CreateFlowObservationRequest, idempotencyKey string) (string, error) {
	observedAt, err := parseTime(req.ObservedAt)
	if err != nil {
		return "", fmt.Errorf("invalid observed_at: %w", err)
	}

	o := FlowObservation{
		IdempotencyKey:  idempotencyKey,
		StationID:       req.StationID,
		EntranceID:      req.EntranceID,
		TimeSlot:        req.TimeSlot,
		ObservedAt:      observedAt,
		BlockNumber:     req.BlockNumber,
		PedestrianCount: req.PedestrianCount,
		Direction:       req.Direction,
		WeatherNote:     req.WeatherNote,
		SurveyorID:      req.SurveyorID,
	}
	return s.repo.InsertFlowObservation(ctx, o)
}

func (s *Service) SubmitEntryConversion(ctx context.Context, req CreateEntryConversionRequest, idempotencyKey string) (string, error) {
	observedAt, err := parseTime(req.ObservedAt)
	if err != nil {
		return "", fmt.Errorf("invalid observed_at: %w", err)
	}

	// Guard against double-counting per methodology 3.3: entered_count must not
	// exceed passers_by, and completed_purchase_count must not exceed entered_count.
	if req.EnteredCount > req.PassersBy {
		return "", fmt.Errorf("entered_count cannot exceed passers_by")
	}
	if req.CompletedPurchaseCount > req.EnteredCount {
		return "", fmt.Errorf("completed_purchase_count cannot exceed entered_count")
	}

	o := EntryConversionObservation{
		IdempotencyKey:         idempotencyKey,
		StationID:              req.StationID,
		GeraiID:                req.GeraiID,
		Category:               req.Category,
		TimeSlot:               req.TimeSlot,
		ObservedAt:             observedAt,
		BlockNumber:            req.BlockNumber,
		PassersBy:              req.PassersBy,
		EnteredCount:           req.EnteredCount,
		CompletedPurchaseCount: req.CompletedPurchaseCount,
		SurveyorID:             req.SurveyorID,
	}
	return s.repo.InsertEntryConversion(ctx, o)
}

func (s *Service) ListFlowObservations(ctx context.Context, stationID, entranceID, timeSlot string) ([]FlowObservation, error) {
	return s.repo.ListFlowObservations(ctx, stationID, entranceID, timeSlot)
}
