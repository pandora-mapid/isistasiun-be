package premium

import "context"

// expectedSlots is the four blocks that are actually counted (section 5.2).
// Hours between them are neither observed nor interpolated, so a station with
// fewer than four rows has gaps in coverage rather than low numbers.
const expectedSlots = 4

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) DeepAnalysis(ctx context.Context, stationID string) (*DeepAnalysisResponse, error) {
	station, err := s.repo.Station(ctx, stationID)
	if err != nil {
		return nil, err
	}

	gaps, err := s.repo.SpendingGap(ctx, stationID)
	if err != nil {
		return nil, err
	}
	categories, err := s.repo.CategoryGaps(ctx, stationID)
	if err != nil {
		return nil, err
	}
	plots, err := s.repo.RentFlowPlots(ctx, stationID)
	if err != nil {
		return nil, err
	}
	events, err := s.repo.EventPotential(ctx, stationID)
	if err != nil {
		return nil, err
	}
	confidence, err := s.repo.Confidence(ctx, stationID)
	if err != nil {
		return nil, err
	}
	strukTotal, strukAmbiguous, err := s.repo.StrukCoverage(ctx, stationID)
	if err != nil {
		return nil, err
	}

	return &DeepAnalysisResponse{
		Station:        *station,
		SpendingGap:    gaps,
		Totals:         sumSlots(gaps),
		CategoryGaps:   categories,
		RentFlowPlots:  plots,
		EventPotential: events,
		Confidence:     confidence,
		Coverage: Coverage{
			ThinSampleZones:  countThin(confidence),
			TotalZones:       len(confidence),
			StrukTotal:       strukTotal,
			StrukAmbiguous:   strukAmbiguous,
			StrukUsable:      strukTotal - strukAmbiguous,
			HasCompleteSlots: len(gaps) == expectedSlots,
		},
	}, nil
}

// sumSlots adds the observed slots. P10s are summed with P10s and P90s with
// P90s, which is deliberately conservative: it treats the slots as perfectly
// correlated and so widens the band rather than narrowing it. Combining the
// distributions properly belongs in the Monte Carlo run, not here — this layer
// never invents a number the pipeline did not produce.
func sumSlots(slots []SpendingGapSlot) Totals {
	t := Totals{SlotsWithData: len(slots), SlotsExpected: expectedSlots}
	for _, s := range slots {
		t.Potential.P10 += s.Potential.P10
		t.Potential.P90 += s.Potential.P90
		t.Captured.P10 += s.Captured.P10
		t.Captured.P90 += s.Captured.P90
		t.Gap.P10 += s.Gap.P10
		t.Gap.P90 += s.Gap.P90
	}

	// Share of potential spending the station already captures, taken at the
	// midpoint of each band. Left nil when there is no potential to divide by,
	// so the caller sees "not computed" instead of a misleading zero.
	if mid := midpoint(t.Potential); mid > 0 {
		rate := midpoint(t.Captured) / mid
		t.CaptureRateP50 = &rate
	}
	return t
}

func midpoint(r Range) float64 { return (r.P10 + r.P90) / 2 }

func countThin(zones []ConfidenceZone) int {
	n := 0
	for _, z := range zones {
		if z.IsThinSample {
			n++
		}
	}
	return n
}
