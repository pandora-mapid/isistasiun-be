package pipeline

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) UpsertStrukExtraction(ctx context.Context, cb StrukExtractionCallback) error {
	query := `
		INSERT INTO struk_extractions
			(job_id, source_ref, station_id, category, final_amount, payment_method,
			 transacted_at, confidence, is_ambiguous)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (job_id, source_ref) DO UPDATE SET
			final_amount = EXCLUDED.final_amount,
			confidence   = EXCLUDED.confidence,
			is_ambiguous = EXCLUDED.is_ambiguous
	`
	_, err := r.db.Exec(ctx, query,
		cb.JobID, cb.SourceRef, cb.StationID, cb.Category, cb.FinalAmount, cb.PaymentMethod,
		cb.TransactedAt, cb.Confidence, cb.IsAmbiguous,
	)
	return err
}

func (r *Repository) UpsertPropertiExtraction(ctx context.Context, cb PropertiExtractionCallback) error {
	query := `
		INSERT INTO properti_extractions
			(job_id, source_ref, station_id, plot_id, offered_rent, area_sqm, confidence)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT (job_id, source_ref) DO UPDATE SET
			offered_rent = EXCLUDED.offered_rent,
			area_sqm     = EXCLUDED.area_sqm,
			confidence   = EXCLUDED.confidence
	`
	_, err := r.db.Exec(ctx, query,
		cb.JobID, cb.SourceRef, cb.StationID, cb.PlotID, cb.OfferedRent, cb.AreaSqm, cb.Confidence,
	)
	return err
}

func (r *Repository) UpsertGeraiClassification(ctx context.Context, cb GeraiClassificationCallback) error {
	query := `
		INSERT INTO gerai_classifications
			(job_id, source_ref, station_id, gerai_id, category, visibility, confidence)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT (job_id, source_ref) DO UPDATE SET
			category   = EXCLUDED.category,
			visibility = EXCLUDED.visibility,
			confidence = EXCLUDED.confidence
	`
	_, err := r.db.Exec(ctx, query,
		cb.JobID, cb.SourceRef, cb.StationID, cb.GeraiID, cb.Category, cb.Visibility, cb.Confidence,
	)
	return err
}

func (r *Repository) UpsertMonteCarloResult(ctx context.Context, cb MonteCarloResultCallback) error {
	query := `
		INSERT INTO spending_gap_estimates
			(job_id, station_id, time_slot, potential_low_p10, potential_high_p90,
			 captured_low_p10, captured_high_p90, gap_low_p10, gap_high_p90, computed_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9, now())
		ON CONFLICT (station_id, time_slot) DO UPDATE SET
			job_id             = EXCLUDED.job_id,
			potential_low_p10  = EXCLUDED.potential_low_p10,
			potential_high_p90 = EXCLUDED.potential_high_p90,
			captured_low_p10   = EXCLUDED.captured_low_p10,
			captured_high_p90  = EXCLUDED.captured_high_p90,
			gap_low_p10        = EXCLUDED.gap_low_p10,
			gap_high_p90       = EXCLUDED.gap_high_p90,
			computed_at        = now()
	`
	gapLow := cb.PotentialLowP10 - cb.CapturedHighP90   // conservative lower bound
	gapHigh := cb.PotentialHighP90 - cb.CapturedLowP10  // conservative upper bound

	_, err := r.db.Exec(ctx, query,
		cb.JobID, cb.StationID, cb.TimeSlot,
		cb.PotentialLowP10, cb.PotentialHighP90,
		cb.CapturedLowP10, cb.CapturedHighP90,
		gapLow, gapHigh,
	)
	return err
}
