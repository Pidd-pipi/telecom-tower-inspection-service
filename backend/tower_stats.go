package main

import (
	"context"
	"sync"
)

type TowerSummary struct {
	Total       int `json:"total"`
	Scheduled   int `json:"scheduled"`
	InProgress  int `json:"in_progress"`
	Completed   int `json:"completed"`
	Cancelled   int `json:"cancelled"`
	OpenFindings int `json:"open_findings"`
	HighRisk    int `json:"high_risk"`
}

func computeTowerSummary(inspections []TowerInspection, findings []Finding, towers []TowerInfo) TowerSummary {
	out := TowerSummary{}
	openFindings := map[string]int{}
	for _, f := range findings {
		openFindings[f.TowerID]++
	}
	for _, item := range inspections {
		out.Total++
		switch item.Status {
		case InspectionScheduled:
			out.Scheduled++
		case InspectionInProgress:
			out.InProgress++
		case InspectionCompleted:
			out.Completed++
		case InspectionCancelled:
			out.Cancelled++
		}
	}
	for towerID, count := range openFindings {
		out.OpenFindings += count
		corrosion := 0
		for _, t := range towers {
			if t.ID == towerID {
				corrosion = t.Corrosion
				break
			}
		}
		if corrosion >= 7 || count >= 3 {
			out.HighRisk++
		}
	}
	return out
}

type TowerStats struct {
	mu     sync.Mutex
	cached *TowerSummary
}

func newTowerStats() *TowerStats { return &TowerStats{} }

func (s *TowerStats) Refresh(ctx context.Context, store *InspectionStore) (TowerSummary, error) {
	inspections, err := store.ListInspections(ctx)
	if err != nil {
		return TowerSummary{}, err
	}
	findings, err := store.ListFindings(ctx)
	if err != nil {
		return TowerSummary{}, err
	}
	out := computeTowerSummary(inspections, findings, store.Towers())
	s.mu.Lock()
	copy := out
	s.cached = &copy
	s.mu.Unlock()
	return out, nil
}

func (s *TowerStats) Cached() TowerSummary {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cached == nil {
		return TowerSummary{}
	}
	return *s.cached
}

