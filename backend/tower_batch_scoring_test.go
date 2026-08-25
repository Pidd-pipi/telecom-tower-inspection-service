package main

import (
	"context"
	"testing"
	"time"
)

func batchStoreWithProgress(t *testing.T, scheduledAts []string) *InspectionStore {
	t.Helper()
	seedCounters()
	store := newInspectionStore(seedTowers())
	ctx := context.Background()
	for i, at := range scheduledAts {
		item := TowerInspection{
			ID:          newInspectionID(),
			TowerID:     "TWR-118",
			Region:      "Kanto North",
			Inspector:   "tester",
			ScheduledAt: at,
			Status:      InspectionInProgress,
			FindingIDs:  []string{},
		}
		if err := store.AddInspection(ctx, item); err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
	}
	return store
}

func TestBatchLeaseReleasedOnError(t *testing.T) {
	store := batchStoreWithProgress(t, []string{"2026-08-20T09:00:00Z"})
	audit := newTowerAudit()
	processor := newTowerBatchProcessor(store, audit, newTowerPlanner(newOpsClock()), newTowerNotifier())
	_, err := processor.Run(context.Background(), "not-a-date")
	if err == nil {
		t.Fatal("expected invalid cutoff error")
	}
	completed, err := processor.Run(context.Background(), "2026-08-30T00:00:00Z")
	if err != nil {
		t.Fatalf("second run should not be blocked by leaked lease: %v", err)
	}
	if len(completed) != 1 {
		t.Fatalf("expected 1 completed, got %d", len(completed))
	}
}

func TestBatchAuditOrderPreserved(t *testing.T) {
	seedCounters()
	store := newInspectionStore(seedTowers())
	ctx := context.Background()
	order := []struct {
		id    string
		tower string
	}{{"ins-a", "TWR-118"}, {"ins-b", "TWR-204"}, {"ins-c", "TWR-301"}}
	for _, o := range order {
		item := TowerInspection{ID: o.id, TowerID: o.tower, Region: "Kanto North", Inspector: "t", ScheduledAt: "2026-08-20T09:00:00Z", Status: InspectionInProgress, FindingIDs: []string{}}
		if err := store.AddInspection(ctx, item); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	audit := newTowerAudit()
	processor := newTowerBatchProcessor(store, audit, newTowerPlanner(newOpsClock()), newTowerNotifier())
	_, err := processor.Run(ctx, "2026-08-30T00:00:00Z")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	all := audit.Since(time.Time{})
	batchTowers := []string{}
	for _, ev := range all {
		if ev.Type == "batch_completed" {
			batchTowers = append(batchTowers, ev.TowerID)
		}
	}
	if len(batchTowers) != 3 {
		t.Fatalf("expected 3 batch_completed events, got %d", len(batchTowers))
	}
	want := []string{"TWR-118", "TWR-204", "TWR-301"}
	for i := range want {
		if batchTowers[i] != want[i] {
			t.Fatalf("audit events recorded out of order: got %v want %v", batchTowers, want)
		}
	}
}

func TestBatchNotifyPerTower(t *testing.T) {
	seedCounters()
	store := newInspectionStore(seedTowers())
	ctx := context.Background()
	items := []TowerInspection{
		{ID: "ins-a", TowerID: "TWR-118", Region: "Kanto North", Inspector: "t", ScheduledAt: "2026-08-20T09:00:00Z", Status: InspectionInProgress, FindingIDs: []string{}},
		{ID: "ins-b", TowerID: "TWR-204", Region: "Kanto Coast", Inspector: "t", ScheduledAt: "2026-08-20T09:00:00Z", Status: InspectionInProgress, FindingIDs: []string{}},
	}
	for _, item := range items {
		if err := store.AddInspection(ctx, item); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	audit := newTowerAudit()
	notifier := newTowerNotifier()
	chA := notifier.Subscribe("TWR-118")
	chB := notifier.Subscribe("TWR-204")
	processor := newTowerBatchProcessor(store, audit, newTowerPlanner(newOpsClock()), notifier)
	if _, err := processor.Run(ctx, "2026-08-30T00:00:00Z"); err != nil {
		t.Fatalf("run: %v", err)
	}
	select {
	case ev := <-chA:
		if ev.TowerID != "TWR-118" {
			t.Fatalf("TWR-118 subscriber got event for %s", ev.TowerID)
		}
	default:
		t.Fatalf("TWR-118 subscriber received no completion event")
	}
	select {
	case ev := <-chB:
		if ev.TowerID != "TWR-204" {
			t.Fatalf("TWR-204 subscriber got event for %s", ev.TowerID)
		}
	default:
		t.Fatalf("TWR-204 subscriber received no completion event")
	}
}

func TestBatchSkipsFutureInspections(t *testing.T) {
	seedCounters()
	store := newInspectionStore(seedTowers())
	ctx := context.Background()
	future := TowerInspection{ID: "ins-future", TowerID: "TWR-118", Region: "Kanto North", Inspector: "t", ScheduledAt: "2026-12-31T09:00:00Z", Status: InspectionInProgress, FindingIDs: []string{}}
	overdue := TowerInspection{ID: "ins-overdue", TowerID: "TWR-118", Region: "Kanto North", Inspector: "t", ScheduledAt: "2026-08-01T09:00:00Z", Status: InspectionInProgress, FindingIDs: []string{}}
	for _, item := range []TowerInspection{future, overdue} {
		if err := store.AddInspection(ctx, item); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	audit := newTowerAudit()
	processor := newTowerBatchProcessor(store, audit, newTowerPlanner(newOpsClock()), newTowerNotifier())
	completed, err := processor.Run(ctx, "2026-09-01T00:00:00Z")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	for _, id := range completed {
		if id == "ins-future" {
			t.Fatalf("future inspection should not be completed by batch")
		}
	}
	got, err := store.GetInspection(ctx, "ins-future")
	if err != nil {
		t.Fatalf("get future: %v", err)
	}
	if got.Status == InspectionCompleted {
		t.Fatalf("future inspection was completed")
	}
}

var _ = time.RFC3339
