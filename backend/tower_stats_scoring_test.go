package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestSummaryViaHTTPNoRace(t *testing.T) {
	seedCounters()
	store := newInspectionStore(seedTowers())
	ctx := context.Background()
	for _, item := range seedInspections() {
		_ = store.AddInspection(ctx, item)
	}
	service := newInspectionService(store)
	handlers := newTowerHandlers(service)
	handler := handlers.routes()
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < 20; j++ {
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/summary", nil))
				if rec.Code != http.StatusOK {
					t.Errorf("summary status %d", rec.Code)
				}
			}
		}()
	}
	close(start)
	wg.Wait()
}

func TestSummaryCacheNoDataRace(t *testing.T) {
	seedCounters()
	store := newInspectionStore(seedTowers())
	stats := newTowerStats()
	ctx := context.Background()
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < 50; j++ {
				_, _ = stats.Refresh(ctx, store)
				_ = stats.Cached()
			}
		}()
	}
	close(start)
	wg.Wait()
}

func TestSummaryCachedBeforeRefresh(t *testing.T) {
	stats := newTowerStats()
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < 20; j++ {
				summary := stats.Cached()
				if summary.Total != 0 {
					t.Errorf("expected zero summary before first refresh, got %+v", summary)
				}
			}
		}()
	}
	close(start)
	wg.Wait()
}

func TestStatsTickerStopsCleanly(t *testing.T) {
	store := newInspectionStore(seedTowers())
	stats := newTowerStats()
	stop := make(chan struct{})
	done := startTowerStatsTicker(stats, store, 20*time.Millisecond, stop)
	start := make(chan struct{})
	results := make(chan struct{}, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			select {
			case <-done:
				results <- struct{}{}
			case <-time.After(2 * time.Second):
			}
		}()
	}
	close(start)
	close(stop)
	wg.Wait()
	if len(results) != 2 {
		t.Fatalf("ticker goroutine did not stop for all waiters: %d/2", len(results))
	}
}

func TestStatsTickerRefreshesRepeatedly(t *testing.T) {
	seedCounters()
	store := newInspectionStore(seedTowers())
	ctx := context.Background()
	for _, item := range seedInspections() {
		_ = store.AddInspection(ctx, item)
	}
	stats := newTowerStats()
	stop := make(chan struct{})
	done := startTowerStatsTicker(stats, store, 20*time.Millisecond, stop)
	defer func() {
		close(stop)
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Error("ticker leaked")
		}
	}()
	// wait for the first refresh
	time.Sleep(60 * time.Millisecond)
	extra := TowerInspection{ID: "ins-extra", TowerID: "TWR-118", Region: "Kanto North", Inspector: "t", ScheduledAt: "2026-09-01T09:00:00Z", Status: InspectionScheduled, FindingIDs: []string{}}
	_ = store.AddInspection(ctx, extra)
	time.Sleep(80 * time.Millisecond)
	if cached := stats.Cached(); cached.Total != 5 {
		t.Fatalf("ticker did not refresh repeatedly: cached total=%d want 5", cached.Total)
	}
}
