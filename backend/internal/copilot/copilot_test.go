package copilot

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
)

type stubCopilotService struct {
	received QueryRequest
}

func (s *stubCopilotService) Query(_ context.Context, req QueryRequest) (*QueryResponse, error) {
	s.received = req
	return &QueryResponse{Answer: "Jawaban"}, nil
}

type failingClient struct{}

func (failingClient) Query(context.Context, QueryRequest) (*QueryResponse, error) {
	return nil, errors.New("secret upstream detail")
}

func TestHandlerCountsUnicodeCharacters(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		wantStatus int
	}{
		{name: "500 unicode characters accepted", query: strings.Repeat("é", 500), wantStatus: fiber.StatusOK},
		{name: "501 unicode characters rejected", query: strings.Repeat("é", 501), wantStatus: fiber.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			NewHandler(&stubCopilotService{}).RegisterRoutes(app)
			payload, err := json.Marshal(QueryRequest{Query: tt.query})
			require.NoError(t, err)
			req := httptest.NewRequest(http.MethodPost, "/copilot/query", strings.NewReader(string(payload)))
			req.Header.Set("Content-Type", "application/json")

			res, err := app.Test(req)
			require.NoError(t, err)
			require.Equal(t, tt.wantStatus, res.StatusCode)
		})
	}
}

func TestServiceFallbackDoesNotExposeUpstreamError(t *testing.T) {
	svc := NewService(failingClient{})

	result, err := svc.Query(context.Background(), QueryRequest{Query: "test"})
	require.NoError(t, err)
	require.NotContains(t, result.Answer, "secret upstream detail")
	require.Contains(t, result.Answer, "tidak tersedia")
}

func TestClientUsesBareResponseContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/copilot/query", r.URL.Path)
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))
		_ = json.NewEncoder(w).Encode(QueryResponse{
			Answer:          "  Peluang tertinggi ada di area A.  ",
			SuggestedLayers: []string{"spending-gap"},
		})
	}))
	defer server.Close()

	result, err := NewClient(server.URL+"/", 2).Query(context.Background(), QueryRequest{Query: "peluang"})
	require.NoError(t, err)
	require.Equal(t, "Peluang tertinggi ada di area A.", result.Answer)
	require.Equal(t, []string{"spending-gap"}, result.SuggestedLayers)
}

func TestClientRejectsInvalidResponses(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr string
	}{
		{name: "upstream error", status: http.StatusBadGateway, body: "failed", wantErr: "ai service error (502)"},
		{name: "malformed json", status: http.StatusOK, body: "{", wantErr: "decode ai service response"},
		{name: "empty answer", status: http.StatusOK, body: `{"answer":"   "}`, wantErr: "empty answer"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			_, err := NewClient(server.URL, 2).Query(context.Background(), QueryRequest{Query: "test"})
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}
