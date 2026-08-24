package station

import "context"

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListStations(ctx context.Context, areaType, operator string) ([]StationResponse, error) {
	stations, err := s.repo.List(ctx, areaType, operator)
	if err != nil {
		return nil, err
	}
	out := make([]StationResponse, 0, len(stations))
	for _, st := range stations {
		out = append(out, toStationResponse(st))
	}
	return out, nil
}

func (s *Service) GetStation(ctx context.Context, id string) (*StationResponse, error) {
	st, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := toStationResponse(*st)
	return &resp, nil
}

func (s *Service) ListEntrances(ctx context.Context, stationID string) ([]EntranceResponse, error) {
	// Ensure station exists first so callers get a clean 404 instead of an empty list.
	if _, err := s.repo.GetByID(ctx, stationID); err != nil {
		return nil, err
	}
	entrances, err := s.repo.ListEntrances(ctx, stationID)
	if err != nil {
		return nil, err
	}
	out := make([]EntranceResponse, 0, len(entrances))
	for _, e := range entrances {
		out = append(out, toEntranceResponse(e))
	}
	return out, nil
}
