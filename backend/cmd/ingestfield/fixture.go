package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/list-pandora/isi-stasiun-backend/internal/survey"
)

// Transform layer for the field fixtures published in
// `isistasiun-ai/data/source/field/`. Those files are shaped for the AI
// service (station *names*, "Senin"/"pagi", a pre-computed `beli_est`), while
// `/survey/*` wants UUIDs, RFC3339 timestamps and the baku slot enum — so the
// mapping lives here, in one place, with every judgement call written down.

// surveyDates resolves the fixture weekday labels to the actual survey days.
// Confirmed by the survey team on 2026-09-12; the fixtures carry no dates, and
// `Context/01-DATA-SOURCES-AND-SURVEY.md` (line 71) still records "1 hari
// total", which the two-day fixture contradicts. Do not guess these.
var surveyDates = map[string]string{
	"Senin":  "2026-08-31",
	"Selasa": "2026-09-01",
}

// slotToTimeSlot maps the fixture Indonesian slot labels onto the baku enum
// used by the DB CHECK constraint.
var slotToTimeSlot = map[string]string{
	"pagi":  "morning",
	"siang": "midday",
	"sore":  "evening",
	"malam": "night",
}

// wib is the survey local zone. The fixture clocks ("08.00", "17.30") are
// wall-clock Jakarta time, so they are only unambiguous with this offset.
var wib = time.FixedZone("WIB", 7*60*60)

// geraiNamespace anchors the deterministic gerai ids. There is no `gerai`
// table in the schema — `entry_conversion_observations.gerai_id` is a bare
// UUID with no foreign key (migration 000003) — so a stable UUIDv5 derived
// from station code + gerai label gives every re-run the same id without
// adding a migration in an area owned by someone else.
var geraiNamespace = uuid.NewSHA1(uuid.NameSpaceURL, []byte("https://isistasiun.id/gerai"))

// geraiID is stable across runs, machines and fixture regenerations.
func geraiID(stationCode, label string) string {
	key := stationCode + ":" + strings.TrimSpace(label)
	return uuid.NewSHA1(geraiNamespace, []byte(key)).String()
}

type entryFixture struct {
	Data []entryRow `json:"data"`
}

type entryRow struct {
	ID          string `json:"id"`
	Station     string `json:"station"`
	Day         string `json:"day"`
	Slot        string `json:"slot"`
	Time        string `json:"time"`
	Gerai       string `json:"gerai"`
	Category    string `json:"category"`
	Lewat       int    `json:"lewat"`
	Masuk       int    `json:"masuk"`
	BeliEst     int    `json:"beli_est"`
	LewatSource string `json:"lewat_source"`
	Source      string `json:"source"`
	Note        string `json:"note"`
}

type flowFixture struct {
	Data []flowRow `json:"data"`
}

type flowRow struct {
	ID      string `json:"id"`
	Station string `json:"station"`
	Pintu   string `json:"pintu"`
	Keluar  int    `json:"keluar"`
	Masuk   int    `json:"masuk"`
	Source  string `json:"source"`
}

// station carries what the transform needs from the `stations` table.
type station struct {
	ID   string
	Code string
}

// skip records a fixture row that is deliberately not ingested, so the run log
// accounts for every one of the 47 rows instead of silently dropping any.
type skip struct {
	ID     string
	Reason string
}

// submission is one ready-to-POST observation plus the idempotency key that
// makes a re-run a no-op.
type submission struct {
	Path           string
	IdempotencyKey string
	Body           any
}

// buildEntryConversions turns fixture rows into
// `/survey/entry-conversion-observations` payloads. Rows flagged
// `source: "mock"` are the synthetic midday block — SOURCE.md requires them to
// be marked "estimasi (tidak disurvei)", so feeding them in as measured survey
// rows would put synthetic numbers into the source of truth. They are skipped,
// as are rows whose station or timing is unknown.
func buildEntryConversions(rows []entryRow, stations map[string]station, surveyorID string) ([]submission, []skip, error) {
	var out []submission
	var skipped []skip

	for _, row := range rows {
		switch {
		case row.Source == "mock":
			skipped = append(skipped, skip{row.ID, "baris siang sintetis (source=mock), bukan pengamatan lapangan"})
			continue
		case strings.TrimSpace(row.Station) == "":
			skipped = append(skipped, skip{row.ID, "stasiun belum dikonfirmasi di fixture"})
			continue
		case row.Day == "" || row.Slot == "" || row.Time == "":
			skipped = append(skipped, skip{row.ID, "tanpa hari/slot/jam, observed_at tidak bisa ditentukan"})
			continue
		}

		st, ok := stations[row.Station]
		if !ok {
			return nil, nil, fmt.Errorf("baris %s: stasiun %q tidak ada di tabel stations", row.ID, row.Station)
		}
		observed, err := observedAt(row.Day, row.Time)
		if err != nil {
			return nil, nil, fmt.Errorf("baris %s: %w", row.ID, err)
		}
		timeSlot, ok := slotToTimeSlot[row.Slot]
		if !ok {
			return nil, nil, fmt.Errorf("baris %s: slot %q tidak dikenal", row.ID, row.Slot)
		}

		out = append(out, submission{
			Path:           "/survey/entry-conversion-observations",
			IdempotencyKey: "field-" + row.ID,
			Body: survey.CreateEntryConversionRequest{
				StationID:  st.ID,
				GeraiID:    geraiID(st.Code, row.Gerai),
				Category:   row.Category,
				TimeSlot:   timeSlot,
				ObservedAt: observed,
				// The fixture holds one measured block per gerai per slot, not
				// the two the protocol allows, so everything lands in block 1.
				BlockNumber:  1,
				PassersBy:    row.Lewat,
				EnteredCount: row.Masuk,
				// `beli_est` is already round(0.95 * masuk) — the same C the
				// pipeline now holds constant (A1). The column stays populated
				// for QA/transparency even though it no longer drives C.
				CompletedPurchaseCount: row.BeliEst,
				SurveyorID:             surveyorID,
			},
		})
	}
	return out, skipped, nil
}

// entranceSpec pins each fixture door label to a real entrance row.
//
// The fixture names doors "A"/"B"/"atas" and carries no coordinates for them.
// The A-is-bawah / B-is-atas reading for Manggarai was given by the survey
// team on 2026-09-12. The seeded demo entrances ("Pintu Utama", "Koridor
// Transit ...") are a different, invented set and are deliberately not
// reused.
//
// Corrected 2026-09-13: the original "SUD"/"atas" coordinate was mislabeled —
// it actually sits on the main lower entrance, not the upper one. Confirmed
// by the survey team via re-measurement; all coordinates below (Manggarai and
// Sudirman) come from that re-measurement, not the original field survey.
// The fixture's "atas" pintu code is kept as-is (it's the join key against
// flow-observations.json / entry-conversion.json) even though the corrected
// label is "Pintu Bawah Utama" — renaming the code would require re-keying
// the historical flow rows too, which were never split by door in the first
// place.
type entranceSpec struct {
	StationCode string
	Pintu       string
	Label       string
	Lat         float64
	Lon         float64
}

var fieldEntrances = []entranceSpec{
	{StationCode: "MRI", Pintu: "A", Label: "Pintu Bawah", Lat: -6.209898745923517, Lon: 106.85021581782942},
	{StationCode: "MRI", Pintu: "B", Label: "Pintu Atas", Lat: -6.210112908837247, Lon: 106.8492864593219},
	{StationCode: "SUD", Pintu: "atas", Label: "Pintu Bawah Utama", Lat: -6.202267977330985, Lon: 106.82318166742444},
	{StationCode: "SUD", Pintu: "bawah_belakang", Label: "Pintu Bawah Belakang", Lat: -6.20262664463407, Lon: 106.8246451141926},
	{StationCode: "SUD", Pintu: "atas_asli", Label: "Pintu Atas", Lat: -6.202413019184441, Lon: 106.82357353948973},
}

// flowClock is the start of the measured morning block per station, from the
// survey team: Manggarai 08.00-08.20, Sudirman 08.30-08.40.
var flowClock = map[string]string{"MRI": "08.00", "SUD": "08.30"}

// flowDay — the fixture rows carry no day, so they are taken as the first
// survey day.
const flowDay = "Senin"

// buildFlowObservations splits each fixture row into the two directional rows
// the table models: `masuk` becomes direction "in", `keluar` becomes "out".
func buildFlowObservations(rows []flowRow, stations map[string]station, entranceIDs map[string]string, surveyorID string) ([]submission, []skip, error) {
	var out []submission
	var skipped []skip

	for _, row := range rows {
		st, ok := stations[row.Station]
		if !ok {
			return nil, nil, fmt.Errorf("baris %s: stasiun %q tidak ada di tabel stations", row.ID, row.Station)
		}
		entranceID, ok := entranceIDs[st.Code+":"+row.Pintu]
		if !ok {
			skipped = append(skipped, skip{row.ID, fmt.Sprintf("pintu %q di %s belum punya padanan entrance", row.Pintu, row.Station)})
			continue
		}
		observed, err := observedAt(flowDay, flowClock[st.Code])
		if err != nil {
			return nil, nil, fmt.Errorf("baris %s: %w", row.ID, err)
		}

		for _, dir := range []struct {
			name  string
			count int
		}{{"in", row.Masuk}, {"out", row.Keluar}} {
			out = append(out, submission{
				Path:           "/survey/flow-observations",
				IdempotencyKey: "field-" + row.ID + "-" + dir.name,
				Body: survey.CreateFlowObservationRequest{
					StationID:   st.ID,
					EntranceID:  entranceID,
					TimeSlot:    "morning",
					ObservedAt:  observed,
					BlockNumber: 1,
					// One measured count per door in the fixture, so both
					// directions share the same block.
					PedestrianCount: dir.count,
					Direction:       dir.name,
					SurveyorID:      surveyorID,
				},
			})
		}
	}
	return out, skipped, nil
}

// observedAt joins a fixture weekday label and wall clock into RFC3339 WIB.
// Clocks come as "08.00" or as a range like "08.15-08.30"; the range start is
// the block start, which is what observed_at means.
func observedAt(day, clock string) (string, error) {
	date, ok := surveyDates[day]
	if !ok {
		return "", fmt.Errorf("hari %q tidak punya tanggal survei", day)
	}
	hhmm, err := parseClock(clock)
	if err != nil {
		return "", err
	}
	ts, err := time.ParseInLocation("2006-01-02 15:04", date+" "+hhmm, wib)
	if err != nil {
		return "", fmt.Errorf("jam %q tidak bisa dibaca: %w", clock, err)
	}
	return ts.Format(time.RFC3339), nil
}

func parseClock(clock string) (string, error) {
	v := strings.TrimSpace(clock)
	if i := strings.IndexAny(v, "-–"); i > 0 {
		v = v[:i]
	}
	if i := strings.Index(v, " "); i > 0 {
		v = v[:i]
	}
	v = strings.TrimPrefix(v, "~")
	v = strings.ReplaceAll(v, ".", ":")
	if _, err := time.Parse("15:04", v); err != nil {
		return "", fmt.Errorf("jam %q tidak bisa dibaca", clock)
	}
	return v, nil
}
