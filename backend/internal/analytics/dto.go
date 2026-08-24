package analytics

// SpendingGapResponse mirrors the Monte Carlo output (P10-P90 range) written
// by the batch pipeline into the spending_gap_estimates table.
type SpendingGapResponse struct {
	StationID    string  `json:"station_id"`
	StationName  string  `json:"station_name"`
	PotentialLow float64 `json:"potential_low_p10"`
	PotentialHigh float64 `json:"potential_high_p90"`
	CapturedLow  float64 `json:"captured_low_p10"`
	CapturedHigh float64 `json:"captured_high_p90"`
	GapLow       float64 `json:"gap_low_p10"`
	GapHigh      float64 `json:"gap_high_p90"`
	TimeSlot     string  `json:"time_slot"`
	ComputedAt   string  `json:"computed_at"`
}

type CategoryGapResponse struct {
	StationID string `json:"station_id"`
	Category  string `json:"category"`
	DemandInArea bool `json:"demand_in_area"`
	AvailableInStation bool `json:"available_in_station"`
}

type RentFlowIndexResponse struct {
	PlotID       string  `json:"plot_id"`
	StationID    string  `json:"station_id"`
	OfferedRent  float64 `json:"offered_rent"`
	MeasuredFlow float64 `json:"measured_flow"`
	Index        float64 `json:"index"`
	IsOutlier    bool    `json:"is_outlier"`
}

type EventPotentialResponse struct {
	StationID       string  `json:"station_id"`
	ZoneID          string  `json:"zone_id"`
	ActivationScore float64 `json:"activation_score"`
	RecommendedSlot string  `json:"recommended_slot"`
}
