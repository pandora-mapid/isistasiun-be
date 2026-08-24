package copilot

type QueryRequest struct {
	Query     string `json:"query" validate:"required,max=500"`
	StationID string `json:"station_id,omitempty"` // optional scoping context from the map view
}

type QueryResponse struct {
	Answer          string                 `json:"answer"`
	SuggestedLayers []string               `json:"suggested_layers,omitempty"`
	SpatialFilter   map[string]interface{} `json:"spatial_filter,omitempty"`
}
