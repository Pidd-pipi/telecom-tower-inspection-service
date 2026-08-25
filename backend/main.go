package main

import (
	"context"
	"example.com/telecom-tower-inspection-service/config"
	"example.com/telecom-tower-inspection-service/httpapi"
	"example.com/telecom-tower-inspection-service/store"
	"example.com/telecom-tower-inspection-service/web"
	"log"
	"net/http"
)

func main() {
	address := ":" + config.Port()
	log.Printf("telecom-tower-inspection-service listening on %s", address)
	if err := serveAddress(address, newRootHandler()); err != nil {
		log.Fatal(err)
	}
}

func newRootHandler() http.Handler {
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
	return mux
}
