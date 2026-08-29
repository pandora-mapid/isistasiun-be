package pipeline

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Each extraction callback writes two rows in one transaction: the raw
// extraction record and — when the pipeline supplied a redacted PhotoURL — the
// public transparency card linked to it. The link column is UNIQUE, so a
// re-run of the same batch job updates the same card rather than duplicating
// it (mirrors the job_id+source_ref idempotency on the extraction side).

func (r *Repository) UpsertStrukExtraction(ctx context.Context, cb StrukExtractionCallback) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var extractionID string
	err = tx.QueryRow(ctx, `
		INSERT INTO struk_extractions
			(job_id, source_ref, station_id, category, final_amount, payment_method,
			 transacted_at, confidence, is_ambiguous)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (job_id, source_ref) DO UPDATE SET
			final_amount = EXCLUDED.final_amount,
			confidence   = EXCLUDED.confidence,
			is_ambiguous = EXCLUDED.is_ambiguous
		RETURNING id
	`,
		cb.JobID, cb.SourceRef, cb.StationID, cb.Category, cb.FinalAmount, cb.PaymentMethod,
		cb.TransactedAt, cb.Confidence, cb.IsAmbiguous,
	).Scan(&extractionID)
	if err != nil {
		return err
	}

	if cb.PhotoURL != "" {
		reading := jsonText(map[string]any{
			"category":       cb.Category,
			"final_amount":   cb.FinalAmount,
			"payment_method": cb.PaymentMethod,
			"transacted_at":  cb.TransactedAt,
		})
		if _, err = tx.Exec(ctx, `
			INSERT INTO transparency_struk
				(station_id, photo_url, ai_reading, confidence, is_ambiguous, struk_extraction_id)
			VALUES ($1,$2,$3,$4,$5,$6)
			ON CONFLICT (struk_extraction_id) DO UPDATE SET
				photo_url    = EXCLUDED.photo_url,
				ai_reading   = EXCLUDED.ai_reading,
				confidence   = EXCLUDED.confidence,
				is_ambiguous = EXCLUDED.is_ambiguous
		`, cb.StationID, cb.PhotoURL, reading, cb.Confidence, cb.IsAmbiguous, extractionID); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *Repository) UpsertPropertiExtraction(ctx context.Context, cb PropertiExtractionCallback) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var extractionID string
	err = tx.QueryRow(ctx, `
		INSERT INTO properti_extractions
			(job_id, source_ref, station_id, plot_id, offered_rent, area_sqm, confidence)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT (job_id, source_ref) DO UPDATE SET
			offered_rent = EXCLUDED.offered_rent,
			area_sqm     = EXCLUDED.area_sqm,
			confidence   = EXCLUDED.confidence
		RETURNING id
	`,
		cb.JobID, cb.SourceRef, cb.StationID, cb.PlotID, cb.OfferedRent, cb.AreaSqm, cb.Confidence,
	).Scan(&extractionID)
	if err != nil {
		return err
	}

	if cb.PhotoURL != "" {
		reading := jsonText(map[string]any{
			"plot_id":      cb.PlotID,
			"offered_rent": cb.OfferedRent,
			"area_sqm":     cb.AreaSqm,
		})
		if _, err = tx.Exec(ctx, `
			INSERT INTO transparency_properti
				(station_id, photo_url, ai_reading, confidence, properti_extraction_id)
			VALUES ($1,$2,$3,$4,$5)
			ON CONFLICT (properti_extraction_id) DO UPDATE SET
				photo_url  = EXCLUDED.photo_url,
				ai_reading = EXCLUDED.ai_reading,
				confidence = EXCLUDED.confidence
		`, cb.StationID, cb.PhotoURL, reading, cb.Confidence, extractionID); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *Repository) UpsertGeraiClassification(ctx context.Context, cb GeraiClassificationCallback) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var classificationID string
	err = tx.QueryRow(ctx, `
		INSERT INTO gerai_classifications
			(job_id, source_ref, station_id, gerai_id, category, visibility, confidence)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT (job_id, source_ref) DO UPDATE SET
			category   = EXCLUDED.category,
			visibility = EXCLUDED.visibility,
			confidence = EXCLUDED.confidence
		RETURNING id
	`,
		cb.JobID, cb.SourceRef, cb.StationID, cb.GeraiID, cb.Category, cb.Visibility, cb.Confidence,
	).Scan(&classificationID)
	if err != nil {
		return err
	}

	if cb.PhotoURL != "" {
		reading := jsonText(map[string]any{
			"gerai_id":   cb.GeraiID,
			"category":   cb.Category,
			"visibility": cb.Visibility,
		})
		if _, err = tx.Exec(ctx, `
			INSERT INTO transparency_gerai
				(station_id, photo_url, ai_reading, confidence, gerai_classification_id)
			VALUES ($1,$2,$3,$4,$5)
			ON CONFLICT (gerai_classification_id) DO UPDATE SET
				photo_url  = EXCLUDED.photo_url,
				ai_reading = EXCLUDED.ai_reading,
				confidence = EXCLUDED.confidence
		`, cb.StationID, cb.PhotoURL, reading, cb.Confidence, classificationID); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *Repository) UpsertMonteCarloResult(ctx context.Context, cb MonteCarloResultCallback) error {
	query := `
		INSERT INTO spending_gap_estimates
			(job_id, station_id, time_slot, potential_low_p10, potential_high_p90,
			 captured_low_p10, captured_high_p90, gap_low_p10, gap_high_p90, iterations, computed_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10, now())
		ON CONFLICT (station_id, time_slot) DO UPDATE SET
			job_id             = EXCLUDED.job_id,
			potential_low_p10  = EXCLUDED.potential_low_p10,
			potential_high_p90 = EXCLUDED.potential_high_p90,
			captured_low_p10   = EXCLUDED.captured_low_p10,
			captured_high_p90  = EXCLUDED.captured_high_p90,
			gap_low_p10        = EXCLUDED.gap_low_p10,
			gap_high_p90       = EXCLUDED.gap_high_p90,
			iterations         = EXCLUDED.iterations,
			computed_at        = now()
	`
	gapLow := cb.PotentialLowP10 - cb.CapturedHighP90  // conservative lower bound
	gapHigh := cb.PotentialHighP90 - cb.CapturedLowP10 // conservative upper bound

	_, err := r.db.Exec(ctx, query,
		cb.JobID, cb.StationID, cb.TimeSlot,
		cb.PotentialLowP10, cb.PotentialHighP90,
		cb.CapturedLowP10, cb.CapturedHighP90,
		gapLow, gapHigh, cb.Iterations,
	)
	return err
}

// jsonText renders the extracted fields as the JSON-as-text snapshot stored in
// transparency_*.ai_reading. Marshal of a map[string]any with scalar values
// never fails, so the error is intentionally dropped.
func jsonText(m map[string]any) string {
	b, _ := json.Marshal(m)
	return string(b)
}
