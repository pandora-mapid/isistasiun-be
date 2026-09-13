package survey

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"

	"github.com/list-pandora/isi-stasiun-backend/internal/response"
)

type stubService struct {
	flowCalled       bool
	entryCalled      bool
	gotIdempotencyID string
	returnID         string
}

func (s *stubService) SubmitFlowObservation(_ context.Context, _ CreateFlowObservationRequest, key string) (string, error) {
	s.flowCalled = true
	s.gotIdempotencyID = key
	return s.returnID, nil
}

func (s *stubService) SubmitEntryConversion(_ context.Context, _ CreateEntryConversionRequest, key string) (string, error) {
	s.entryCalled = true
	s.gotIdempotencyID = key
	return s.returnID, nil
}

func (s *stubService) ListFlowObservations(context.Context, string, string, string) ([]FlowObservation, error) {
	return make([]FlowObservation, 0), nil
}

func newApp(svc surveyService) *fiber.App {
	app := fiber.New()
	NewHandler(svc).RegisterRoutes(app)
	return app
}

func post(t *testing.T, app *fiber.App, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("POST", path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	res, err := app.Test(req, -1)
	require.NoError(t, err)
	rec := httptest.NewRecorder()
	rec.Code = res.StatusCode
	_, _ = rec.Body.ReadFrom(res.Body)
	return rec
}

const validFlowBody = `{
	"station_id":"11111111-1111-1111-1111-111111111111",
	"entrance_id":"22222222-2222-2222-2222-222222222222",
	"time_slot":"morning","observed_at":"2026-08-29T07:00:00Z",
	"block_number":1,"pedestrian_count":42,"direction":"in","surveyor_id":"s1"
}`

func TestFlowObservationRejectsMissingFields(t *testing.T) {
	svc := &stubService{}
	rec := post(t, newApp(svc), "/survey/flow-observations", `{"time_slot":"morning"}`, nil)

	require.Equal(t, fiber.StatusBadRequest, rec.Code)
	require.False(t, svc.flowCalled, "service must not be called on invalid payload")
}

func TestFlowObservationRejectsBadTimeSlot(t *testing.T) {
	svc := &stubService{}
	body := strings.Replace(validFlowBody, `"time_slot":"morning"`, `"time_slot":"lunch"`, 1)
	rec := post(t, newApp(svc), "/survey/flow-observations", body, nil)

	require.Equal(t, fiber.StatusBadRequest, rec.Code)
	require.False(t, svc.flowCalled)
}

func TestFlowObservationAcceptsValidPayloadAndForwardsIdempotencyKey(t *testing.T) {
	svc := &stubService{returnID: "abc"}
	rec := post(t, newApp(svc), "/survey/flow-observations", validFlowBody,
		map[string]string{"Idempotency-Key": "key-123"})

	require.Equal(t, fiber.StatusCreated, rec.Code)
	require.True(t, svc.flowCalled)
	require.Equal(t, "key-123", svc.gotIdempotencyID)

	var body response.Envelope
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	require.True(t, body.Success)
	require.Equal(t, "abc", body.Data.(map[string]any)["id"])
}

func TestEntryConversionRejectsUnknownCategory(t *testing.T) {
	svc := &stubService{}
	body := `{
		"station_id":"11111111-1111-1111-1111-111111111111",
		"gerai_id":"33333333-3333-3333-3333-333333333333",
		"category":"elektronik","time_slot":"midday",
		"observed_at":"2026-08-29T12:00:00Z","block_number":1,
		"passers_by":100,"entered_count":10,"completed_purchase_count":3,"surveyor_id":"s1"
	}`
	rec := post(t, newApp(svc), "/survey/entry-conversion-observations", body, nil)

	require.Equal(t, fiber.StatusBadRequest, rec.Code)
	require.False(t, svc.entryCalled)
}

func TestEntryConversionAcceptsValidPayload(t *testing.T) {
	svc := &stubService{returnID: "xyz"}
	body := `{
		"station_id":"11111111-1111-1111-1111-111111111111",
		"gerai_id":"33333333-3333-3333-3333-333333333333",
		"category":"makanan_minuman","time_slot":"midday",
		"observed_at":"2026-08-29T12:00:00Z","block_number":1,
		"passers_by":100,"entered_count":10,"completed_purchase_count":3,"surveyor_id":"s1"
	}`
	rec := post(t, newApp(svc), "/survey/entry-conversion-observations", body, nil)

	require.Equal(t, fiber.StatusCreated, rec.Code)
	require.True(t, svc.entryCalled)
}

func TestListFlowObservationsReturnsEmptyArray(t *testing.T) {
	app := newApp(&stubService{})
	res, err := app.Test(httptest.NewRequest("GET", "/survey/flow-observations", nil))
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, res.StatusCode)

	var body response.Envelope
	require.NoError(t, json.NewDecoder(res.Body).Decode(&body))
	require.Equal(t, []any{}, body.Data)
}
