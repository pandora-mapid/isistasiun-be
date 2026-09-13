// Package summary serves the station-level rollup behind the free-tier
// "ringkasan & perbandingan antarsimpul" (Compare View, 2 simpul).
//
// It is a sibling of internal/analytics, not part of it: analytics answers
// "what is happening at one observation point on one slot" (Priyapta);
// summary answers "what does this station look like as a whole, and how do
// two stations compare" (Arzaka — see ../Context/02-BACKEND-SPEC.md §2).
// Kept in its own package so the two owners' changes don't collide.
//
// Read-only. The numbers are produced by the Python batch pipeline
// (station-scoped Monte Carlo) and land here via the pipeline callback.
package summary

// MoneyRange is a Monte Carlo P10/P50/P90 range in rupiah.
type MoneyRange struct {
	P10 float64 `json:"p10"`
	P50 float64 `json:"p50"`
	P90 float64 `json:"p90"`
}

// ConfidenceBand is the spread of per-point confidence scores that fed the
// station roll-up. Null when no point was estimable.
type ConfidenceBand struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

// CategoryComposition is one business category at the station level: area
// demand vs. how many gerai actually serve it inside the station.
type CategoryComposition struct {
	Category    string  `json:"category"` // makanan_minuman | ritel_kemasan | apotek_kesehatan | jasa | lainnya
	DemandShare float64 `json:"demand_share"`
	GeraiCount  int     `json:"gerai_count"`
	IsMissing   bool    `json:"is_missing"`
}

// StationPeak is the busiest observation point of the station on its peak
// slot — it carries the station's F/E/C/V character (the axis Persona 2
// compares "dwell / transit" vs "pass-through" on).
type StationPeak struct {
	PointLabel string     `json:"point_label"`
	TimeSlot   string     `json:"time_slot"`
	F          float64    `json:"f"`
	E          float64    `json:"e"`
	C          float64    `json:"c"`
	V          float64    `json:"v"`
	Gap        MoneyRange `json:"gap"`
}

// StationSummaryResponse is one station's rollup. `Basis` says whether the
// range is a real station-scoped Monte Carlo run ("monte-carlo-simpul") or a
// stopgap aggregate of point figures ("agregat-titik") — surfaced so a summed
// number can't pass for a simulated one.
type StationSummaryResponse struct {
	StationID    string                `json:"station_id"`
	StationName  string                `json:"station_name"`
	Typology     string                `json:"typology"` // stations.area_type
	DayType      string                `json:"day_type"`
	PintuDicacah int                   `json:"pintu_dicacah"`
	PintuDitahan int                   `json:"pintu_ditahan"`
	Potensi      MoneyRange            `json:"potensi"`
	Tertangkap   MoneyRange            `json:"tertangkap"`
	Gap          MoneyRange            `json:"gap"`
	CaptureRate  *float64              `json:"capture_rate"`
	Confidence   *ConfidenceBand       `json:"confidence"`
	StrukTerbaca int                   `json:"struk_terbaca"`
	Peak         *StationPeak          `json:"peak"`
	Composition  []CategoryComposition `json:"composition"`
	Basis        string                `json:"basis"`
	ComputedAt   string                `json:"computed_at"`
}
