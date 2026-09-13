package pipeline

import (
	"context"

	"github.com/list-pandora/isi-stasiun-backend/internal/summary"
)

// summaryWriter is the one write the station-summary owner exposes to this
// package. Kept as an interface so the callback stays testable without a DB
// and so internal/summary never has to know about pipeline types.
type summaryWriter interface {
	UpsertStationSummary(ctx context.Context, in summary.StationSummaryWrite) error
}

type Service struct {
	repo    *Repository
	summary summaryWriter
}

func NewService(repo *Repository, summaryRepo summaryWriter) *Service {
	return &Service{repo: repo, summary: summaryRepo}
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

// IngestStationSummary persists the station-scoped rollup. This is the write
// path station_summary never had: the table and the read handler shipped, but
// nothing filled them, so GET /analytics/station-summary could only ever
// answer from the demo seed.
func (s *Service) IngestStationSummary(ctx context.Context, in summary.StationSummaryWrite) error {
	return s.summary.UpsertStationSummary(ctx, in)
}
