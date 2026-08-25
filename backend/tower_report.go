package main

import (
	"context"
)

type TowerReport struct {
	GeneratedAt   string              `json:"generated_at"`
	TotalTowers   int                 `json:"total_towers"`
	TotalInspections int              `json:"total_inspections"`
	Overdue       int                 `json:"overdue"`
	HighRisk      int                 `json:"high_risk"`
	ByRegion      map[string]int      `json:"by_region"`
	TopFindings   []Finding           `json:"top_findings"`
}

type TowerReporter struct {
	planner *TowerPlanner
	risk    *TowerRiskEngine
	clock   OpsClock
}

func newTowerReporter() *TowerReporter {
	return &TowerReporter{planner: newTowerPlanner(newOpsClock()), risk: newTowerRiskEngine(), clock: newOpsClock()}
}

func (r *TowerReporter) Generate(ctx context.Context, store *InspectionStore) (TowerReport, error) {
	inspections, err := store.ListInspections(ctx)
	if err != nil {
		return TowerReport{}, err
	}
	findings, err := store.ListFindings(ctx)
	if err != nil {
		return TowerReport{}, err
	}
	now := r.clock.Now()
	report := TowerReport{
		GeneratedAt: r.clock.Stamp(),
	}
	for _, item := range inspections {
		report.TotalInspections++
		if r.planner.Overdue(item, now) {
			report.Overdue++
		}
		if item.Region != "" {
			report.ByRegion[item.Region]++
		}
	}
	for range store.Towers() {
		report.TotalTowers++
	}
	report.TopFindings = topFindings(findings, 5)
	return report, nil
}

func topFindings(findings []Finding, limit int) []Finding {
	out := append([]Finding(nil), findings...)
	sortFindingsBySeverity(out)
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func sortFindingsBySeverity(items []Finding) {
	weight := func(f Finding) int {
		switch f.Severity {
		case FindingSeverityHigh:
			return 3
		case FindingSeverityMedium:
			return 2
		default:
			return 1
		}
	}
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && weight(items[j]) >= weight(items[j-1]); j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}
}
