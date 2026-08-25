package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func seededInspectionAPI(t *testing.T) *inspectionAPI {
	t.Helper()
	seedCounters()
	store := newInspectionStore(seedTowers())
	ctx := context.Background()
	for _, item := range seedInspections() {
		if err := store.AddInspection(ctx, item); err != nil {
			t.Fatalf("seed inspection: %v", err)
		}
	}
	service := newInspectionService(store)
	return newInspectionAPI(service)
}

func TestInspectionStoreListReturnsCopy(t *testing.T) {
	ctx := context.Background()
	store := newInspectionStore(seedTowers())
	for _, item := range seedInspections() {
		if err := store.AddInspection(ctx, item); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	items, err := store.ListInspections(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("expected seeded inspections")
	}
	items[0].Status = InspectionCancelled
	again, err := store.ListInspections(ctx)
	if err != nil {
		t.Fatalf("list again: %v", err)
	}
	for _, item := range again {
		if item.ID == items[0].ID && item.Status == InspectionCancelled {
			t.Fatalf("store returned shared slice: mutation leaked into store")
		}
	}
}

func TestInspectionFilterInputNotMutated(t *testing.T) {
	items := []TowerInspection{
		{ID: "a", Region: "Kanto North", Status: InspectionScheduled},
		{ID: "b", Region: "Kanto Coast", Status: InspectionScheduled},
		{ID: "c", Region: "Kanto North", Status: InspectionInProgress},
		{ID: "d", Region: "Chubu Hills", Status: InspectionScheduled},
	}
	before := make([]TowerInspection, len(items))
	copy(before, items)
	filtered := filterInspections(items, InspectionQuery{Region: "Kanto North"})
	if len(filtered) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(filtered))
	}
	for i := range items {
		if items[i].ID != before[i].ID {
			t.Fatalf("filter mutated input at index %d: got %s want %s", i, items[i].ID, before[i].ID)
		}
	}
}

func TestFindingsFilterInputNotMutated(t *testing.T) {
	items := []Finding{
		{ID: "f1", TowerID: "TWR-118", Severity: FindingSeverityHigh},
		{ID: "f2", TowerID: "TWR-204", Severity: FindingSeverityLow},
		{ID: "f3", TowerID: "TWR-301", Severity: FindingSeverityMedium},
	}
	before := make([]Finding, len(items))
	copy(before, items)
	filtered := filterFindings(items, InspectionQuery{Severity: FindingSeverityLow})
	if len(filtered) != 1 {
		t.Fatalf("expected 1 match, got %d", len(filtered))
	}
	for i := range items {
		if items[i].ID != before[i].ID {
			t.Fatalf("findings filter mutated input at index %d: got %s want %s", i, items[i].ID, before[i].ID)
		}
	}
}

func TestInspectionListCombinedFilterCorrect(t *testing.T) {
	api := seededInspectionAPI(t)
	handler := api.routes()
	for round := 0; round < 10; round++ {
		req := httptest.NewRequest(http.MethodGet, "/api/inspections?region=Kyushu%20South&status=scheduled", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("round %d: status %d", round, rec.Code)
		}
		var page InspectionPage
		if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
			t.Fatalf("round %d: decode: %v", round, err)
		}
		if page.Total != 1 || len(page.Items) != 1 {
			t.Fatalf("round %d: expected exactly 1 matching inspection, got total=%d items=%d", round, page.Total, len(page.Items))
		}
		for _, item := range page.Items {
			if item.Region != "Kyushu South" || item.Status != InspectionScheduled {
				t.Fatalf("round %d: unexpected item %+v", round, item)
			}
		}
	}
}

func TestInspectionListPaginationFiltered(t *testing.T) {
	api := seededInspectionAPI(t)
	handler := api.routes()
	req := httptest.NewRequest(http.MethodGet, "/api/inspections?region=Kyushu%20South&status=scheduled&page=1&page_size=1", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	var page InspectionPage
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if page.Total != 1 {
		t.Fatalf("pagination total should be 1, got %d", page.Total)
	}
	for _, item := range page.Items {
		if item.Region != "Kyushu South" || item.Status != InspectionScheduled {
			t.Fatalf("pagination returned non-matching item %+v", item)
		}
	}
}
