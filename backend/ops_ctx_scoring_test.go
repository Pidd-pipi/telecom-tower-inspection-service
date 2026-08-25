package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestOpsTransitionRespectsCancel(t *testing.T) {
	service := newOpsService(seedOpsRecords())
	handler := newOpsAPIHandler(service)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-canceled
	body := strings.NewReader(`{"expected":1,"target":"active","actor":"lead"}`)
	req := httptest.NewRequest(http.MethodPost, "/ops/records/ops-0002/transition", body)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	record, err := service.Get(context.Background(), "ops-0002")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if record.Status != OpsStatusQueued {
		t.Fatalf("canceled request still transitioned record: status=%s", record.Status)
	}
}

func TestOpsContextKeepsParent(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	ctx, _ := opsContext(parent, time.Second)
	cancel()
	select {
	case <-ctx.Done():
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("opsContext dropped parent cancellation")
	}
}

func TestOpsDelayRespectsCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	err := opsDelay(ctx, 800*time.Millisecond)
	if err == nil {
		t.Fatalf("opsDelay should return the cancellation error")
	}
	if elapsed := time.Since(start); elapsed > 400*time.Millisecond {
		t.Fatalf("opsDelay ignored cancellation, waited %v", elapsed)
	}
}

func TestOpsTransitionContextUsesRequest(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodPost, "/ops/records/ops-0002/transition", nil)
	req = req.WithContext(ctx)
	tctx, tcancel := opsTransitionContext(req)
	defer tcancel()
	select {
	case <-tctx.Done():
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("opsTransitionContext dropped request cancellation")
	}
}
