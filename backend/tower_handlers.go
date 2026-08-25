package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// towerHandlers exposes findings, work orders, risk and summary endpoints.
type towerHandlers struct {
	service  *InspectionService
	exporter *InspectionExporter
	stats    *TowerStats
	reporter *TowerReporter

	// statsStop closes to signal the background stats ticker to exit;
	// statsDone is closed by the ticker goroutine once it has fully stopped.
	statsStop chan struct{}
	statsDone <-chan struct{}
}

func newTowerHandlers(service *InspectionService) *towerHandlers {
	store := service.Store()
	stats := newTowerStats()
	// Keep the stats cache fresh: mark it stale whenever inspection, finding
	// or work-order data changes, so the summary endpoint never serves a
	// cached value computed before the latest mutation.
	service.setStatsInvalidator(stats.Invalidate)
	return &towerHandlers{
		service:  service,
		exporter: newInspectionExporter(store, service.audit),
		stats:    stats,
		reporter: newTowerReporter(),
	}
}

func (h *towerHandlers) listFindings(w http.ResponseWriter, r *http.Request) {
	q := InspectionQuery{
		TowerID:  r.URL.Query().Get("tower_id"),
		Severity: FindingSeverity(r.URL.Query().Get("severity")),
	}
	if page, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil {
		q.Page = page
	}
	if size, err := strconv.Atoi(r.URL.Query().Get("page_size")); err == nil {
		q.PageSize = size
	}
	items, err := h.service.Store().ListFindings(r.Context())
	if err != nil {
		opsJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	filtered := filterFindings(items, q)
	q = towerQueryDefaults(q)
	start, end := towerBounds(len(filtered), q.Page, q.PageSize)
	opsJSON(w, http.StatusOK, map[string]any{
		"items": filtered[start:end], "page": q.Page, "page_size": q.PageSize,
		"total": len(filtered), "has_next": end < len(filtered),
	})
}

func (h *towerHandlers) createFinding(w http.ResponseWriter, r *http.Request) {
	var req RecordFindingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		opsJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	finding, err := h.service.RecordFinding(r.Context(), req)
	if err != nil {
		opsJSON(w, inspectionStatusForError(err), map[string]string{"error": err.Error()})
		return
	}
	opsJSON(w, http.StatusCreated, finding)
}

func (h *towerHandlers) listOrders(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.Store().ListOrders(r.Context())
	if err != nil {
		opsJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	opsJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *towerHandlers) assignOrder(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/workorders/")
	id = strings.TrimSuffix(id, "/assign")
	var req struct {
		Assignee string `json:"assignee"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		opsJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	order, err := h.service.AssignOrder(r.Context(), id, req.Assignee)
	if err != nil {
		opsJSON(w, inspectionStatusForError(err), map[string]string{"error": err.Error()})
		return
	}
	opsJSON(w, http.StatusOK, order)
}

func (h *towerHandlers) resolveOrder(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/workorders/")
	id = strings.TrimSuffix(id, "/resolve")
	order, err := h.service.ResolveOrder(r.Context(), id)
	if err != nil {
		opsJSON(w, inspectionStatusForError(err), map[string]string{"error": err.Error()})
		return
	}
	opsJSON(w, http.StatusOK, order)
}

func (h *towerHandlers) risk(w http.ResponseWriter, r *http.Request) {
	towerID := strings.TrimPrefix(r.URL.Path, "/api/risk/")
	profile, err := h.service.Risk(r.Context(), towerID)
	if err != nil {
		opsJSON(w, inspectionStatusForError(err), map[string]string{"error": err.Error()})
		return
	}
	opsJSON(w, http.StatusOK, profile)
}

func (h *towerHandlers) summary(w http.ResponseWriter, r *http.Request) {
	// Summary recomputes only when the cache is stale (set by Invalidate after
	// data changes) or absent; otherwise it serves the cached value. This keeps
	// the numbers correct without a full recompute on every request.
	summary, err := h.stats.Summary(r.Context(), h.service.Store())
	if err != nil {
		opsJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	opsJSON(w, http.StatusOK, summary)
}

func (h *towerHandlers) export(w http.ResponseWriter, r *http.Request) {
	var req struct {
		From InspectionStatus `json:"from"`
		To   InspectionStatus `json:"to"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		opsJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	result, err := h.exporter.Export(ctx, req.From, req.To)
	if err != nil {
		opsJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	opsJSON(w, http.StatusOK, result)
}

func (h *towerHandlers) report(w http.ResponseWriter, r *http.Request) {
	report, err := h.reporter.Generate(r.Context(), h.service.Store())
	if err != nil {
		opsJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	opsJSON(w, http.StatusOK, report)
}

// startBackgroundStats launches a ticker that periodically refreshes the
// summary cache. It is a safety net for cases where Invalidate was not wired
// (e.g. seed data or future callers) and keeps the cache from sitting stale.
// Calling it twice is a no-op.
func (h *towerHandlers) startBackgroundStats(interval time.Duration) {
	if h.statsStop != nil {
		return
	}
	stop := make(chan struct{})
	h.statsStop = stop
	h.statsDone = startTowerStatsTicker(h.stats, h.service.Store(), interval, stop)
}

// Shutdown stops the background stats ticker and blocks until it has exited,
// so the process can come down cleanly instead of leaking a goroutine. It is
// safe to call more than once.
func (h *towerHandlers) Shutdown() {
	if h.statsStop == nil {
		return
	}
	close(h.statsStop)
	if h.statsDone != nil {
		<-h.statsDone
	}
	h.statsStop = nil
}

func (h *towerHandlers) routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/findings", h.listFindings)
	mux.HandleFunc("POST /api/findings", h.createFinding)
	mux.HandleFunc("GET /api/workorders", h.listOrders)
	mux.HandleFunc("POST /api/workorders/{id}/assign", h.assignOrder)
	mux.HandleFunc("POST /api/workorders/{id}/resolve", h.resolveOrder)
	mux.HandleFunc("GET /api/risk/{towerID}", h.risk)
	mux.HandleFunc("GET /api/summary", h.summary)
	mux.HandleFunc("POST /api/export", h.export)
	mux.HandleFunc("GET /api/report", h.report)
	return mux
}
