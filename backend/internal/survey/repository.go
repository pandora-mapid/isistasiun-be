package survey

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) InsertFlowObservation(ctx context.Context, o FlowObservation) (string, error) {
	// NULLIF keeps the empty (no-header) case out of the partial unique index,
	// so those rows never collide. A keyed replay updates the original row so
	// UPDATE lets RETURNING hand back the original row's id — idempotent replay.
	query := `
		INSERT INTO flow_observations
			(station_id, entrance_id, time_slot, observed_at, block_number,
			 pedestrian_count, direction, weather_note, surveyor_id, idempotency_key)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NULLIF($10,''))
		ON CONFLICT (idempotency_key) WHERE idempotency_key IS NOT NULL
		DO UPDATE SET station_id = EXCLUDED.station_id,
		              entrance_id = EXCLUDED.entrance_id,
		              time_slot = EXCLUDED.time_slot,
		              observed_at = EXCLUDED.observed_at,
		              block_number = EXCLUDED.block_number,
		              pedestrian_count = EXCLUDED.pedestrian_count,
		              direction = EXCLUDED.direction,
		              weather_note = EXCLUDED.weather_note,
		              surveyor_id = EXCLUDED.surveyor_id
		RETURNING id
	`
	var id string
	err := r.db.QueryRow(ctx, query,
		o.StationID, o.EntranceID, o.TimeSlot, o.ObservedAt, o.BlockNumber,
		o.PedestrianCount, o.Direction, o.WeatherNote, o.SurveyorID, o.IdempotencyKey,
	).Scan(&id)
	return id, err
}

func (r *Repository) InsertEntryConversion(ctx context.Context, o EntryConversionObservation) (string, error) {
	query := `
		INSERT INTO entry_conversion_observations
			(station_id, gerai_id, category, time_slot, observed_at, block_number,
			 passers_by, entered_count, completed_purchase_count, surveyor_id, idempotency_key)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NULLIF($11,''))
		ON CONFLICT (idempotency_key) WHERE idempotency_key IS NOT NULL
		DO UPDATE SET station_id = EXCLUDED.station_id,
		              gerai_id = EXCLUDED.gerai_id,
		              category = EXCLUDED.category,
		              time_slot = EXCLUDED.time_slot,
		              observed_at = EXCLUDED.observed_at,
		              block_number = EXCLUDED.block_number,
		              passers_by = EXCLUDED.passers_by,
		              entered_count = EXCLUDED.entered_count,
		              completed_purchase_count = EXCLUDED.completed_purchase_count,
		              surveyor_id = EXCLUDED.surveyor_id
		RETURNING id
	`
	var id string
	err := r.db.QueryRow(ctx, query,
		o.StationID, o.GeraiID, o.Category, o.TimeSlot, o.ObservedAt, o.BlockNumber,
		o.PassersBy, o.EnteredCount, o.CompletedPurchaseCount, o.SurveyorID, o.IdempotencyKey,
	).Scan(&id)
	return id, err
}

func (r *Repository) ListFlowObservations(ctx context.Context, stationID, entranceID, timeSlot string) ([]FlowObservation, error) {
	query := `
		SELECT id, station_id, entrance_id, time_slot, observed_at, block_number,
		       pedestrian_count, direction, COALESCE(weather_note, ''), surveyor_id, created_at
		FROM flow_observations
		-- The id filters arrive as strings ("" means "no filter"), so the UUID
		-- columns are cast rather than the parameters: comparing uuid = text
		-- directly has no operator and fails the whole query.
		WHERE ($1 = '' OR station_id::text = $1)
		  AND ($2 = '' OR entrance_id::text = $2)
		  AND ($3 = '' OR time_slot = $3)
		ORDER BY observed_at DESC
		LIMIT 500
	`
	rows, err := r.db.Query(ctx, query, stationID, entranceID, timeSlot)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []FlowObservation
	for rows.Next() {
		var o FlowObservation
		if err := rows.Scan(
			&o.ID, &o.StationID, &o.EntranceID, &o.TimeSlot, &o.ObservedAt, &o.BlockNumber,
			&o.PedestrianCount, &o.Direction, &o.WeatherNote, &o.SurveyorID, &o.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// parseTime is a small helper kept local to avoid pulling a time-parsing dep
// into the service layer; DTOs carry RFC3339 strings from the mobile survey app.
func parseTime(v string) (time.Time, error) {
	return time.Parse(time.RFC3339, v)
}
