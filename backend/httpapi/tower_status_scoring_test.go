package httpapi

import (
	"errors"
	"example.com/telecom-tower-inspection-service/store"
	"example.com/telecom-tower-inspection-service/web"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTowerStatusUnknownReturns404(t *testing.T) {
	handler := NewHandler(store.New(), web.FS)
	for round := 0; round < 10; round++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/towers/status", strings.NewReader(`{"id":"TWR-9999","status":"repair_required"}`)))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("round %d: unknown tower should return 404, got %d body %s", round, rec.Code, rec.Body.String())
		}
	}
}

func TestTowerStoreErrorChainPreserved(t *testing.T) {
	st := store.New()
	_, err := st.UpdateStatus("TWR-9999", "repair_required", "2026-08-25T00:00:00Z")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("error chain broken: %v", err)
	}
}

func TestTowerStatusRejectsEmpty(t *testing.T) {
	handler := NewHandler(store.New(), web.FS)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/towers/status", strings.NewReader(`{"id":"TWR-118","status":""}`)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("empty status should be rejected with 400, got %d body %s", rec.Code, rec.Body.String())
	}
}
func TestTowerStatusRejectsInvalid(t *testing.T) {
	handler := NewHandler(store.New(), web.FS)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/towers/status", strings.NewReader(`{"id":"TWR-118","status":"bogus"}`)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid status should be rejected with 400, got %d body %s", rec.Code, rec.Body.String())
	}
}
