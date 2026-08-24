package analytics

import "context"

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) SpendingGap(ctx context.Context, stationID string) ([]SpendingGapResponse, error) {
	return s.repo.SpendingGap(ctx, stationID)
}

func (s *Service) CategoryGap(ctx context.Context, stationID string) ([]CategoryGapResponse, error) {
	return s.repo.CategoryGap(ctx, stationID)
}

func (s *Service) RentFlowIndex(ctx context.Context, stationID string) ([]RentFlowIndexResponse, error) {
	return s.repo.RentFlowIndex(ctx, stationID)
}

func (s *Service) EventPotential(ctx context.Context, stationID string) ([]EventPotentialResponse, error) {
	return s.repo.EventPotential(ctx, stationID)
}
