package rental

import "context"

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, params ListParams) ([]AssetResponse, error) {
	assets, err := s.repo.List(ctx, params)
	if err != nil {
		return nil, err
	}
	out := make([]AssetResponse, 0, len(assets))
	for _, asset := range assets {
		out = append(out, toResponse(asset))
	}
	return out, nil
}
