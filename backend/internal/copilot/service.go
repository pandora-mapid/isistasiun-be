package copilot

import (
	"context"
	"log"
	"strings"
)

// unavailableNote is appended when the model could not be reached. It says so
// in the answer itself rather than only in the log, because the reply is still
// served with 200 and the reader deserves to know it was assembled from
// keywords. The upstream error is never included: it can carry internal detail.
const unavailableNote = " Catatan: layanan AI sedang tidak tersedia, jadi jawaban ini disusun dari pembacaan kata kunci saja."

// Service holds the copilot's AI logic (Firaz per BACKEND_TASK_DIVISION):
// reading a natural-language query, mapping it to a spatial filter, choosing
// the relevant analytics endpoint, and shaping the reply. Transport — routing,
// validation, rate limiting — belongs to the handler and Client (Priyapta).
//
// The model is an enrichment, not a dependency. Every query is classified
// locally first, so the endpoint returns a usable filter and a truthful answer
// even when the model is unreachable. Section 3.6: the map must never be
// blocked by a copilot failure.
type Service struct {
	client queryClient
}

type queryClient interface {
	Query(ctx context.Context, req QueryRequest) (*QueryResponse, error)
}

func NewService(client queryClient) *Service {
	return &Service{client: client}
}

func (s *Service) Query(ctx context.Context, req QueryRequest) (*QueryResponse, error) {
	req.Query = strings.TrimSpace(req.Query)

	analysis := Classify(req.Query, req.StationID)
	local := &QueryResponse{
		Answer:          analysis.Answer(),
		SuggestedLayers: analysis.Layers,
		SpatialFilter:   analysis.SpatialFilter(),
	}

	if s.client == nil {
		local.Answer += unavailableNote
		return local, nil
	}

	resp, err := s.client.Query(ctx, req)
	if err != nil {
		// Degrade to the local reading rather than surfacing the failure: the
		// filter is still correct, only the prose is less fluent. Logged
		// without the error itself so a degraded upstream shows up in the logs
		// while its internals never reach the client.
		log.Print("copilot upstream request failed, serving local classification")
		local.Answer += unavailableNote
		return local, nil
	}

	return merge(local, resp), nil
}

// merge lets the model improve the prose while the locally derived filter
// stands as the floor. A model that returns nothing useful must not be able to
// strip the map of a filter the query plainly asked for; one that returns a
// better filter is trusted, because that is the part it was asked to do.
func merge(local, remote *QueryResponse) *QueryResponse {
	if remote == nil {
		return local
	}
	out := *remote
	if strings.TrimSpace(out.Answer) == "" {
		out.Answer = local.Answer
	}
	if len(out.SuggestedLayers) == 0 {
		out.SuggestedLayers = local.SuggestedLayers
	}
	if len(out.SpatialFilter) == 0 {
		out.SpatialFilter = local.SpatialFilter
	}
	return &out
}
