package copilot

import (
	"fmt"
	"strings"
)

// Intent is what the user is actually asking about, expressed in the terms the
// map already uses. Classification happens here rather than in the LLM prompt
// so that a copilot answer is reproducible and the endpoint keeps working when
// the model is unreachable — section 3.6 requires the map never block on it.
type Intent string

const (
	IntentSpendingGap    Intent = "spending_gap"
	IntentCategoryGap    Intent = "category_gap"
	IntentRentFlow       Intent = "rent_flow_index"
	IntentEventPotential Intent = "event_potential"
	IntentConfidence     Intent = "confidence"
	IntentFlow           Intent = "flow"
	IntentUnknown        Intent = "unknown"
)

// Layer row keys the Peta panel understands (LAYER_ROWS in PetaScreen.tsx).
// Only gap, potensi and kepercayaan are wired to real map layers today; the
// rest are listed there as "belum ada data". Naming them anyway keeps the
// contract stable — the panel activates a row the moment it gains data, and
// the copilot does not need changing then.
const (
	layerGap         = "gap"
	layerPotensi     = "potensi"
	layerKepercayaan = "kepercayaan"
	layerKategori    = "kategori-hilang"
	layerArus        = "arus"
	layerSewa        = "sewa"
	layerEvent       = "event"
)

// Analysis is the structured reading of one query: what was asked, which
// slice of the data answers it, and which analytics endpoint holds it.
type Analysis struct {
	Intent    Intent
	Category  string // canonical backend key, empty if unspecified
	TimeSlot  string // morning|midday|evening|night, empty if unspecified
	StationID string
	Layers    []string
	Endpoint  string
}

// keyPhrases is an ordered lookup, not a map: map iteration order in Go is
// randomised, so a query matching two entries would classify differently
// between calls. Reproducibility is the whole reason classification lives here
// instead of in a prompt, so the order has to be fixed and readable.
type keyPhrases struct {
	key     string
	phrases []string
}

// Canonical category keys — the CHECK constraints in migrations 003/004/005
// enforce these exact strings, so the mapping is one-way and deliberate.
// "makan siang" would match both makanan_minuman and the midday slot; that is
// intended, they are different axes and both get applied.
var categoryPhrases = []keyPhrases{
	{"makanan_minuman", []string{"makanan", "minuman", "makan", "minum", "kopi", "kuliner", "f&b", "fnb", "resto", "kafe", "cafe"}},
	{"ritel_kemasan", []string{"ritel", "retail", "kemasan", "minimarket", "swalayan", "convenience", "oleh-oleh"}},
	{"apotek_kesehatan", []string{"apotek", "obat", "kesehatan", "farmasi", "klinik"}},
	{"jasa", []string{"jasa", "layanan", "servis", "laundry", "barbershop", "salon"}},
	{"lainnya", []string{"lainnya", "lain-lain"}},
}

// Slot keys follow the backend CHECK constraint, not the Indonesian labels the
// UI shows. Both spellings are accepted on input because users type either.
// Bare hour numbers are matched last so "kopi 17 ribu" does not read as sore
// before a word like "sore" has had its chance.
var slotPhrases = []keyPhrases{
	{"morning", []string{"pagi", "morning", "berangkat", "jam 6", "jam 7", "jam 8", "06.", "07.", "08."}},
	{"midday", []string{"siang", "midday", "makan siang", "jam 11", "jam 12", "jam 13", "11.", "12.", "13."}},
	{"evening", []string{"sore", "evening", "pulang kerja", "pulang", "jam 16", "jam 17", "jam 18", "16.", "17.", "18."}},
	{"night", []string{"malam", "night", "jam 19", "jam 20", "jam 21", "19.", "20.", "21."}},
}

var intentPhrases = []struct {
	intent   Intent
	phrases  []string
	layers   []string
	endpoint string
}{
	{
		intent:   IntentCategoryGap,
		phrases:  []string{"kategori", "usaha apa", "gerai apa", "jenis usaha", "belum ada", "kategori hilang", "yang kurang", "cocok dibuka", "peluang usaha", "tenant"},
		layers:   []string{layerKategori, layerGap},
		endpoint: "/api/v1/analytics/category-gap",
	},
	{
		intent:   IntentRentFlow,
		phrases:  []string{"sewa", "harga sewa", "petak", "mahal", "murah", "kemahalan", "kontrak", "tarif ruang"},
		layers:   []string{layerSewa},
		endpoint: "/api/v1/analytics/rent-flow-index",
	},
	{
		intent:   IntentEventPotential,
		phrases:  []string{"event", "acara", "bazar", "pop-up", "popup", "aktivasi", "tenant sementara", "kapan ramai untuk", "pameran"},
		layers:   []string{layerEvent, layerArus},
		endpoint: "/api/v1/analytics/event-potential",
	},
	{
		intent:   IntentConfidence,
		phrases:  []string{"sampel", "sample", "kepercayaan", "confidence", "seberapa yakin", "akurat", "data tipis", "bisa dipercaya", "keandalan"},
		layers:   []string{layerKepercayaan},
		endpoint: "/api/v1/confidence-layer",
	},
	{
		intent:   IntentFlow,
		phrases:  []string{"arus", "pejalan", "ramai", "orang lewat", "pintu mana", "trafik", "traffic", "lalu lalang", "penumpang"},
		layers:   []string{layerArus, layerGap},
		endpoint: "/api/v1/analytics/spending-gap",
	},
	{
		intent:   IntentSpendingGap,
		phrases:  []string{"kesenjangan", "gap", "potensi", "belanja", "peluang", "pendapatan", "tertangkap", "rupiah", "omzet", "non-tiket"},
		layers:   []string{layerGap, layerPotensi},
		endpoint: "/api/v1/analytics/spending-gap",
	},
}

// Classify reads a free-text query into an Analysis.
//
// Order matters: the specific intents are tested before spending_gap, because
// "kategori apa yang potensinya besar" is a category question that happens to
// contain the word "potensi". The broadest intent is therefore last, acting as
// the default for anything that mentions money at all.
func Classify(query, stationID string) Analysis {
	q := strings.ToLower(query)

	a := Analysis{
		Intent:    IntentUnknown,
		StationID: stationID,
		Category:  matchCategory(q),
		TimeSlot:  matchSlot(q),
		Layers:    []string{layerGap},
		Endpoint:  "/api/v1/analytics/spending-gap",
	}

	for _, cand := range intentPhrases {
		if containsAny(q, cand.phrases) {
			a.Intent = cand.intent
			a.Layers = cand.layers
			a.Endpoint = cand.endpoint
			break
		}
	}

	// A bare category or time mention with no other signal is still a question
	// about money at that slice — the map's default reading.
	if a.Intent == IntentUnknown && (a.Category != "" || a.TimeSlot != "") {
		a.Intent = IntentSpendingGap
		a.Layers = []string{layerGap, layerPotensi}
	}

	return a
}

// SpatialFilter renders the analysis as the filter payload the map applies to
// itself. Keys are omitted rather than sent empty so the frontend can tell
// "not mentioned" from "explicitly all".
func (a Analysis) SpatialFilter() map[string]interface{} {
	f := map[string]interface{}{}
	if a.StationID != "" {
		f["station_id"] = a.StationID
	}
	if a.Category != "" {
		f["category"] = a.Category
	}
	if a.TimeSlot != "" {
		f["time_slot"] = a.TimeSlot
	}
	if len(f) == 0 {
		return nil
	}
	return f
}

// Answer is the deterministic reply, used when the AI service is unavailable.
// It states what the copilot understood and where the number lives, and never
// asserts a figure it has not read — inventing one would break the promise
// that every number on screen is traceable.
func (a Analysis) Answer() string {
	var b strings.Builder

	switch a.Intent {
	case IntentSpendingGap:
		b.WriteString("Pertanyaan ini soal kesenjangan belanja — selisih antara potensi belanja komuter dan yang tertangkap gerai di dalam stasiun.")
	case IntentCategoryGap:
		b.WriteString("Pertanyaan ini soal kategori hilang — kategori usaha yang permintaannya terbaca di kawasan tetapi belum tersedia di dalam stasiun.")
	case IntentRentFlow:
		b.WriteString("Pertanyaan ini soal indeks sewa per arus. Indeks ini deskriptif dan berlaku untuk properti di kawasan sekitar stasiun, bukan penilaian harga wajar untuk petak di dalam stasiun.")
	case IntentEventPotential:
		b.WriteString("Pertanyaan ini soal potensi event dan aktivasi — disusun dari luas ruang yang bisa diaktivasi, jam ruang tidak terpakai, dan arus puncak.")
	case IntentConfidence:
		b.WriteString("Pertanyaan ini soal kepercayaan data. Kawasan bersampel tipis tidak diberi estimasi dan tidak boleh dibaca sebagai aman maupun bermasalah.")
	case IntentFlow:
		b.WriteString("Pertanyaan ini soal arus pejalan kaki (variabel F) per pintu dan per slot waktu.")
	default:
		b.WriteString("Belum jelas bagian mana yang ditanyakan. Coba sebut kesenjangan belanja, kategori usaha, indeks sewa, potensi event, arus pintu, atau kepercayaan data.")
	}

	if a.Category != "" {
		fmt.Fprintf(&b, " Disaring ke kategori %s.", categoryLabel(a.Category))
	}
	if a.TimeSlot != "" {
		fmt.Fprintf(&b, " Pada slot %s (%s).", slotLabel(a.TimeSlot), slotHours(a.TimeSlot))
	}
	if a.Intent != IntentUnknown {
		fmt.Fprintf(&b, " Angkanya ada di %s, dan seluruhnya disajikan sebagai rentang P10-P90.", a.Endpoint)
	}

	return b.String()
}

func categoryLabel(key string) string {
	switch key {
	case "makanan_minuman":
		return "makanan & minuman"
	case "ritel_kemasan":
		return "ritel kemasan"
	case "apotek_kesehatan":
		return "apotek & kesehatan"
	case "jasa":
		return "jasa"
	default:
		return "lainnya"
	}
}

func slotLabel(key string) string {
	switch key {
	case "morning":
		return "pagi"
	case "midday":
		return "siang"
	case "evening":
		return "sore"
	default:
		return "malam"
	}
}

// slotHours reports the range actually counted, which is not the range the
// button label shows — the "16-19" button counts 16.00-18.59 (ROADMAP §7).
func slotHours(key string) string {
	switch key {
	case "morning":
		return "06.00-08.59"
	case "midday":
		return "11.00-13.59"
	case "evening":
		return "16.00-18.59"
	default:
		return "19.00-20.59"
	}
}

func matchCategory(q string) string { return matchFirst(q, categoryPhrases) }

func matchSlot(q string) string { return matchFirst(q, slotPhrases) }

func matchFirst(q string, table []keyPhrases) string {
	for _, e := range table {
		if containsAny(q, e.phrases) {
			return e.key
		}
	}
	return ""
}

func containsAny(q string, phrases []string) bool {
	for _, p := range phrases {
		if strings.Contains(q, p) {
			return true
		}
	}
	return false
}
