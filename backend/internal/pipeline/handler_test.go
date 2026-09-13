package pipeline

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"

	"github.com/list-pandora/isi-stasiun-backend/internal/response"
	"github.com/list-pandora/isi-stasiun-backend/internal/summary"
)

type stubService struct {
	strukCalled, propertiCalled, geraiCalled, monteCarloCalled bool
	stationSummary                                             *summary.StationSummaryWrite
}

func (s *stubService) IngestStrukExtraction(context.Context, StrukExtractionCallback) error {
	s.strukCalled = true
	return nil
}
func (s *stubService) IngestPropertiExtraction(context.Context, PropertiExtractionCallback) error {
	s.propertiCalled = true
	return nil
}
func (s *stubService) IngestGeraiClassification(context.Context, GeraiClassificationCallback) error {
	s.geraiCalled = true
	return nil
}
func (s *stubService) IngestMonteCarloResult(context.Context, MonteCarloResultCallback) error {
	s.monteCarloCalled = true
	return nil
}

func (s *stubService) IngestStationSummary(_ context.Context, in summary.StationSummaryWrite) error {
	s.stationSummary = &in
	return nil
}

func newApp(svc pipelineService) *fiber.App {
	app := fiber.New()
	NewHandler(svc).RegisterRoutes(app)
	return app
}

func post(t *testing.T, app *fiber.App, path, body string) (int, response.Envelope) {
	t.Helper()
	req := httptest.NewRequest("POST", path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req, -1)
	require.NoError(t, err)
	var env response.Envelope
	_ = json.NewDecoder(res.Body).Decode(&env)
	return res.StatusCode, env
}

const validStruk = `{
	"job_id":"job1","source_ref":"r2/key.jpg",
	"station_id":"11111111-1111-1111-1111-111111111111",
	"category":"makanan_minuman","final_amount":25000,
	"payment_method":"qris","transacted_at":"2026-08-29T07:00:00Z",
	"confidence":0.9,"is_ambiguous":false
}`

func TestStrukExtractionRejectsUnknownCategory(t *testing.T) {
	svc := &stubService{}
	body := strings.Replace(validStruk, `"category":"makanan_minuman"`, `"category":"fashion"`, 1)
	code, _ := post(t, newApp(svc), "/pipeline/extractions/struk", body)

	require.Equal(t, fiber.StatusBadRequest, code)
	require.False(t, svc.strukCalled)
}

func TestStrukExtractionAcceptsValidPayload(t *testing.T) {
	svc := &stubService{}
	code, env := post(t, newApp(svc), "/pipeline/extractions/struk", validStruk)

	require.Equal(t, fiber.StatusOK, code)
	require.True(t, svc.strukCalled)
	require.Equal(t, "ingested", env.Data.(map[string]any)["status"])
}

func TestStrukExtractionRejectsZeroAmountWhenNotAmbiguous(t *testing.T) {
	svc := &stubService{}
	body := strings.Replace(validStruk, `"final_amount":25000`, `"final_amount":0`, 1)
	code, _ := post(t, newApp(svc), "/pipeline/extractions/struk", body)

	require.Equal(t, fiber.StatusBadRequest, code)
	require.False(t, svc.strukCalled)
}

func TestStrukExtractionAcceptsZeroAmountWhenAmbiguous(t *testing.T) {
	svc := &stubService{}
	body := strings.NewReplacer(`"final_amount":25000`, `"final_amount":0`, `"is_ambiguous":false`, `"is_ambiguous":true`).Replace(validStruk)
	code, _ := post(t, newApp(svc), "/pipeline/extractions/struk", body)

	require.Equal(t, fiber.StatusOK, code)
	require.True(t, svc.strukCalled)
}

func TestStrukExtractionRejectsMalformedPhotoURL(t *testing.T) {
	svc := &stubService{}
	body := strings.Replace(validStruk, `"is_ambiguous":false`, `"is_ambiguous":false,"photo_url":"not a url"`, 1)
	code, _ := post(t, newApp(svc), "/pipeline/extractions/struk", body)

	require.Equal(t, fiber.StatusBadRequest, code)
	require.False(t, svc.strukCalled)
}

func TestGeraiClassificationRejectsBadVisibility(t *testing.T) {
	svc := &stubService{}
	body := `{
		"job_id":"j","source_ref":"k","station_id":"11111111-1111-1111-1111-111111111111",
		"gerai_id":"22222222-2222-2222-2222-222222222222",
		"category":"jasa","visibility":"sideways","confidence":0.5
	}`
	code, _ := post(t, newApp(svc), "/pipeline/extractions/gerai", body)

	require.Equal(t, fiber.StatusBadRequest, code)
	require.False(t, svc.geraiCalled)
}

func TestMonteCarloRejectsPartialRun(t *testing.T) {
	svc := &stubService{}
	body := `{
		"job_id":"j","station_id":"11111111-1111-1111-1111-111111111111",
		"time_slot":"morning","potential_low_p10":1,"potential_high_p90":2,
		"captured_low_p10":0.5,"captured_high_p90":1,"iterations":500
	}`
	code, _ := post(t, newApp(svc), "/pipeline/simulations/monte-carlo", body)

	require.Equal(t, fiber.StatusBadRequest, code)
	require.False(t, svc.monteCarloCalled)
}

func TestMonteCarloAcceptsFullRun(t *testing.T) {
	svc := &stubService{}
	body := `{
		"job_id":"j","station_id":"11111111-1111-1111-1111-111111111111",
		"time_slot":"morning","potential_low_p10":1,"potential_high_p90":2,
		"captured_low_p10":0.5,"captured_high_p90":1,"iterations":10000
	}`
	code, _ := post(t, newApp(svc), "/pipeline/simulations/monte-carlo", body)

	require.Equal(t, fiber.StatusOK, code)
	require.True(t, svc.monteCarloCalled)
}

const validStationSummary = `{
	"job_id":"job-summary-1",
	"station_id":"11111111-1111-1111-1111-111111111111",
	"day_type":"weekday","iterations":10000,
	"potensi":{"p10":1000,"p50":1500,"p90":2000},
	"tertangkap":{"p10":200,"p50":400,"p90":600},
	"gap":{"p10":600,"p50":1100,"p90":1600},
	"capture_rate":0.2667,
	"confidence":{"min":0.3,"max":0.8},
	"struk_terbaca":12,"pintu_dicacah":3,"pintu_ditahan":1,
	"peak":{"point_label":"Pintu Bawah","time_slot":"evening",
	        "f":120,"e":0.03,"c":0.95,"v":22000,
	        "gap":{"p10":300,"p50":500,"p90":700}},
	"composition":[{"category":"makanan_minuman","demand_share":0.5,"gerai_count":4,"is_missing":false},
	               {"category":"apotek_kesehatan","demand_share":0.2,"gerai_count":0,"is_missing":true}],
	"basis":"monte-carlo-simpul"
}`

func TestStationSummaryPersistsTheWholePayload(t *testing.T) {
	svc := &stubService{}
	code, _ := post(t, newApp(svc), "/pipeline/simulations/station-summary", validStationSummary)

	require.Equal(t, fiber.StatusOK, code)
	require.NotNil(t, svc.stationSummary)
	got := *svc.stationSummary
	require.Equal(t, 1500.0, got.Potensi.P50)
	require.Equal(t, 1100.0, got.Gap.P50)
	require.NotNil(t, got.CaptureRate)
	require.NotNil(t, got.Confidence)
	require.NotNil(t, got.Peak)
	require.Equal(t, "evening", got.Peak.TimeSlot)
	require.Len(t, got.Composition, 2)
}

// A summed-up "agregat-titik" number must never be able to arrive labelled as
// a simulated one, and vice versa — the frontend shows this stamp verbatim.
func TestStationSummaryRejectsUnknownBasis(t *testing.T) {
	svc := &stubService{}
	body := strings.Replace(validStationSummary, `"basis":"monte-carlo-simpul"`, `"basis":"tebakan"`, 1)
	code, _ := post(t, newApp(svc), "/pipeline/simulations/station-summary", body)

	require.Equal(t, fiber.StatusBadRequest, code)
	require.Nil(t, svc.stationSummary)
}

// P10 <= P50 <= P90 is what makes the published range readable as a range.
func TestStationSummaryRejectsUnorderedPercentiles(t *testing.T) {
	svc := &stubService{}
	body := strings.Replace(validStationSummary, `"gap":{"p10":600,"p50":1100,"p90":1600}`, `"gap":{"p10":1600,"p50":1100,"p90":600}`, 1)
	code, env := post(t, newApp(svc), "/pipeline/simulations/station-summary", body)

	require.Equal(t, fiber.StatusBadRequest, code)
	require.Contains(t, env.Message, "gap")
	require.Nil(t, svc.stationSummary)
}

// Methodology 3.3 asks for a full 10,000-iteration run; a truncated one would
// publish a narrower range than the model actually supports.
func TestStationSummaryRejectsTruncatedRun(t *testing.T) {
	svc := &stubService{}
	body := strings.Replace(validStationSummary, `"iterations":10000`, `"iterations":500`, 1)
	code, _ := post(t, newApp(svc), "/pipeline/simulations/station-summary", body)

	require.Equal(t, fiber.StatusBadRequest, code)
	require.Nil(t, svc.stationSummary)
}

// Captured above potential means the two sides were fed flows measured on
// different bases, and the published "gap" would come out negative — the exact
// kind of number this project promises never to show.
func TestStationSummaryRejectsCapturedAbovePotential(t *testing.T) {
	svc := &stubService{}
	body := strings.NewReplacer(
		`"tertangkap":{"p10":200,"p50":400,"p90":600}`, `"tertangkap":{"p10":1800,"p50":2000,"p90":2200}`,
		`"gap":{"p10":600,"p50":1100,"p90":1600}`, `"gap":{"p10":-700,"p50":-500,"p90":-200}`,
		`"capture_rate":0.2667`, `"capture_rate":1.33`,
	).Replace(validStationSummary)
	code, _ := post(t, newApp(svc), "/pipeline/simulations/station-summary", body)

	require.Equal(t, fiber.StatusBadRequest, code)
	require.Nil(t, svc.stationSummary)
}
