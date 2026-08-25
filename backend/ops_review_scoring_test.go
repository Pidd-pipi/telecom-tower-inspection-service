package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpsReviewTransitionViaHTTP(t *testing.T) {
	handler := newOpsAPIHandler(newOpsService(seedOpsRecords()))
	body := strings.NewReader(`{"expected":1,"target":"review","actor":"lead"}`)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/ops/records/ops-0001/transition", body))
	if rec.Code != http.StatusOK {
		t.Fatalf("transition into review should succeed, got %d body %s", rec.Code, rec.Body.String())
	}
}

func TestOpsReviewResumeAndClose(t *testing.T) {
	m := newOpsStateMachine()
	if err := m.Move(OpsStatusActive, OpsStatusReview, "send for review"); err != nil {
		t.Fatalf("active->review should be allowed: %v", err)
	}
	if err := m.Move(OpsStatusReview, OpsStatusActive, "resume"); err != nil {
		t.Fatalf("review->active should be allowed: %v", err)
	}
	if err := m.Move(OpsStatusActive, OpsStatusReview, "send for review"); err != nil {
		t.Fatalf("active->review again: %v", err)
	}
	if err := m.Move(OpsStatusReview, OpsStatusClosed, "close after review"); err != nil {
		t.Fatalf("review->closed should be allowed: %v", err)
	}
}

func TestOpsListActiveIncludesReview(t *testing.T) {
	service := newOpsService(seedOpsRecords())
	ctx := context.Background()
	record, err := service.Get(ctx, "ops-0001")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	record.Status = OpsStatusReview
	if err := service.store.Update(ctx, record, 0); err != nil {
		t.Fatalf("update to review: %v", err)
	}
	handler := newOpsAPIHandler(service)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ops/records?status=active", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("list status %d", rec.Code)
	}
	var page OpsPage
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode: %v", err)
	}
	for _, item := range page.Items {
		if item.ID == "ops-0001" {
			return
		}
	}
	t.Fatalf("review record missing from active list; items=%d", page.Total)
}

func TestOpsSnapshotCountsReview(t *testing.T) {
	service := newOpsService(seedOpsRecords())
	ctx := context.Background()
	record, err := service.Get(ctx, "ops-0002")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	record.Status = OpsStatusReview
	if err := service.store.Update(ctx, record, 0); err != nil {
		t.Fatalf("update to review: %v", err)
	}
	snapshot := service.Snapshot()
	// ops-0001 active + ops-0002 review -> active count must be 2
	if snapshot.Active != 2 {
		t.Fatalf("snapshot active should include review records, got %d", snapshot.Active)
	}
}
