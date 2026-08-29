package premium

// DeepAnalysisResponse is the paid tier's per-station bundle (section 4.1:
// operator / pengelola kawasan). It answers in one call what the free
// endpoints only answer one table at a time, and adds the things the public
// tier deliberately withholds: the per-slot breakdown, the plots flagged as
// rent-vs-flow outliers, and the sample-confidence detail behind each figure.
//
// Every rupiah figure stays a P10-P90 range — section 3.3 forbids presenting
// a single number as if the estimate were certain.
type DeepAnalysisResponse struct {
	Station        StationSummary       `json:"station"`
	SpendingGap    []SpendingGapSlot    `json:"spending_gap"`
	Totals         Totals               `json:"totals"`
	CategoryGaps   []CategoryGap        `json:"category_gaps"`
	RentFlowPlots  []RentFlowPlot       `json:"rent_flow_plots"`
	EventPotential []EventPotentialZone `json:"event_potential"`
	Confidence     []ConfidenceZone     `json:"confidence"`
	Coverage       Coverage             `json:"coverage"`
}

type StationSummary struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Code          string `json:"code"`
	Operator      string `json:"operator"`
	AreaType      string `json:"area_type"`
	EntranceCount int    `json:"entrance_count"`
}

// Range mirrors the Monte Carlo output. P50 is absent because the pipeline
// does not persist it yet — see the mismatch register, item 6.
type Range struct {
	P10 float64 `json:"p10"`
	P90 float64 `json:"p90"`
}

type SpendingGapSlot struct {
	TimeSlot   string `json:"time_slot"`
	Potential  Range  `json:"potential"`
	Captured   Range  `json:"captured"`
	Gap        Range  `json:"gap"`
	ComputedAt string `json:"computed_at"`
}

// Totals sums the four counted slots. Hours between slots are not counted and
// not interpolated (section 3.3), so this is a sum over observed slots — never
// a daily figure.
type Totals struct {
	Potential      Range    `json:"potential"`
	Captured       Range    `json:"captured"`
	Gap            Range    `json:"gap"`
	SlotsWithData  int      `json:"slots_with_data"`
	SlotsExpected  int      `json:"slots_expected"`
	CaptureRateP50 *float64 `json:"capture_rate_p50"`
}

type CategoryGap struct {
	Category           string `json:"category"`
	DemandInArea       bool   `json:"demand_in_area"`
	AvailableInStation bool   `json:"available_in_station"`
	// Missing is the actual recommendation: demand readable in the catchment,
	// no outlet inside the station.
	Missing bool `json:"missing"`
}

type RentFlowPlot struct {
	PlotID       string  `json:"plot_id"`
	OfferedRent  float64 `json:"offered_rent"`
	MeasuredFlow float64 `json:"measured_flow"`
	Index        float64 `json:"index"`
	IsOutlier    bool    `json:"is_outlier"`
}

type EventPotentialZone struct {
	ZoneID          string  `json:"zone_id"`
	ActivationScore float64 `json:"activation_score"`
	RecommendedSlot string  `json:"recommended_slot"`
}

type ConfidenceZone struct {
	ZoneID          string  `json:"zone_id"`
	SampleCount     int     `json:"sample_count"`
	IsThinSample    bool    `json:"is_thin_sample"`
	ConfidenceScore float64 `json:"confidence_score"`
}

// Coverage is what makes the numbers above auditable rather than merely
// printed. Lampiran 2 requires thin-sample areas be flagged, not silently
// estimated; a caller that ignores this is reading the ranges wrong.
type Coverage struct {
	ThinSampleZones  int  `json:"thin_sample_zones"`
	TotalZones       int  `json:"total_zones"`
	StrukTotal       int  `json:"struk_total"`
	StrukAmbiguous   int  `json:"struk_ambiguous"`
	StrukUsable      int  `json:"struk_usable"`
	HasCompleteSlots bool `json:"has_complete_slots"`
}
