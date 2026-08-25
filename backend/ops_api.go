package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type opsAPIHandler struct {
	service *OpsService
}

func newOpsAPIHandler(service *OpsService) *opsAPIHandler {
	return &opsAPIHandler{service: service}
}

func opsStatusForError(err error) int {
	switch opsCode(err) {
	case "not_found":
		return http.StatusNotFound
	case "conflict":
		return http.StatusConflict
	case "transition":
		return http.StatusConflict
	case "invalid":
		return http.StatusBadRequest
	case "policy":
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}

func (h *opsAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, "/ops/") {
		opsJSON(w, http.StatusNotFound, map[string]string{"error": "route not found"})
		return
	}
	switch {
	case r.URL.Path == "/ops/records" && r.Method == http.MethodGet:
		h.list(w, r)
	case r.URL.Path == "/ops/records" && r.Method == http.MethodPost:
		h.create(w, r)
	case strings.HasPrefix(r.URL.Path, "/ops/records/") && strings.HasSuffix(r.URL.Path, "/transition"):
		h.transition(w, r)
	case strings.HasPrefix(r.URL.Path, "/ops/records/") && strings.HasSuffix(r.URL.Path, "/audit"):
		h.audit(w, r)
	case strings.HasPrefix(r.URL.Path, "/ops/records/"):
		h.get(w, r)
	case r.URL.Path == "/ops/snapshot" && r.Method == http.MethodGet:
		h.snapshot(w, r)
	case r.URL.Path == "/ops/rules" && r.Method == http.MethodGet:
		h.rules(w, r)
	default:
		opsJSON(w, http.StatusNotFound, map[string]string{"error": "route not found"})
	}
}

func (h *opsAPIHandler) list(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	q := OpsQuery{
		Subject:  query.Get("subject"),
		Status:   OpsStatus(query.Get("status")),
		Priority: OpsPriority(query.Get("priority")),
		Owner:    query.Get("owner"),
	}
	if page, err := strconv.Atoi(query.Get("page")); err == nil {
		q.Page = page
	}
	if size, err := strconv.Atoi(query.Get("page_size")); err == nil {
		q.PageSize = size
	}
	page, err := h.service.Search(r.Context(), q)
	if err != nil {
		opsJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	opsJSON(w, http.StatusOK, page)
}

type createOpsRequest struct {
	ID       string            `json:"id"`
	Subject  string            `json:"subject"`
	Owner    string            `json:"owner"`
	Status   OpsStatus         `json:"status"`
	Priority OpsPriority       `json:"priority"`
	Labels   map[string]string `json:"labels"`
}

func (h *opsAPIHandler) create(w http.ResponseWriter, r *http.Request) {
	var req createOpsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		opsJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	record := OpsRecord{
		ID:       req.ID,
		Subject:  req.Subject,
		Owner:    req.Owner,
		Status:   req.Status,
		Priority: req.Priority,
		Labels:   req.Labels,
	}
	if record.ID == "" {
		record.ID = fmt.Sprintf("ops-%03d", opsCreateSequence())
	}
	created, err := h.service.Create(r.Context(), record)
	if err != nil {
		opsJSON(w, opsStatusForError(err), map[string]string{"error": err.Error()})
		return
	}
	opsJSON(w, http.StatusCreated, created)
}

type transitionOpsRequest struct {
	Expected int       `json:"expected"`
	Target   OpsStatus `json:"target"`
	Actor    string    `json:"actor"`
}

func (h *opsAPIHandler) transition(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		opsJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/ops/records/")
	id = strings.TrimSuffix(id, "/transition")
	if id == "" {
		opsJSON(w, http.StatusNotFound, map[string]string{"error": "route not found"})
		return
	}
	var req transitionOpsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		opsJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	if !opsStatusValid(req.Target) {
		opsJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid target status"})
		return
	}
	record, err := h.service.Transition(context.Background(), id, req.Expected, req.Target, req.Actor)
	if err != nil {
		opsJSON(w, opsStatusForError(err), map[string]string{"error": err.Error()})
		return
	}
	opsJSON(w, http.StatusOK, record)
}

func (h *opsAPIHandler) get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		opsJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/ops/records/")
	if id == "" {
		opsJSON(w, http.StatusNotFound, map[string]string{"error": "route not found"})
		return
	}
	record, err := h.service.Get(r.Context(), id)
	if err != nil {
		opsJSON(w, opsStatusForError(err), map[string]string{"error": err.Error()})
		return
	}
	opsJSON(w, http.StatusOK, record)
}

func (h *opsAPIHandler) audit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		opsJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/ops/records/")
	id = strings.TrimSuffix(id, "/audit")
	opsJSON(w, http.StatusOK, map[string]any{"events": h.service.Audit(id)})
}

func (h *opsAPIHandler) snapshot(w http.ResponseWriter, r *http.Request) {
	opsJSON(w, http.StatusOK, h.service.Snapshot())
}

func (h *opsAPIHandler) rules(w http.ResponseWriter, r *http.Request) {
	opsJSON(w, http.StatusOK, map[string]any{"rules": opsRules()})
}
