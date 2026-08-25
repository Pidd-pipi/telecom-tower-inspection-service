package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func reportStore(t *testing.T) *InspectionStore {
	t.Helper()
	seedCounters()
	store := newInspectionStore(seedTowers())
	ctx := context.Background()
	for _, item := range seedInspections() {
		if err := store.AddInspection(ctx, item); err != nil {
			t.Fatalf("seed inspection: %v", err)
		}
	}
	for _, f := range seedFindings() {
		if err := store.AddFinding(ctx, f); err != nil {
			t.Fatalf("seed finding: %v", err)
		}
	}
	return store
}

func TestReportGenerateNoNilMapPanic(t *testing.T) {
	store := reportStore(t)
	reporter := newTowerReporter()
	report, err := reporter.Generate(context.Background(), store)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if report.TotalInspections == 0 {
		t.Fatalf("expected inspections counted")
	}
}

func TestReportHighRiskCounts(t *testing.T) {
	store := reportStore(t)
	ctx := context.Background()
	// give TWR-301 three findings so it qualifies as high-risk
	for i := 0; i < 2; i++ {
		f := Finding{ID: newFindingID(), TowerID: "TWR-301", Severity: FindingSeverityMedium, Category: "corrosion", Detail: "extra finding", CreatedAt: "2026-08-25T00:00:00Z"}
		if err := store.AddFinding(ctx, f); err != nil {
			t.Fatalf("add finding: %v", err)
		}
	}
	reporter := newTowerReporter()
	report, err := reporter.Generate(ctx, store)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if report.HighRisk == 0 {
		t.Fatalf("expected at least one high-risk tower, got 0")
	}
}

func TestReportTopFindingsStable(t *testing.T) {
	store := reportStore(t)
	reporter := newTowerReporter()
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		f := Finding{ID: newFindingID(), TowerID: "TWR-118", Severity: FindingSeverityMedium, Category: "loose_bolt", Detail: "bolt", CreatedAt: "2026-08-25T00:00:00Z"}
		if err := store.AddFinding(ctx, f); err != nil {
			t.Fatalf("add finding: %v", err)
		}
	}
	first, err := reporter.Generate(ctx, store)
	if err != nil {
		t.Fatalf("generate 1: %v", err)
	}
	second, err := reporter.Generate(ctx, store)
	if err != nil {
		t.Fatalf("generate 2: %v", err)
	}
	if len(first.TopFindings) != len(second.TopFindings) {
		t.Fatalf("top findings length differs: %d vs %d", len(first.TopFindings), len(second.TopFindings))
	}
	for i := range first.TopFindings {
		if first.TopFindings[i].ID != second.TopFindings[i].ID {
			t.Fatalf("top findings order unstable at index %d: %s vs %s", i, first.TopFindings[i].ID, second.TopFindings[i].ID)
		}
	}
}

func TestReportRefreshesAfterChange(t *testing.T) {
	seedCounters()
	store := newInspectionStore(seedTowers())
	ctx := context.Background()
	for _, item := range seedInspections() {
		_ = store.AddInspection(ctx, item)
	}
	service := newInspectionService(store)
	handlers := newTowerHandlers(service)
	handler := handlers.routes()
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/report", nil))
	var first map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &first); err != nil {
		t.Fatalf("decode 1: %v", err)
	}
	// add a new scheduled inspection
	extra := TowerInspection{ID: "ins-new", TowerID: "TWR-118", Region: "Kanto North", Inspector: "t", ScheduledAt: "2026-09-01T09:00:00Z", Status: InspectionScheduled, FindingIDs: []string{}}
	_ = store.AddInspection(ctx, extra)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/api/report", nil))
	var second map[string]any
	if err := json.Unmarshal(rec2.Body.Bytes(), &second); err != nil {
		t.Fatalf("decode 2: %v", err)
	}
	if first["total_inspections"].(float64) == second["total_inspections"].(float64) {
		t.Fatalf("report did not refresh after data change: %v == %v", first["total_inspections"], second["total_inspections"])
	}
}
