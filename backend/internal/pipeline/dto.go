package pipeline

// These DTOs match what the Python batch pipeline (Gemini OCR + Monte Carlo)
// posts back after each stage. Idempotency: pipeline sends `job_id` +
// `source_ref`, callback upserts on that composite so re-running a batch job
// doesn't duplicate rows.
//
// PhotoURL is the already-redacted, publicly servable R2 URL of the source
// photo. When present, the callback also writes a transparency_* evidence card
// linked to the extraction (see repository.go). Redaction is the pipeline's
// job — the API never derives a URL from SourceRef, so an un-redacted original
// can't leak through the transparency panel.

type StrukExtractionCallback struct {
	JobID         string  `json:"job_id" validate:"required"`
	SourceRef     string  `json:"source_ref" validate:"required"` // R2 object key of the receipt photo
	PhotoURL      string  `json:"photo_url" validate:"omitempty,url"`
	StationID     string  `json:"station_id" validate:"required"`
	Category      string  `json:"category" validate:"required,oneof=makanan_minuman ritel_kemasan apotek_kesehatan jasa lainnya"`
	FinalAmount   float64 `json:"final_amount" validate:"min=0"` // 0 allowed only when is_ambiguous (checked in handler)
	PaymentMethod string  `json:"payment_method"`
	TransactedAt  string  `json:"transacted_at" validate:"required"`
	Confidence    float64 `json:"confidence" validate:"min=0,max=1"`
	IsAmbiguous   bool    `json:"is_ambiguous"`
}

type PropertiExtractionCallback struct {
	JobID       string  `json:"job_id" validate:"required"`
	SourceRef   string  `json:"source_ref" validate:"required"`
	PhotoURL    string  `json:"photo_url" validate:"omitempty,url"`
	StationID   string  `json:"station_id" validate:"required"`
	PlotID      string  `json:"plot_id" validate:"required"`
	OfferedRent float64 `json:"offered_rent" validate:"required,min=0"`
	AreaSqm     float64 `json:"area_sqm" validate:"required,min=0"`
	Confidence  float64 `json:"confidence" validate:"min=0,max=1"`
}

type GeraiClassificationCallback struct {
	JobID      string  `json:"job_id" validate:"required"`
	SourceRef  string  `json:"source_ref" validate:"required"`
	PhotoURL   string  `json:"photo_url" validate:"omitempty,url"`
	StationID  string  `json:"station_id" validate:"required"`
	GeraiID    string  `json:"gerai_id" validate:"required"`
	Category   string  `json:"category" validate:"required,oneof=makanan_minuman ritel_kemasan apotek_kesehatan jasa lainnya"`
	Visibility string  `json:"visibility" validate:"omitempty,oneof=high medium low"`
	Confidence float64 `json:"confidence" validate:"min=0,max=1"`
}

type MonteCarloResultCallback struct {
	JobID            string  `json:"job_id" validate:"required"`
	StationID        string  `json:"station_id" validate:"required"`
	TimeSlot         string  `json:"time_slot" validate:"required"`
	PotentialLowP10  float64 `json:"potential_low_p10"`
	PotentialHighP90 float64 `json:"potential_high_p90"`
	CapturedLowP10   float64 `json:"captured_low_p10"`
	CapturedHighP90  float64 `json:"captured_high_p90"`
	Iterations       int     `json:"iterations" validate:"required,min=10000"` // full run per methodology 3.3
}
