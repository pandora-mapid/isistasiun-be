package analytics

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository reads pre-computed analytics tables. All heavy computation
// (Monte Carlo, spatial join, weighted overlay/AHP) happens in the Python
// batch pipeline; this layer is read-only and should stay thin/fast.
type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) SpendingGap(ctx context.Context, stationID string) ([]SpendingGapResponse, error) {
	query := `
		SELECT g.station_id, s.name, g.potential_low_p10, g.potential_high_p90,
		       g.captured_low_p10, g.captured_high_p90, g.gap_low_p10, g.gap_high_p90,
		       g.time_slot, g.computed_at
		FROM spending_gap_estimates g
		JOIN stations s ON s.id = g.station_id
	`
	args := make([]any, 0, 1)
	if stationID != "" {
		query += " WHERE g.station_id = $1"
		args = append(args, stationID)
	}
	query += ` ORDER BY s.name ASC, g.station_id ASC,
		CASE g.time_slot
			WHEN 'morning' THEN 1
			WHEN 'midday' THEN 2
			WHEN 'evening' THEN 3
			WHEN 'night' THEN 4
		END ASC`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out = make([]SpendingGapResponse, 0)
	for rows.Next() {
		var g SpendingGapResponse
		if err := rows.Scan(
			&g.StationID, &g.StationName, &g.PotentialLow, &g.PotentialHigh,
			&g.CapturedLow, &g.CapturedHigh, &g.GapLow, &g.GapHigh,
			&g.TimeSlot, &g.ComputedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (r *Repository) CategoryGap(ctx context.Context, stationID string) ([]CategoryGapResponse, error) {
	query := `
		SELECT station_id, category, demand_in_area, available_in_station
		FROM category_gap_estimates
	`
	query, args := withOptionalStationFilter(query, stationID)
	query += " ORDER BY station_id ASC, category ASC"
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out = make([]CategoryGapResponse, 0)
	for rows.Next() {
		var c CategoryGapResponse
		if err := rows.Scan(&c.StationID, &c.Category, &c.DemandInArea, &c.AvailableInStation); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repository) RentFlowIndex(ctx context.Context, stationID string) ([]RentFlowIndexResponse, error) {
	query := `
		SELECT plot_id, station_id, offered_rent, measured_flow, index_value, is_outlier
		FROM rent_flow_index
	`
	query, args := withOptionalStationFilter(query, stationID)
	query += " ORDER BY station_id ASC, plot_id ASC"
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out = make([]RentFlowIndexResponse, 0)
	for rows.Next() {
		var rf RentFlowIndexResponse
		if err := rows.Scan(&rf.PlotID, &rf.StationID, &rf.OfferedRent, &rf.MeasuredFlow, &rf.Index, &rf.IsOutlier); err != nil {
			return nil, err
		}
		out = append(out, rf)
	}
	return out, rows.Err()
}

func (r *Repository) EventPotential(ctx context.Context, stationID string) ([]EventPotentialResponse, error) {
	query := `
		SELECT station_id, zone_id, activation_score, recommended_slot
		FROM event_potential_scores
	`
	query, args := withOptionalStationFilter(query, stationID)
	query += " ORDER BY activation_score DESC, station_id ASC, zone_id ASC"
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out = make([]EventPotentialResponse, 0)
	for rows.Next() {
		var e EventPotentialResponse
		if err := rows.Scan(&e.StationID, &e.ZoneID, &e.ActivationScore, &e.RecommendedSlot); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func withOptionalStationFilter(query, stationID string) (string, []any) {
	if stationID == "" {
		return strings.TrimSpace(query), nil
	}
	return strings.TrimSpace(query) + " WHERE station_id = $1", []any{stationID}
}
