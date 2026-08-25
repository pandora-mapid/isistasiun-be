package analytics

import (
	"context"

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
		WHERE ($1 = '' OR g.station_id = $1)
		ORDER BY g.time_slot ASC
	`
	rows, err := r.db.Query(ctx, query, stationID)
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
		WHERE ($1 = '' OR station_id = $1)
	`
	rows, err := r.db.Query(ctx, query, stationID)
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
		WHERE ($1 = '' OR station_id = $1)
	`
	rows, err := r.db.Query(ctx, query, stationID)
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
		WHERE ($1 = '' OR station_id = $1)
		ORDER BY activation_score DESC
	`
	rows, err := r.db.Query(ctx, query, stationID)
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
