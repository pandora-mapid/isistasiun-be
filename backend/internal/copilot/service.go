package copilot

import (
	"context"
	"fmt"
)

// Service is the routing-only layer Priyapta owns: validation, forwarding,
// and a safe fallback if the AI service times out or errors. Actual intent
// classification / prompt logic belongs to Firaz — either inside this AI
// service call, or by replacing Client with a direct in-process call into
// Firaz's package once that's ready.
type Service struct {
	client *Client
}

func NewService(client *Client) *Service {
	return &Service{client: client}
}

func (s *Service) Query(ctx context.Context, req QueryRequest) (*QueryResponse, error) {
	resp, err := s.client.Query(ctx, req)
	if err != nil {
		// Fallback per section 3.6: keep the endpoint resilient even if the
		// AI service is degraded — never block the map UI on copilot failures.
		return &QueryResponse{
			Answer: fmt.Sprintf("Maaf, AI copilot sedang tidak tersedia. Coba lagi sebentar lagi. (%v)", err),
		}, nil
	}
	return resp, nil
}
