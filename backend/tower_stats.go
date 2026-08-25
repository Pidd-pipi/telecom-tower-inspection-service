package main

import (
	"context"
	"sync"
)

type TowerSummary struct {
	Total        int `json:"total"`
	Scheduled    int `json:"scheduled"`
	InProgress   int `json:"in_progress"`
	Completed    int `json:"completed"`
	Cancelled    int `json:"cancelled"`
	OpenFindings int `json:"open_findings"`
	HighRisk     int `json:"high_risk"`
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

// TowerStats caches the tower inspection summary. All access to the cached
// value goes through mu so concurrent refreshes and reads stay race-free.
type TowerStats struct {
	mu     sync.RWMutex
	cached *TowerSummary
	dirty  bool
}

func newTowerStats() *TowerStats { return &TowerStats{dirty: true} }

// Refresh recomputes the summary from the store and publishes it atomically.
// It is safe to call concurrently and from multiple goroutines.
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

	// Swap the cached pointer under the write lock; readers never observe a
	// half-written value, and concurrent refreshes do not race.
	s.mu.Lock()
	s.cached = &out
	s.dirty = false
	s.mu.Unlock()
	return out, nil
}

// Cached returns the last computed summary without recomputing. When no
// summary has been produced yet it returns the zero value rather than
// dereferencing a nil pointer.
func (s *TowerStats) Cached() TowerSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cached == nil {
		return TowerSummary{}
	}
	return *s.cached
}

// Summary returns a fresh summary, recomputing only when the cache is missing
// or has been marked stale by Invalidate. This keeps the numbers correct after
// data changes while avoiding a full recompute on every read.
func (s *TowerStats) Summary(ctx context.Context, store *InspectionStore) (TowerSummary, error) {
	s.mu.RLock()
	need := s.cached == nil || s.dirty
	s.mu.RUnlock()
	if !need {
		return s.Cached(), nil
	}
	return s.Refresh(ctx, store)
}

// Invalidate marks the cache stale so the next Summary call recomputes.
// Callers must invoke this whenever inspection, finding or work-order data
// changes, otherwise the summary can lag behind the live store.
func (s *TowerStats) Invalidate() {
	s.mu.Lock()
	s.dirty = true
	s.mu.Unlock()
}
