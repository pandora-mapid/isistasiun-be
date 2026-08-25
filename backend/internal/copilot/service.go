package copilot

import (
	"context"
	"log"
)

// Service is the routing-only layer Priyapta owns: validation, forwarding,
// and a safe fallback if the AI service times out or errors. Actual intent
// classification / prompt logic belongs to Firaz — either inside this AI
// service call, or by replacing Client with a direct in-process call into
// Firaz's package once that's ready.
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
	resp, err := s.client.Query(ctx, req)
	if err != nil {
		log.Print("copilot upstream request failed")
		return &QueryResponse{
			Answer: "Maaf, AI copilot sedang tidak tersedia. Coba lagi sebentar lagi.",
		}, nil
	}
	return resp, nil
}
