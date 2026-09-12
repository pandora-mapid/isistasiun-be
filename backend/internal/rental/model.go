package rental

import "time"

type Asset struct {
	ID                     string
	StationID              string
	StationName            string
	StationCode            string
	SourceID               string
	DataSource             string
	LocationName           string
	PlotName               string
	AreaName               *string
	Latitude               float64
	Longitude              float64
	LandArea               *float64
	BuildingArea           *float64
	Rented                 bool
	AvailabilityStatus     string
	CommercialValue        *float64
	CommercialValueVisible bool
	SourceUpdatedAt        *time.Time
	Note                   *string
}

type ListParams struct {
	StationID   string
	StationCode string
	Status      string
}
