package copilot

import (
	"context"
	"strings"
	"testing"
)

func TestClassifyIntent(t *testing.T) {
	cases := []struct {
		query string
		want  Intent
	}{
		{"berapa kesenjangan belanja di pintu utama", IntentSpendingGap},
		{"potensi pendapatan non-tiket stasiun ini", IntentSpendingGap},
		{"kategori usaha apa yang belum ada di dalam stasiun", IntentCategoryGap},
		{"gerai apa yang cocok dibuka di sini", IntentCategoryGap},
		{"harga sewa petak di sekitar sini kemahalan tidak", IntentRentFlow},
		{"kapan waktu terbaik bikin bazar umkm", IntentEventPotential},
		{"seberapa yakin angka ini, sampelnya cukup", IntentConfidence},
		{"pintu mana yang paling ramai", IntentFlow},
		{"halo", IntentUnknown},

		// A category question that mentions "potensi" must stay a category
		// question — this is why the broad money intent is matched last.
		{"kategori apa yang potensinya paling besar", IntentCategoryGap},

		// A bare slice with no topic word still reads as a money question,
		// which is the map's default view.
		{"apotek", IntentSpendingGap},
		{"sore", IntentSpendingGap},
	}

	for _, tc := range cases {
		if got := Classify(tc.query, "").Intent; got != tc.want {
			t.Errorf("Classify(%q) = %q, want %q", tc.query, got, tc.want)
		}
	}
}

func TestClassifyExtractsFilters(t *testing.T) {
	cases := []struct {
		query        string
		wantCategory string
		wantSlot     string
	}{
		{"kesenjangan kopi pada jam sibuk sore", "makanan_minuman", "evening"},
		{"apotek di pagi hari", "apotek_kesehatan", "morning"},
		{"minimarket malam", "ritel_kemasan", "night"},
		{"laundry", "jasa", ""},
		{"kesenjangan belanja total", "", ""},

		// Both axes fire on "makan siang" by design: one is the category, the
		// other the slot. They are independent dimensions of the same slice.
		{"gerai makan siang", "makanan_minuman", "midday"},
	}

	for _, tc := range cases {
		got := Classify(tc.query, "")
		if got.Category != tc.wantCategory {
			t.Errorf("Classify(%q).Category = %q, want %q", tc.query, got.Category, tc.wantCategory)
		}
		if got.TimeSlot != tc.wantSlot {
			t.Errorf("Classify(%q).TimeSlot = %q, want %q", tc.query, got.TimeSlot, tc.wantSlot)
		}
	}
}

// Category and slot keys must be the strings the database CHECK constraints
// accept, not the Indonesian labels the UI shows. A mismatch here would only
// surface as an empty result at query time.
func TestFilterKeysMatchBackendVocabulary(t *testing.T) {
	validCategories := map[string]bool{
		"makanan_minuman": true, "ritel_kemasan": true,
		"apotek_kesehatan": true, "jasa": true, "lainnya": true,
	}
	validSlots := map[string]bool{"morning": true, "midday": true, "evening": true, "night": true}

	for _, e := range categoryPhrases {
		if !validCategories[e.key] {
			t.Errorf("category key %q is not in the migration CHECK constraint", e.key)
		}
	}
	for _, e := range slotPhrases {
		if !validSlots[e.key] {
			t.Errorf("slot key %q is not in the migration CHECK constraint", e.key)
		}
	}
}

func TestClassifyIsDeterministic(t *testing.T) {
	const q = "kategori makan siang apotek sore mana yang belum ada"
	first := Classify(q, "st-1")
	for i := 0; i < 200; i++ {
		got := Classify(q, "st-1")
		if got.Intent != first.Intent || got.Category != first.Category || got.TimeSlot != first.TimeSlot {
			t.Fatalf("iteration %d differs: %+v vs %+v", i, got, first)
		}
	}
}

func TestSpatialFilterOmitsUnspecifiedKeys(t *testing.T) {
	if f := Classify("halo", "").SpatialFilter(); f != nil {
		t.Errorf("no station, category or slot: got %v, want nil", f)
	}

	f := Classify("apotek sore", "st-1").SpatialFilter()
	if f["station_id"] != "st-1" || f["category"] != "apotek_kesehatan" || f["time_slot"] != "evening" {
		t.Errorf("unexpected filter: %v", f)
	}
}

// The endpoint must answer usefully with no AI service behind it — section 3.6
// forbids the map being blocked by a copilot failure.
func TestQueryFallsBackWhenAIServiceIsDown(t *testing.T) {
	svc := NewService(NewClient("http://127.0.0.1:1", 1)) // connection refused

	resp, err := svc.Query(context.Background(), QueryRequest{
		Query:     "kategori apa yang belum ada di stasiun ini",
		StationID: "st-1",
	})
	if err != nil {
		t.Fatalf("Query returned an error instead of degrading: %v", err)
	}
	if resp.Answer == "" {
		t.Error("empty answer")
	}
	// The reply is still served with 200, so it has to say on its face that no
	// model was involved. Priyapta's TestServiceFallbackDoesNotExposeUpstreamError
	// covers the other half: the upstream error must not leak into it.
	if !strings.Contains(resp.Answer, "tidak tersedia") {
		t.Errorf("degraded answer does not admit the model was unavailable: %q", resp.Answer)
	}
	if len(resp.SuggestedLayers) == 0 {
		t.Error("no suggested layers — the map has nothing to act on")
	}
	if resp.SpatialFilter["station_id"] != "st-1" {
		t.Errorf("station scope lost: %v", resp.SpatialFilter)
	}
}

func TestMergeKeepsLocalFilterWhenModelReturnsNothing(t *testing.T) {
	local := &QueryResponse{
		Answer:          "lokal",
		SuggestedLayers: []string{"gap"},
		SpatialFilter:   map[string]interface{}{"category": "jasa"},
	}

	got := merge(local, &QueryResponse{Answer: "  "})
	if got.Answer != "lokal" {
		t.Errorf("blank model answer overwrote the local one: %q", got.Answer)
	}
	if len(got.SuggestedLayers) == 0 || got.SpatialFilter["category"] != "jasa" {
		t.Errorf("model stripped the locally derived filter: %+v", got)
	}

	got = merge(local, &QueryResponse{Answer: "model", SuggestedLayers: []string{"sewa"}})
	if got.Answer != "model" || got.SuggestedLayers[0] != "sewa" {
		t.Errorf("model output was not preferred: %+v", got)
	}
}
