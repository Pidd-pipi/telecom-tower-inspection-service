package main

import (
	"context"
	"example.com/telecom-tower-inspection-service/config"
	"example.com/telecom-tower-inspection-service/httpapi"
	"example.com/telecom-tower-inspection-service/store"
	"example.com/telecom-tower-inspection-service/web"
	"log"
	"net/http"
	"time"
)

func main() {
	address := ":" + config.Port()
	log.Printf("telecom-tower-inspection-service listening on %s", address)
	handler, shutdown := newRootHandler()
	if err := serveAddress(address, handler, shutdown); err != nil {
		log.Fatal(err)
	}
}

// newRootHandler builds the root HTTP handler and returns a shutdown function
// that must be called when the server is stopping, so background tasks (the
// stats ticker) come down cleanly with the process.
func newRootHandler() (http.Handler, func()) {
	seedCounters()
	inspStore := newInspectionStore(seedTowers())
	for _, item := range seedInspections() {
		_ = inspStore.AddInspection(context.Background(), item)
	}
	for _, finding := range seedFindings() {
		_ = inspStore.AddFinding(context.Background(), finding)
	}
	service := newInspectionService(inspStore)
	insp := newInspectionAPI(service)
	extra := newTowerHandlers(service)
	ops := newOpsAPIHandler(newOpsService(seedOpsRecords()))

	// Seed the summary cache and start a low-frequency refresh ticker as a
	// backstop for staleness; data mutations invalidate the cache immediately,
	// so this only matters for paths that bypass Invalidate.
	extra.startBackgroundStats(30 * time.Second)

	mux := http.NewServeMux()
	httpapi.Register(store.New(), web.FS, mux)
	mux.Handle("/api/inspections", insp.routes())
	mux.Handle("/api/inspections/", insp.routes())
	mux.Handle("/api/findings", extra.routes())
	mux.Handle("/api/findings/", extra.routes())
	mux.Handle("/api/workorders", extra.routes())
	mux.Handle("/api/workorders/", extra.routes())
	mux.Handle("/api/risk/", extra.routes())
	mux.Handle("/api/summary", extra.routes())
	mux.Handle("/api/export", extra.routes())
	mux.Handle("/api/report", extra.routes())
	mux.Handle("/ops/", ops)
	mux.Handle("/ops", ops)
	return mux, extra.Shutdown
}
