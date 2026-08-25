package httpapi

import (
	"example.com/telecom-tower-inspection-service/store"
	"example.com/telecom-tower-inspection-service/web"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTowerRoutes(t *testing.T) {
	handler := NewHandler(store.New(), web.FS)
	get := httptest.NewRecorder()
	handler.ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/api/towers", nil))
	if get.Code != http.StatusOK || !strings.Contains(get.Body.String(), "TWR-118") {
		t.Fatalf("collection: %d %s", get.Code, get.Body.String())
	}
	post := httptest.NewRecorder()
	handler.ServeHTTP(post, httptest.NewRequest(http.MethodPost, "/api/towers/status", strings.NewReader(`{"id":"TWR-204","status":"repair_required"}`)))
	if post.Code != http.StatusOK || !strings.Contains(post.Body.String(), `"status":"repair_required"`) {
		t.Fatalf("update: %d %s", post.Code, post.Body.String())
	}
}
func TestTowerRejectsInvalidStatus(t *testing.T) {
	handler := NewHandler(store.New(), web.FS)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/towers/status", strings.NewReader(`{"id":"TWR-118","status":"safe"}`)))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", response.Code)
	}
}
