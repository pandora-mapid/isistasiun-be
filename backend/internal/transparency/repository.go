package transparency

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("transparency record not found")

type Record struct {
	ID            string  `json:"id"`
	Type          string  `json:"type"` // "struk" | "gerai" | "properti"
	StationID     string  `json:"station_id"`
	PhotoURL      string  `json:"photo_url"`      // R2 public URL
	AIReading     string  `json:"ai_reading"`      // structured JSON as text, or summary
	ConfidenceScore float64 `json:"confidence_score"`
	IsAmbiguous   bool    `json:"is_ambiguous"`
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetStruk(ctx context.Context, id string) (*Record, error) {
	return r.getOne(ctx, `
		SELECT id, 'struk', station_id, photo_url, ai_reading, confidence, is_ambiguous
		FROM transparency_struk WHERE id = $1
	`, id)
}

func (r *Repository) GetGerai(ctx context.Context, id string) (*Record, error) {
	return r.getOne(ctx, `
		SELECT id, 'gerai', station_id, photo_url, ai_reading, confidence, false
		FROM transparency_gerai WHERE id = $1
	`, id)
}

func (r *Repository) GetProperti(ctx context.Context, id string) (*Record, error) {
	return r.getOne(ctx, `
		SELECT id, 'properti', station_id, photo_url, ai_reading, confidence, false
		FROM transparency_properti WHERE id = $1
	`, id)
}

func (r *Repository) getOne(ctx context.Context, query, id string) (*Record, error) {
	var rec Record
	err := r.db.QueryRow(ctx, query, id).Scan(
		&rec.ID, &rec.Type, &rec.StationID, &rec.PhotoURL, &rec.AIReading, &rec.ConfidenceScore, &rec.IsAmbiguous,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

func (r *Repository) ListByStation(ctx context.Context, stationID string) ([]Record, error) {
	query := `
		SELECT id, 'struk', station_id, photo_url, ai_reading, confidence, is_ambiguous FROM transparency_struk WHERE station_id = $1
		UNION ALL
		SELECT id, 'gerai', station_id, photo_url, ai_reading, confidence, false FROM transparency_gerai WHERE station_id = $1
		UNION ALL
		SELECT id, 'properti', station_id, photo_url, ai_reading, confidence, false FROM transparency_properti WHERE station_id = $1
	`
	rows, err := r.db.Query(ctx, query, stationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Record
	for rows.Next() {
		var rec Record
		if err := rows.Scan(&rec.ID, &rec.Type, &rec.StationID, &rec.PhotoURL, &rec.AIReading, &rec.ConfidenceScore, &rec.IsAmbiguous); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}
