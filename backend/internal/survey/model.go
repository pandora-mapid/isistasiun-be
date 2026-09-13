package survey

import "time"

// FlowObservation = one 15-minute pedestrian count block at one entrance
// (variable F). Matches the field protocol: NCHRP 797, blocks 15 min per
// entrance per time slot, 2 blocks per slot.
type FlowObservation struct {
	ID              string    `json:"id"`
	StationID       string    `json:"station_id"`
	EntranceID      string    `json:"entrance_id"`
	TimeSlot        string    `json:"time_slot"` // "morning" | "midday" | "evening" | "night"
	ObservedAt      time.Time `json:"observed_at"`
	BlockNumber     int       `json:"block_number"` // 1 or 2
	PedestrianCount int       `json:"pedestrian_count"`
	Direction       string    `json:"direction"` // "in" | "out"
	WeatherNote     string    `json:"weather_note,omitempty"`
	SurveyorID      string    `json:"surveyor_id"`
	CreatedAt       time.Time `json:"created_at"`

	// IdempotencyKey carries the optional Idempotency-Key request header. When
	// set, a repeated submission returns the original row instead of inserting
	// a duplicate. Not serialized back to clients.
	IdempotencyKey string `json:"-"`
}

// EntryConversionObservation = variables E and C measured together in front
// of one gerai during one block, per methodology section 5.2.
type EntryConversionObservation struct {
	ID                     string    `json:"id"`
	StationID              string    `json:"station_id"`
	GeraiID                string    `json:"gerai_id"`
	Category               string    `json:"category"` // one of the 5 baku categories
	TimeSlot               string    `json:"time_slot"`
	ObservedAt             time.Time `json:"observed_at"`
	BlockNumber            int       `json:"block_number"`
	PassersBy              int       `json:"passers_by"`               // denominator for E
	EnteredCount           int       `json:"entered_count"`            // numerator for E / denominator for C
	CompletedPurchaseCount int       `json:"completed_purchase_count"` // numerator for C
	SurveyorID             string    `json:"surveyor_id"`
	CreatedAt              time.Time `json:"created_at"`

	IdempotencyKey string `json:"-"` // see FlowObservation.IdempotencyKey
}
