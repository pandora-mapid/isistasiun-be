package station

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("station not found")

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) List(ctx context.Context, areaType, operator string) ([]Station, error) {
	query := `
		SELECT id, name, code, operator, area_type,
		       ST_Y(location::geometry) AS latitude,
		       ST_X(location::geometry) AS longitude,
		       (SELECT COUNT(*) FROM station_entrances e WHERE e.station_id = s.id) AS entrance_count,
		       created_at, updated_at
		FROM stations s
		WHERE ($1 = '' OR area_type = $1)
		  AND ($2 = '' OR operator = $2)
		ORDER BY name ASC
	`
	rows, err := r.db.Query(ctx, query, areaType, operator)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stations []Station
	for rows.Next() {
		var s Station
		if err := rows.Scan(
			&s.ID, &s.Name, &s.Code, &s.Operator, &s.AreaType,
			&s.Latitude, &s.Longitude, &s.EntranceCount,
			&s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		stations = append(stations, s)
	}
	return stations, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id string) (*Station, error) {
	query := `
		SELECT id, name, code, operator, area_type,
		       ST_Y(location::geometry) AS latitude,
		       ST_X(location::geometry) AS longitude,
		       (SELECT COUNT(*) FROM station_entrances e WHERE e.station_id = s.id) AS entrance_count,
		       created_at, updated_at
		FROM stations s
		WHERE id = $1
	`
	var s Station
	err := r.db.QueryRow(ctx, query, id).Scan(
		&s.ID, &s.Name, &s.Code, &s.Operator, &s.AreaType,
		&s.Latitude, &s.Longitude, &s.EntranceCount,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repository) ListEntrances(ctx context.Context, stationID string) ([]Entrance, error) {
	query := `
		SELECT id, station_id, label,
		       ST_Y(location::geometry) AS latitude,
		       ST_X(location::geometry) AS longitude
		FROM station_entrances
		WHERE station_id = $1
		ORDER BY label ASC
	`
	rows, err := r.db.Query(ctx, query, stationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entrances []Entrance
	for rows.Next() {
		var e Entrance
		if err := rows.Scan(&e.ID, &e.StationID, &e.Label, &e.Latitude, &e.Longitude); err != nil {
			return nil, err
		}
		entrances = append(entrances, e)
	}
	return entrances, rows.Err()
}
