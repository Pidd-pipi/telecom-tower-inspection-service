package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpsTransitionMissingReturns404(t *testing.T) {
	handler := newOpsAPIHandler(newOpsService(seedOpsRecords()))
	for round := 0; round < 10; round++ {
		body := strings.NewReader(`{"expected":1,"target":"active","actor":"lead"}`)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/ops/records/ops-9999/transition", body))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("round %d: expected 404 for missing record, got %d", round, rec.Code)
		}
	}
}

func TestOpsGetMissingReturns404(t *testing.T) {
	handler := newOpsAPIHandler(newOpsService(seedOpsRecords()))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ops/records/ops-9999", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing record, got %d", rec.Code)
	}
}

func TestOpsErrorChainPreserved(t *testing.T) {
	ctx := context.Background()
	store := newOpsStore(seedOpsRecords())
	_, err := store.Get(ctx, "ops-9999")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrOpsNotFound) {
		t.Fatalf("get error chain broken: %v", err)
	}
}

func TestOpsUpdateErrorChainPreserved(t *testing.T) {
	store := newOpsStore(seedOpsRecords())
	err := store.Update(context.Background(), OpsRecord{ID: "ops-9999"}, 0)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrOpsNotFound) {
		t.Fatalf("update error chain broken: %v", err)
	}
}

func TestOpsDeleteErrorChainPreserved(t *testing.T) {
	store := newOpsStore(seedOpsRecords())
	err := store.Delete(context.Background(), "ops-9999")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrOpsNotFound) {
		t.Fatalf("delete error chain broken: %v", err)
	}
}

func TestOpsCodeClassifiesNotFound(t *testing.T) {
	store := newOpsStore(seedOpsRecords())
	_, err := store.Get(context.Background(), "ops-9999")
	if err == nil {
		t.Fatal("expected error")
	}
	if code := opsCode(err); code != "not_found" {
		t.Fatalf("expected opsCode not_found, got %q", code)
	}
}
func TestOpsConflictChainPreserved(t *testing.T) {
	store := newOpsStore(seedOpsRecords())
	err := store.Update(context.Background(), OpsRecord{ID: "ops-0001"}, 999)
	if err == nil {
		t.Fatal("expected conflict error")
	}
	if !errors.Is(err, ErrOpsConflict) {
		t.Fatalf("conflict error chain broken: %v", err)
	}
}
