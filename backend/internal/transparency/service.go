package transparency

import "context"

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetStruk(ctx context.Context, id string) (*Record, error)    { return s.repo.GetStruk(ctx, id) }
func (s *Service) GetGerai(ctx context.Context, id string) (*Record, error)    { return s.repo.GetGerai(ctx, id) }
func (s *Service) GetProperti(ctx context.Context, id string) (*Record, error) { return s.repo.GetProperti(ctx, id) }
func (s *Service) ListByStation(ctx context.Context, stationID string) ([]Record, error) {
	return s.repo.ListByStation(ctx, stationID)
}
