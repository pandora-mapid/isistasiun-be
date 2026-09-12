package rental

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) List(ctx context.Context, params ListParams) ([]Asset, error) {
	query := `
		SELECT ra.id, ra.station_id, s.name, s.code, ra.source_id,
		       ra.data_source, ra.location_name, ra.plot_name, ra.area_name,
		       ra.latitude, ra.longitude, ra.land_area, ra.building_area,
		       ra.rented, ra.availability_status, ra.commercial_value,
		       ra.commercial_value_visible, ra.source_updated_at, ra.note
		FROM rental_assets ra
		JOIN stations s ON s.id = ra.station_id
		WHERE ($1 = '' OR ra.station_id = $1::uuid)
		  AND ($2 = '' OR s.code = $2)
		  AND ($3 = '' OR ra.availability_status = $3)
		ORDER BY s.name ASC, ra.availability_status ASC, ra.plot_name ASC
	`
	rows, err := r.db.Query(ctx, query, params.StationID, strings.ToUpper(params.StationCode), params.Status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	assets := make([]Asset, 0)
	for rows.Next() {
		var asset Asset
		if err := rows.Scan(
			&asset.ID, &asset.StationID, &asset.StationName, &asset.StationCode,
			&asset.SourceID, &asset.DataSource, &asset.LocationName, &asset.PlotName,
			&asset.AreaName, &asset.Latitude, &asset.Longitude, &asset.LandArea,
			&asset.BuildingArea, &asset.Rented, &asset.AvailabilityStatus,
			&asset.CommercialValue, &asset.CommercialValueVisible,
			&asset.SourceUpdatedAt, &asset.Note,
		); err != nil {
			return nil, err
		}
		assets = append(assets, asset)
	}
	return assets, rows.Err()
}
