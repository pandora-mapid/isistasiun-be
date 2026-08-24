package pipeline

import "context"

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) IngestStrukExtraction(ctx context.Context, cb StrukExtractionCallback) error {
	return s.repo.UpsertStrukExtraction(ctx, cb)
}

func (s *Service) IngestPropertiExtraction(ctx context.Context, cb PropertiExtractionCallback) error {
	return s.repo.UpsertPropertiExtraction(ctx, cb)
}

func (s *Service) IngestGeraiClassification(ctx context.Context, cb GeraiClassificationCallback) error {
	return s.repo.UpsertGeraiClassification(ctx, cb)
}

func (s *Service) IngestMonteCarloResult(ctx context.Context, cb MonteCarloResultCallback) error {
	return s.repo.UpsertMonteCarloResult(ctx, cb)
}
