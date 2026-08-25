package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTowerRiskScoreNoNilMapPanic(t *testing.T) {
	seedCounters()
	store := newInspectionStore(seedTowers())
	ctx := context.Background()
	for _, f := range seedFindings() {
		if err := store.AddFinding(ctx, f); err != nil {
			t.Fatalf("seed finding: %v", err)
		}
	}
	service := newInspectionService(store)
	handlers := newTowerHandlers(service)
	rec := httptest.NewRecorder()
	handlers.routes().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/risk/TWR-301", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("risk endpoint for tower with findings should return 200, got %d body %s", rec.Code, rec.Body.String())
	}
}

func TestTowerRiskScoreCapped(t *testing.T) {
	engine := newTowerRiskEngine()
	findings := []Finding{}
	for i := 0; i < 12; i++ {
		findings = append(findings, Finding{TowerID: "TWR-X", Severity: FindingSeverityHigh})
	}
	profile := engine.Score(TowerInfo{ID: "TWR-X", Corrosion: 10, WindLoadPct: 100}, findings)
	if profile.Score > 100 {
		t.Fatalf("score must be capped at 100, got %d", profile.Score)
	}
}

func TestTowerRiskOpenFindingsPerTower(t *testing.T) {
	engine := newTowerRiskEngine()
	findings := []Finding{
		{TowerID: "TWR-118", Severity: FindingSeverityMedium},
		{TowerID: "TWR-301", Severity: FindingSeverityHigh},
		{TowerID: "TWR-118", Severity: FindingSeverityLow},
	}
	profile := engine.Score(TowerInfo{ID: "TWR-118", Corrosion: 2, WindLoadPct: 30}, findings)
	if profile.OpenFindings != 2 {
		t.Fatalf("open findings must count only this tower's findings, got %d", profile.OpenFindings)
	}
}

func TestRecordFindingRequiresSeverity(t *testing.T) {
	seedCounters()
	store := newInspectionStore(seedTowers())
	service := newInspectionService(store)
	handlers := newTowerHandlers(service)
	body := strings.NewReader(`{"tower_id":"TWR-118","category":"loose_bolt","detail":"missing severity"}`)
	rec := httptest.NewRecorder()
	handlers.routes().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/findings", body))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("finding without severity must be rejected with 400, got %d body %s", rec.Code, rec.Body.String())
	}
}

func TestRecordFindingRejectsInvalidSeverity(t *testing.T) {
	seedCounters()
	store := newInspectionStore(seedTowers())
	service := newInspectionService(store)
	handlers := newTowerHandlers(service)
	body := strings.NewReader(`{"tower_id":"TWR-118","severity":"bogus","category":"loose_bolt","detail":"bad severity"}`)
	rec := httptest.NewRecorder()
	handlers.routes().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/findings", body))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("finding with invalid severity must be rejected with 400, got %d body %s", rec.Code, rec.Body.String())
	}
}
