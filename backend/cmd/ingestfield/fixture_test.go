package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/list-pandora/isi-stasiun-backend/internal/survey"
)

var testStations = map[string]station{
	"Manggarai": {ID: "a0000000-0000-4000-8000-000000000001", Code: "MRI"},
	"Sudirman":  {ID: "a0000000-0000-4000-8000-000000000002", Code: "SUD"},
}

func TestBuildEntryConversionsMapsAMeasuredRow(t *testing.T) {
	subs, skips, err := buildEntryConversions([]entryRow{{
		ID: "man_senin_pagi_indomaret", Station: "Manggarai", Day: "Senin", Slot: "pagi",
		Time: "08.00", Gerai: "Indomaret", Category: "ritel_kemasan",
		Lewat: 5, Masuk: 2, BeliEst: 2, Source: "measured",
	}}, testStations, "tim-survei")
	require.NoError(t, err)
	require.Empty(t, skips)
	require.Len(t, subs, 1)

	body := subs[0].Body.(survey.CreateEntryConversionRequest)
	assert.Equal(t, "field-man_senin_pagi_indomaret", subs[0].IdempotencyKey)
	assert.Equal(t, testStations["Manggarai"].ID, body.StationID)
	assert.Equal(t, "2026-08-31T08:00:00+07:00", body.ObservedAt)
	assert.Equal(t, "morning", body.TimeSlot)
	assert.Equal(t, 5, body.PassersBy)
	assert.Equal(t, 2, body.EnteredCount)
	assert.Equal(t, 2, body.CompletedPurchaseCount)
	assert.Equal(t, geraiID("MRI", "Indomaret"), body.GeraiID)
}

// The synthetic midday block must never enter the survey tables as measured
// data — that is the whole point of the provenance flags in the fixture.
func TestBuildEntryConversionsSkipsSyntheticAndUnattributedRows(t *testing.T) {
	subs, skips, err := buildEntryConversions([]entryRow{
		{ID: "man_x_siang_indomaret", Station: "Manggarai", Slot: "siang", Time: "~12.00 (mock)", Source: "mock"},
		{ID: "x_group1", Station: "", Gerai: "Lawson", Source: "measured"},
		{ID: "sud_kios_atas", Station: "Sudirman", Gerai: "Kios atas", Source: "measured"},
	}, testStations, "tim-survei")
	require.NoError(t, err)
	assert.Empty(t, subs)
	require.Len(t, skips, 3)
	assert.Equal(t, "man_x_siang_indomaret", skips[0].ID)
	assert.Equal(t, "x_group1", skips[1].ID)
	assert.Equal(t, "sud_kios_atas", skips[2].ID)
}

// Sudirman morning carries a mocked `lewat` (lewat_source "mock_rush_hour")
// but a real `masuk`, so the row is ingested; only fully synthetic rows drop.
func TestBuildEntryConversionsKeepsPartialMockRows(t *testing.T) {
	subs, skips, err := buildEntryConversions([]entryRow{{
		ID: "sud_senin_pagi_roti_o", Station: "Sudirman", Day: "Senin", Slot: "pagi",
		Time: "08.15-08.30", Gerai: "Roti O", Category: "makanan_minuman",
		Lewat: 324, Masuk: 5, BeliEst: 5, LewatSource: "mock_rush_hour", Source: "partial_mock",
	}}, testStations, "tim-survei")
	require.NoError(t, err)
	require.Empty(t, skips)
	require.Len(t, subs, 1)

	body := subs[0].Body.(survey.CreateEntryConversionRequest)
	assert.Equal(t, "2026-08-31T08:15:00+07:00", body.ObservedAt, "range clock uses the block start")
}

func TestBuildEntryConversionsMapsSelasaEvening(t *testing.T) {
	subs, _, err := buildEntryConversions([]entryRow{{
		ID: "man_selasa_sore_cfc", Station: "Manggarai", Day: "Selasa", Slot: "sore",
		Time: "17.30", Gerai: "CFC", Category: "makanan_minuman", Source: "measured",
	}}, testStations, "tim-survei")
	require.NoError(t, err)
	require.Len(t, subs, 1)

	body := subs[0].Body.(survey.CreateEntryConversionRequest)
	assert.Equal(t, "2026-09-01T17:30:00+07:00", body.ObservedAt)
	assert.Equal(t, "evening", body.TimeSlot)
}

func TestBuildFlowObservationsSplitsDirections(t *testing.T) {
	entranceIDs := map[string]string{"MRI:A": "e0000000-0000-4000-8000-000000000001"}
	subs, skips, err := buildFlowObservations([]flowRow{
		{ID: "flow_1", Station: "Manggarai", Pintu: "A", Masuk: 44, Keluar: 32, Source: "measured"},
		{ID: "flow_2", Station: "Manggarai", Pintu: "B", Masuk: 18, Keluar: 9, Source: "measured"},
	}, testStations, entranceIDs, "tim-survei")
	require.NoError(t, err)
	require.Len(t, subs, 2, "flow_2 has no entrance mapping in this table")
	require.Len(t, skips, 1)
	assert.Equal(t, "flow_2", skips[0].ID)

	in := subs[0].Body.(survey.CreateFlowObservationRequest)
	out := subs[1].Body.(survey.CreateFlowObservationRequest)
	assert.Equal(t, "in", in.Direction)
	assert.Equal(t, 44, in.PedestrianCount)
	assert.Equal(t, "out", out.Direction)
	assert.Equal(t, 32, out.PedestrianCount)
	assert.Equal(t, "2026-08-31T08:00:00+07:00", in.ObservedAt)
	assert.NotEqual(t, subs[0].IdempotencyKey, subs[1].IdempotencyKey)
}

// Re-running the ingest must address the same gerai rows, otherwise every run
// would fan the same store out into new ids and break the pipeline rollups.
func TestGeraiIDIsDeterministicAndStationScoped(t *testing.T) {
	assert.Equal(t, geraiID("MRI", "Indomaret"), geraiID("MRI", " Indomaret "))
	assert.NotEqual(t, geraiID("MRI", "Indomaret"), geraiID("SUD", "Indomaret"))
	assert.Equal(t, "64d44986-a4a6-5082-82f1-e076e2d2973c", geraiID("MRI", "Indomaret"))
}

func TestParseClock(t *testing.T) {
	for input, want := range map[string]string{
		"08.00":       "08:00",
		"08.15-08.30": "08:15",
		"17.10":       "17:10",
	} {
		got, err := parseClock(input)
		require.NoError(t, err, input)
		assert.Equal(t, want, got, input)
	}

	_, err := parseClock("pagi")
	assert.Error(t, err)
}
