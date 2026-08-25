package confidence

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

type LayerEntry struct {
	StationID       string  `json:"station_id"`
	ZoneID          string  `json:"zone_id"`
	SampleCount     int     `json:"sample_count"`
	IsThin          bool    `json:"is_thin_sample"` // greyed-out on the map per Gambar 2 methodology
	ConfidenceScore float64 `json:"confidence_score"`
}

func (r *Repository) List(ctx context.Context, stationID string) ([]LayerEntry, error) {
	query := `
		SELECT station_id, zone_id, sample_count, is_thin_sample, confidence_score
		FROM confidence_layer
	`
	args := make([]any, 0, 1)
	if stationID != "" {
		query += " WHERE station_id = $1"
		args = append(args, stationID)
	}
	query += " ORDER BY station_id ASC, zone_id ASC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out = make([]LayerEntry, 0)
	for rows.Next() {
		var e LayerEntry
		if err := rows.Scan(&e.StationID, &e.ZoneID, &e.SampleCount, &e.IsThin, &e.ConfidenceScore); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
