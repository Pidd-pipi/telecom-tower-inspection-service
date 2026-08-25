package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func seedStoreWithProgress(t *testing.T, count int) *InspectionStore {
	t.Helper()
	seedCounters()
	store := newInspectionStore(seedTowers())
	ctx := context.Background()
	for i := 0; i < count; i++ {
		item := TowerInspection{
			ID:          newInspectionID(),
			TowerID:     "TWR-118",
			Region:      "Kanto North",
			Inspector:   "tester",
			ScheduledAt: "2026-08-20T09:00:00Z",
			Status:      InspectionInProgress,
			FindingIDs:  []string{},
		}
		if err := store.AddInspection(ctx, item); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	return store
}

func countExportedEvents(handler http.Handler) (int, error) {
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/inspections/ins-0002/audit", nil))
	if rec.Code != http.StatusOK {
		return 0, nil
	}
	var events struct {
		Events []TowerEvent `json:"events"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &events); err != nil {
		return 0, err
	}
	exported := 0
	for _, ev := range events.Events {
		if ev.Type == "exported" {
			exported++
		}
	}
	return exported, nil
}

func TestExportAuditCountViaHTTP(t *testing.T) {
	seedCounters()
	handlers := make([]http.Handler, 4)
	for i := range handlers {
		handlers[i] = newRootHandler()
	}
	start := make(chan struct{})
	var wg sync.WaitGroup
	var mu sync.Mutex
	total := 0
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			handler := handlers[idx]
			body := strings.NewReader(`{"from":"in_progress","to":"completed"}`)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/export", body))
			if rec.Code != http.StatusOK {
				return
			}
			count, err := countExportedEvents(handler)
			if err != nil {
				return
			}
			mu.Lock()
			total += count
			mu.Unlock()
		}(i)
	}
	close(start)
	wg.Wait()
	if total != 4 {
		t.Fatalf("each export must add exactly one exported event per inspection, total=%d want 4", total)
	}
}

func TestExportPoolExhaustedReturnsError(t *testing.T) {
	store := seedStoreWithProgress(t, 3)
	exporter := newInspectionExporter(store, newTowerAudit())
	for i := 0; i < inspectionExportLeaseLimit; i++ {
		exporter.tokens <- struct{}{}
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := exporter.Export(context.Background(), InspectionInProgress, InspectionCompleted)
			results <- err
		}()
	}
	close(start)
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		close(results)
		for err := range results {
			if err == nil {
				t.Fatalf("expected pool exhaustion error, got nil")
			}
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("export hung when lease pool exhausted")
	}
}

func TestExportAuditConcurrentSafe(t *testing.T) {
	audit := newTowerAudit()
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < 100; j++ {
				audit.Add("TWR-118", "exported", "system")
			}
		}()
	}
	close(start)
	wg.Wait()
	if got := audit.Count(); got != 400 {
		t.Fatalf("concurrent audit writes lost events: got %d want 400", got)
	}
}
