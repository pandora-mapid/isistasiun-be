package summary

import "context"

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) StationSummary(ctx context.Context, stationID string) ([]StationSummaryResponse, error) {
	return s.repo.StationSummary(ctx, stationID)
}
