package premium

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrStationNotFound = errors.New("station not found")

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Each reader below is scoped to one station and casts the parameter to uuid
// explicitly. The public analytics repository uses `($1 = '' OR station_id = $1)`
// to make the filter optional, which fails at runtime on a uuid column — there
// is no optional case here, so the cast is unambiguous.

func (r *Repository) Station(ctx context.Context, stationID string) (*StationSummary, error) {
	var s StationSummary
	err := r.db.QueryRow(ctx, `
		SELECT id, name, code, operator, area_type,
		       (SELECT COUNT(*) FROM station_entrances e WHERE e.station_id = s.id)
		FROM stations s
		WHERE id = $1::uuid
	`, stationID).Scan(&s.ID, &s.Name, &s.Code, &s.Operator, &s.AreaType, &s.EntranceCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrStationNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repository) SpendingGap(ctx context.Context, stationID string) ([]SpendingGapSlot, error) {
	rows, err := r.db.Query(ctx, `
		SELECT time_slot,
		       potential_low_p10, potential_high_p90,
		       captured_low_p10, captured_high_p90,
		       gap_low_p10, gap_high_p90,
		       computed_at::text
		FROM spending_gap_estimates
		WHERE station_id = $1::uuid
		ORDER BY CASE time_slot
		           WHEN 'morning' THEN 1
		           WHEN 'midday'  THEN 2
		           WHEN 'evening' THEN 3
		           WHEN 'night'   THEN 4
		         END
	`, stationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []SpendingGapSlot{}
	for rows.Next() {
		var s SpendingGapSlot
		if err := rows.Scan(
			&s.TimeSlot,
			&s.Potential.P10, &s.Potential.P90,
			&s.Captured.P10, &s.Captured.P90,
			&s.Gap.P10, &s.Gap.P90,
			&s.ComputedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *Repository) CategoryGaps(ctx context.Context, stationID string) ([]CategoryGap, error) {
	rows, err := r.db.Query(ctx, `
		SELECT category, demand_in_area, available_in_station
		FROM category_gap_estimates
		WHERE station_id = $1::uuid
		ORDER BY category
	`, stationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []CategoryGap{}
	for rows.Next() {
		var c CategoryGap
		if err := rows.Scan(&c.Category, &c.DemandInArea, &c.AvailableInStation); err != nil {
			return nil, err
		}
		c.Missing = c.DemandInArea && !c.AvailableInStation
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repository) RentFlowPlots(ctx context.Context, stationID string) ([]RentFlowPlot, error) {
	rows, err := r.db.Query(ctx, `
		SELECT plot_id, offered_rent, measured_flow, index_value, is_outlier
		FROM rent_flow_index
		WHERE station_id = $1::uuid
		ORDER BY is_outlier DESC, index_value DESC
	`, stationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []RentFlowPlot{}
	for rows.Next() {
		var p RentFlowPlot
		if err := rows.Scan(&p.PlotID, &p.OfferedRent, &p.MeasuredFlow, &p.Index, &p.IsOutlier); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *Repository) EventPotential(ctx context.Context, stationID string) ([]EventPotentialZone, error) {
	rows, err := r.db.Query(ctx, `
		SELECT zone_id, activation_score, COALESCE(recommended_slot, '')
		FROM event_potential_scores
		WHERE station_id = $1::uuid
		ORDER BY activation_score DESC
	`, stationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []EventPotentialZone{}
	for rows.Next() {
		var e EventPotentialZone
		if err := rows.Scan(&e.ZoneID, &e.ActivationScore, &e.RecommendedSlot); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *Repository) Confidence(ctx context.Context, stationID string) ([]ConfidenceZone, error) {
	rows, err := r.db.Query(ctx, `
		SELECT zone_id, sample_count, is_thin_sample, confidence_score
		FROM confidence_layer
		WHERE station_id = $1::uuid
		ORDER BY confidence_score ASC
	`, stationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []ConfidenceZone{}
	for rows.Next() {
		var c ConfidenceZone
		if err := rows.Scan(&c.ZoneID, &c.SampleCount, &c.IsThinSample, &c.ConfidenceScore); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// StrukCoverage reports how much of the receipt evidence behind V actually
// survived OCR. Ambiguous receipts are excluded from the simulation but still
// counted, per the rule in section 3.4 — reporting the exclusion rate is part
// of the method, not an afterthought.
func (r *Repository) StrukCoverage(ctx context.Context, stationID string) (total, ambiguous int, err error) {
	err = r.db.QueryRow(ctx, `
		SELECT COUNT(*), COUNT(*) FILTER (WHERE is_ambiguous)
		FROM struk_extractions
		WHERE station_id = $1::uuid
	`, stationID).Scan(&total, &ambiguous)
	return total, ambiguous, err
}
