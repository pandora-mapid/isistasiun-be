package rental

import "time"

type AssetResponse struct {
	ID                     string     `json:"id"`
	StationID              string     `json:"station_id"`
	StationName            string     `json:"station_name"`
	StationCode            string     `json:"station_code"`
	SourceID               string     `json:"source_id"`
	DataSource             string     `json:"data_source"`
	LocationName           string     `json:"location_name"`
	PlotName               string     `json:"plot_name"`
	AreaName               *string    `json:"area_name"`
	Latitude               float64    `json:"latitude"`
	Longitude              float64    `json:"longitude"`
	LandArea               *float64   `json:"land_area"`
	BuildingArea           *float64   `json:"building_area"`
	Rented                 bool       `json:"rented"`
	AvailabilityStatus     string     `json:"availability_status"`
	CommercialValue        *float64   `json:"commercial_value"`
	CommercialValueVisible bool       `json:"commercial_value_visible"`
	SourceUpdatedAt        *time.Time `json:"source_updated_at"`
	Note                   *string    `json:"note"`
}

func toResponse(asset Asset) AssetResponse {
	commercialValue := asset.CommercialValue
	if !asset.CommercialValueVisible {
		commercialValue = nil
	}
	return AssetResponse{
		ID:                     asset.ID,
		StationID:              asset.StationID,
		StationName:            asset.StationName,
		StationCode:            asset.StationCode,
		SourceID:               asset.SourceID,
		DataSource:             asset.DataSource,
		LocationName:           asset.LocationName,
		PlotName:               asset.PlotName,
		AreaName:               asset.AreaName,
		Latitude:               asset.Latitude,
		Longitude:              asset.Longitude,
		LandArea:               asset.LandArea,
		BuildingArea:           asset.BuildingArea,
		Rented:                 asset.Rented,
		AvailabilityStatus:     asset.AvailabilityStatus,
		CommercialValue:        commercialValue,
		CommercialValueVisible: asset.CommercialValueVisible,
		SourceUpdatedAt:        asset.SourceUpdatedAt,
		Note:                   asset.Note,
	}
}
