package station

type ListStationsQuery struct {
	AreaType string `query:"area_type"`
	Operator string `query:"operator"`
}

type StationResponse struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Code          string   `json:"code"`
	Operator      string   `json:"operator"`
	AreaType      AreaType `json:"area_type"`
	Latitude      float64  `json:"latitude"`
	Longitude     float64  `json:"longitude"`
	EntranceCount int      `json:"entrance_count"`
}

type EntranceResponse struct {
	ID        string  `json:"id"`
	Label     string  `json:"label"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

func toStationResponse(s Station) StationResponse {
	return StationResponse{
		ID:            s.ID,
		Name:          s.Name,
		Code:          s.Code,
		Operator:      s.Operator,
		AreaType:      s.AreaType,
		Latitude:      s.Latitude,
		Longitude:     s.Longitude,
		EntranceCount: s.EntranceCount,
	}
}

func toEntranceResponse(e Entrance) EntranceResponse {
	return EntranceResponse{
		ID:        e.ID,
		Label:     e.Label,
		Latitude:  e.Latitude,
		Longitude: e.Longitude,
	}
}
