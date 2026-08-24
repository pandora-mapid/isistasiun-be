package station

import "time"

type AreaType string

const (
	AreaTypeResidential AreaType = "residential"
	AreaTypeOffice      AreaType = "office"
	AreaTypeMixed       AreaType = "mixed"
)

type Station struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Code         string    `json:"code"`
	Operator     string    `json:"operator"`
	AreaType     AreaType  `json:"area_type"`
	Latitude     float64   `json:"latitude"`
	Longitude    float64   `json:"longitude"`
	EntranceCount int      `json:"entrance_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Entrance struct {
	ID        string  `json:"id"`
	StationID string  `json:"station_id"`
	Label     string  `json:"label"` // e.g. "T1", "T2"
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}
