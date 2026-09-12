package summary

import (
	"context"
	"fmt"
)

// The write side of station_summary. The package doc calls the read path
// read-only, and it stays that way: nothing here is reachable from the public
// /analytics/station-summary handler. The only caller is the batch-pipeline
// callback (internal/pipeline), which sits behind X-Service-Key.
//
// The wire shape deliberately mirrors StationSummaryResponse field for field.
// A station rollup that round-trips through the API unchanged is much easier
// to eyeball against what the frontend renders — and it means the pipeline
// cannot quietly publish a number the read path has no place to show.

// StationSummaryWrite is one station-scoped Monte Carlo rollup as the pipeline
// posts it. Unlike the per-(station, time_slot) MonteCarloResultCallback this
// is a whole-station, whole-day figure, which is why it carries P50, a gap
// simulated in its own right, and the composition/peak context.
type StationSummaryWrite struct {
	JobID     string `json:"job_id" validate:"required"`
	StationID string `json:"station_id" validate:"required,uuid"`
	DayType   string `json:"day_type" validate:"required,oneof=weekday weekend"`

	// Iterations is validated but not stored — station_summary has no column
	// for it. It is here so a truncated run cannot be published as a full one.
	Iterations int `json:"iterations" validate:"required,min=10000"`

	Potensi    MoneyRange `json:"potensi"`
	Tertangkap MoneyRange `json:"tertangkap"`
	// Gap is simulated per iteration (potential_i - captured_i), not derived
	// by subtracting the two sets of percentiles — subtracting percentiles
	// would overstate the spread and break the P10 <= P50 <= P90 ordering.
	Gap MoneyRange `json:"gap"`

	// CaptureRate and Confidence are nullable: a station with nothing
	// estimable must report "unknown", never 0.
	CaptureRate  *float64        `json:"capture_rate"`
	Confidence   *ConfidenceBand `json:"confidence"`
	StrukTerbaca int             `json:"struk_terbaca" validate:"min=0"`
	PintuDicacah int             `json:"pintu_dicacah" validate:"min=0"`
	PintuDitahan int             `json:"pintu_ditahan" validate:"min=0"`

	Peak        *StationPeak          `json:"peak" validate:"omitempty"`
	Composition []CategoryComposition `json:"composition" validate:"dive"`

	Basis string `json:"basis" validate:"required,oneof=monte-carlo-simpul agregat-titik"`
}

// Validate covers the cross-field rules the struct tags cannot express.
func (w StationSummaryWrite) Validate() error {
	for _, r := range []struct {
		name  string
		value MoneyRange
	}{{"potensi", w.Potensi}, {"tertangkap", w.Tertangkap}, {"gap", w.Gap}} {
		if r.value.P10 > r.value.P50 || r.value.P50 > r.value.P90 {
			return fmt.Errorf("%s: p10/p50/p90 tidak terurut", r.name)
		}
	}
	if w.Peak != nil {
		switch w.Peak.TimeSlot {
		case "morning", "midday", "evening", "night":
		default:
			return fmt.Errorf("peak.time_slot %q tidak dikenal", w.Peak.TimeSlot)
		}
	}
	return nil
}

// UpsertStationSummary writes one station rollup and its category composition
// in a single transaction, keyed on (station_id, day_type) to match the
// table's UNIQUE constraint. Re-posting the same job replaces the row rather
// than adding one, so a retried pipeline run is a no-op — the same
// idempotency contract as the other /pipeline/* callbacks.
func (r *Repository) UpsertStationSummary(ctx context.Context, in StationSummaryWrite) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var peakLabel, peakSlot *string
	var peakF, peakE, peakC, peakV *float64
	var peakGapP10, peakGapP50, peakGapP90 *float64
	if in.Peak != nil {
		p := in.Peak
		peakLabel, peakSlot = &p.PointLabel, &p.TimeSlot
		peakF, peakE, peakC, peakV = &p.F, &p.E, &p.C, &p.V
		peakGapP10, peakGapP50, peakGapP90 = &p.Gap.P10, &p.Gap.P50, &p.Gap.P90
	}
	var confMin, confMax *float64
	if in.Confidence != nil {
		confMin, confMax = &in.Confidence.Min, &in.Confidence.Max
	}

	var summaryID string
	err = tx.QueryRow(ctx, `
		INSERT INTO station_summary (
			job_id, station_id, day_type,
			potential_p10, potential_p50, potential_p90,
			captured_p10, captured_p50, captured_p90,
			gap_p10, gap_p50, gap_p90,
			capture_rate, confidence_min, confidence_max, struk_terbaca,
			pintu_dicacah, pintu_ditahan,
			peak_point_label, peak_time_slot, peak_f, peak_e, peak_c, peak_v,
			peak_gap_p10, peak_gap_p50, peak_gap_p90, basis, computed_at
		) VALUES (
			$1,$2,$3, $4,$5,$6, $7,$8,$9, $10,$11,$12,
			$13,$14,$15,$16, $17,$18,
			$19,$20,$21,$22,$23,$24, $25,$26,$27, $28, now()
		)
		ON CONFLICT (station_id, day_type) DO UPDATE SET
			job_id           = EXCLUDED.job_id,
			potential_p10    = EXCLUDED.potential_p10,
			potential_p50    = EXCLUDED.potential_p50,
			potential_p90    = EXCLUDED.potential_p90,
			captured_p10     = EXCLUDED.captured_p10,
			captured_p50     = EXCLUDED.captured_p50,
			captured_p90     = EXCLUDED.captured_p90,
			gap_p10          = EXCLUDED.gap_p10,
			gap_p50          = EXCLUDED.gap_p50,
			gap_p90          = EXCLUDED.gap_p90,
			capture_rate     = EXCLUDED.capture_rate,
			confidence_min   = EXCLUDED.confidence_min,
			confidence_max   = EXCLUDED.confidence_max,
			struk_terbaca    = EXCLUDED.struk_terbaca,
			pintu_dicacah    = EXCLUDED.pintu_dicacah,
			pintu_ditahan    = EXCLUDED.pintu_ditahan,
			peak_point_label = EXCLUDED.peak_point_label,
			peak_time_slot   = EXCLUDED.peak_time_slot,
			peak_f           = EXCLUDED.peak_f,
			peak_e           = EXCLUDED.peak_e,
			peak_c           = EXCLUDED.peak_c,
			peak_v           = EXCLUDED.peak_v,
			peak_gap_p10     = EXCLUDED.peak_gap_p10,
			peak_gap_p50     = EXCLUDED.peak_gap_p50,
			peak_gap_p90     = EXCLUDED.peak_gap_p90,
			basis            = EXCLUDED.basis,
			computed_at      = now()
		RETURNING id
	`,
		in.JobID, in.StationID, in.DayType,
		in.Potensi.P10, in.Potensi.P50, in.Potensi.P90,
		in.Tertangkap.P10, in.Tertangkap.P50, in.Tertangkap.P90,
		in.Gap.P10, in.Gap.P50, in.Gap.P90,
		in.CaptureRate, confMin, confMax, in.StrukTerbaca,
		in.PintuDicacah, in.PintuDitahan,
		peakLabel, peakSlot, peakF, peakE, peakC, peakV,
		peakGapP10, peakGapP50, peakGapP90, in.Basis,
	).Scan(&summaryID)
	if err != nil {
		return err
	}

	// Replace the composition wholesale rather than upserting row by row: a
	// re-run may drop a category entirely (no gerai left in it), and a
	// leftover row would keep showing on the map after the pipeline stopped
	// reporting it.
	if _, err := tx.Exec(ctx,
		`DELETE FROM station_summary_category WHERE station_summary_id = $1`, summaryID,
	); err != nil {
		return err
	}
	for _, c := range in.Composition {
		if _, err := tx.Exec(ctx, `
			INSERT INTO station_summary_category
				(station_summary_id, category, demand_share, gerai_count, is_missing)
			VALUES ($1,$2,$3,$4,$5)
		`, summaryID, c.Category, c.DemandShare, c.GeraiCount, c.IsMissing); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}
