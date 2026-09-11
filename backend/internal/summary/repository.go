package summary

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository reads the pre-computed station_summary tables. Like
// internal/analytics it stays thin and read-only — every heavy number is
// produced by the Python batch pipeline.
type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) StationSummary(ctx context.Context, stationID string) ([]StationSummaryResponse, error) {
	query := `
		SELECT ss.id::text, ss.station_id::text, st.name, st.area_type, ss.day_type,
		       ss.pintu_dicacah, ss.pintu_ditahan,
		       ss.potential_p10, ss.potential_p50, ss.potential_p90,
		       ss.captured_p10, ss.captured_p50, ss.captured_p90,
		       ss.gap_p10, ss.gap_p50, ss.gap_p90,
		       ss.capture_rate, ss.confidence_min, ss.confidence_max,
		       ss.struk_terbaca,
		       ss.peak_point_label, ss.peak_time_slot,
		       ss.peak_f, ss.peak_e, ss.peak_c, ss.peak_v,
		       ss.peak_gap_p10, ss.peak_gap_p50, ss.peak_gap_p90,
		       ss.basis, ss.computed_at
		FROM station_summary ss
		JOIN stations st ON st.id = ss.station_id
	`
	args := make([]any, 0, 1)
	if stationID != "" {
		query += " WHERE ss.station_id = $1"
		args = append(args, stationID)
	}
	query += " ORDER BY ss.gap_p50 DESC, st.name ASC"

	rows, err := r.db.Query(ctx, strings.TrimSpace(query), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]StationSummaryResponse, 0)
	ids := make([]string, 0)
	// Index, not pointer: `out` reallocates on append, which would dangle any
	// *StationSummaryResponse taken before the last row was read.
	idxByID := make(map[string]int)

	for rows.Next() {
		var (
			id                                 string
			s                                  StationSummaryResponse
			computedAt                         time.Time
			captureRate, confMin, confMax      *float64
			peakLabel, peakSlot                *string
			peakF, peakE, peakC, peakV         *float64
			peakGapP10, peakGapP50, peakGapP90 *float64
		)
		if err := rows.Scan(
			&id, &s.StationID, &s.StationName, &s.Typology, &s.DayType,
			&s.PintuDicacah, &s.PintuDitahan,
			&s.Potensi.P10, &s.Potensi.P50, &s.Potensi.P90,
			&s.Tertangkap.P10, &s.Tertangkap.P50, &s.Tertangkap.P90,
			&s.Gap.P10, &s.Gap.P50, &s.Gap.P90,
			&captureRate, &confMin, &confMax,
			&s.StrukTerbaca,
			&peakLabel, &peakSlot,
			&peakF, &peakE, &peakC, &peakV,
			&peakGapP10, &peakGapP50, &peakGapP90,
			&s.Basis, &computedAt,
		); err != nil {
			return nil, err
		}
		s.ComputedAt = computedAt.UTC().Format(time.RFC3339)

		s.CaptureRate = captureRate
		if confMin != nil && confMax != nil {
			s.Confidence = &ConfidenceBand{Min: *confMin, Max: *confMax}
		}
		if peakLabel != nil && peakSlot != nil {
			s.Peak = &StationPeak{
				PointLabel: *peakLabel,
				TimeSlot:   *peakSlot,
				F:          deref(peakF), E: deref(peakE), C: deref(peakC), V: deref(peakV),
				Gap: MoneyRange{
					P10: deref(peakGapP10), P50: deref(peakGapP50), P90: deref(peakGapP90),
				},
			}
		}
		s.Composition = make([]CategoryComposition, 0)

		out = append(out, s)
		ids = append(ids, id)
		idxByID[id] = len(out) - 1
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return out, nil
	}

	catRows, err := r.db.Query(ctx, `
		SELECT station_summary_id::text, category, demand_share, gerai_count, is_missing
		FROM station_summary_category
		WHERE station_summary_id = ANY($1::uuid[])
		ORDER BY demand_share DESC
	`, ids)
	if err != nil {
		return nil, err
	}
	defer catRows.Close()

	for catRows.Next() {
		var summaryID string
		var c CategoryComposition
		if err := catRows.Scan(&summaryID, &c.Category, &c.DemandShare, &c.GeraiCount, &c.IsMissing); err != nil {
			return nil, err
		}
		if i, ok := idxByID[summaryID]; ok {
			out[i].Composition = append(out[i].Composition, c)
		}
	}
	return out, catRows.Err()
}

func deref(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v
}
