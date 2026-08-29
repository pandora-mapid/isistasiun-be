package survey

type CreateFlowObservationRequest struct {
	StationID       string `json:"station_id" validate:"required"`
	EntranceID      string `json:"entrance_id" validate:"required"`
	TimeSlot        string `json:"time_slot" validate:"required,oneof=morning midday evening night"`
	ObservedAt      string `json:"observed_at" validate:"required"` // RFC3339
	BlockNumber     int    `json:"block_number" validate:"required,min=1,max=2"`
	PedestrianCount int    `json:"pedestrian_count" validate:"min=0"`
	Direction       string `json:"direction" validate:"required,oneof=in out"`
	WeatherNote     string `json:"weather_note"`
	SurveyorID      string `json:"surveyor_id" validate:"required"`
}

type CreateEntryConversionRequest struct {
	StationID              string `json:"station_id" validate:"required"`
	GeraiID                string `json:"gerai_id" validate:"required"`
	Category               string `json:"category" validate:"required,oneof=makanan_minuman ritel_kemasan apotek_kesehatan jasa lainnya"`
	TimeSlot               string `json:"time_slot" validate:"required,oneof=morning midday evening night"`
	ObservedAt             string `json:"observed_at" validate:"required"`
	BlockNumber            int    `json:"block_number" validate:"required,min=1,max=2"`
	PassersBy              int    `json:"passers_by" validate:"min=0"`
	EnteredCount           int    `json:"entered_count" validate:"min=0"`
	CompletedPurchaseCount int    `json:"completed_purchase_count" validate:"min=0"`
	SurveyorID             string `json:"surveyor_id" validate:"required"`
}

type ListFlowObservationsQuery struct {
	StationID  string `query:"station_id"`
	EntranceID string `query:"entrance_id"`
	TimeSlot   string `query:"time_slot"`
}
