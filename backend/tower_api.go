package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
)

type inspectionAPI struct {
	service  *InspectionService
	exporter *InspectionExporter
	notifier *TowerNotifier
	stats    *TowerStats
	reporter *TowerReporter
}

func newInspectionAPI(service *InspectionService) *inspectionAPI {
	store := service.Store()
	return &inspectionAPI{
		service:  service,
		exporter: newInspectionExporter(store, service.audit),
		notifier: newTowerNotifier(),
		stats:    newTowerStats(),
		reporter: newTowerReporter(),
	}
}

func inspectionStatusForError(err error) int {
	switch {
	case errors.Is(err, ErrOpsNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrOpsConflict):
		return http.StatusConflict
	case errors.Is(err, ErrOpsInvalid):
		return http.StatusBadRequest
	case errors.Is(err, ErrOpsTransition):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func (a *inspectionAPI) list(w http.ResponseWriter, r *http.Request) {
	q := InspectionQuery{
		TowerID: r.URL.Query().Get("tower_id"),
		Region:  r.URL.Query().Get("region"),
		Status:  InspectionStatus(r.URL.Query().Get("status")),
	}
	if page, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil {
		q.Page = page
	}
	if size, err := strconv.Atoi(r.URL.Query().Get("page_size")); err == nil {
		q.PageSize = size
	}
	items, err := a.service.Store().ListInspections(r.Context())
	if err != nil {
		opsJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	filtered := filterInspections(items, q)
	page := towerPage(filtered, q)
	opsJSON(w, http.StatusOK, page)
}

func (a *inspectionAPI) create(w http.ResponseWriter, r *http.Request) {
	var req ScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		opsJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	item, err := a.service.Schedule(r.Context(), req)
	if err != nil {
		opsJSON(w, inspectionStatusForError(err), map[string]string{"error": err.Error()})
		return
	}
	opsJSON(w, http.StatusCreated, item)
}

func (a *inspectionAPI) get(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/inspections/")
	if id == "" {
		opsJSON(w, http.StatusNotFound, map[string]string{"error": "route not found"})
		return
	}
	item, err := a.service.Store().GetInspection(r.Context(), id)
	if err != nil {
		opsJSON(w, inspectionStatusForError(err), map[string]string{"error": err.Error()})
		return
	}
	opsJSON(w, http.StatusOK, item)
}

func (a *inspectionAPI) complete(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/inspections/")
	id = strings.TrimSuffix(id, "/complete")
	item, err := a.service.Complete(r.Context(), id)
	if err != nil {
		opsJSON(w, inspectionStatusForError(err), map[string]string{"error": err.Error()})
		return
	}
	a.notifier.Publish(TowerEvent{TowerID: item.TowerID, Type: "completed", Actor: item.Inspector})
	opsJSON(w, http.StatusOK, item)
}

func (a *inspectionAPI) start(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/inspections/")
	id = strings.TrimSuffix(id, "/start")
	item, err := a.service.Start(r.Context(), id)
	if err != nil {
		opsJSON(w, inspectionStatusForError(err), map[string]string{"error": err.Error()})
		return
	}
	a.notifier.Publish(TowerEvent{TowerID: item.TowerID, Type: "started", Actor: item.Inspector})
	opsJSON(w, http.StatusOK, item)
}

func (a *inspectionAPI) audit(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/inspections/")
	id = strings.TrimSuffix(id, "/audit")
	opsJSON(w, http.StatusOK, map[string]any{"events": a.service.Audit(id)})
}

func (a *inspectionAPI) subscribe(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/inspections/")
	id = strings.TrimSuffix(id, "/subscribe")
	ch := a.notifier.Subscribe(id)
	defer a.notifier.Unsubscribe(id, ch)
	select {
	case event := <-ch:
		opsJSON(w, http.StatusOK, event)
	case <-r.Context().Done():
		opsJSON(w, http.StatusGatewayTimeout, map[string]string{"error": "subscription timed out"})
	}
}

func (a *inspectionAPI) routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/inspections", a.list)
	mux.HandleFunc("POST /api/inspections", a.create)
	mux.HandleFunc("GET /api/inspections/{id}", a.get)
	mux.HandleFunc("POST /api/inspections/{id}/start", a.start)
	mux.HandleFunc("POST /api/inspections/{id}/complete", a.complete)
	mux.HandleFunc("GET /api/inspections/{id}/audit", a.audit)
	mux.HandleFunc("GET /api/inspections/{id}/subscribe", a.subscribe)
	return mux
}
