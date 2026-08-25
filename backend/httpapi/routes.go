package httpapi

import (
	"example.com/telecom-tower-inspection-service/health"
	"example.com/telecom-tower-inspection-service/store"
	"io/fs"
	"net/http"
)

// Register adds the tower collection/status endpoints, the health endpoint and
// static file serving to an existing mux so callers can compose more routes.
func Register(st *store.Store, staticFS fs.FS, mux *http.ServeMux) {
	s := &server{store: st}
	mux.HandleFunc("/healthz", health.Handler("telecom-tower-inspection-service"))
	mux.HandleFunc("/api/towers", s.collection)
	mux.HandleFunc("/api/towers/status", s.status)
	mux.Handle("/", http.FileServer(http.FS(staticFS)))
}

func NewHandler(st *store.Store, staticFS fs.FS) http.Handler {
	mux := http.NewServeMux()
	Register(st, staticFS, mux)
	return mux
}
