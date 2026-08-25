package httpapi

import (
	"encoding/json"
	"example.com/telecom-tower-inspection-service/domain"
	"example.com/telecom-tower-inspection-service/store"
	"net/http"
	"strings"
	"time"
)

type server struct{ store *store.Store }

func (s *server) collection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string][]domain.Tower{"items": s.store.List()})
}
func (s *server) status(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var request struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&request); err != nil || strings.TrimSpace(request.ID) == "" {
		writeError(w, http.StatusBadRequest, "id and status are required")
		return
	}
	item, _ := s.store.UpdateStatus(request.ID, request.Status, time.Now().UTC().Format(time.RFC3339))
	writeJSON(w, http.StatusOK, item)
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
