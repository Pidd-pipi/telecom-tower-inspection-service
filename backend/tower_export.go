package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

var ErrExportPoolExhausted = errors.New("inspection export lease pool exhausted")

const inspectionExportLeaseLimit = 4

type InspectionExporter struct {
	store  *InspectionStore
	audit  *TowerAudit
	tokens chan struct{}
}

func newInspectionExporter(store *InspectionStore, audit *TowerAudit) *InspectionExporter {
	return &InspectionExporter{store: store, audit: audit, tokens: make(chan struct{}, inspectionExportLeaseLimit)}
}

func (e *InspectionExporter) acquire() (release func(), ok bool) {
	select {
	case e.tokens <- struct{}{}:
		var once bool
		return func() {
			if !once {
				once = true
				<-e.tokens
			}
		}, true
	default:
		return nil, false
	}
}

func (e *InspectionExporter) finalize(ids []string) error {
	for _, id := range ids {
		e.audit.Add(id, "exported", "system")
	}
	return nil
}

// Export moves every inspection currently in `from` status into `to` status.
func (e *InspectionExporter) Export(ctx context.Context, from, to InspectionStatus) (ExportResult, error) {
	if !inspectionStatusValid(from) || !inspectionStatusValid(to) || from == to {
		return ExportResult{}, fmt.Errorf("%w: from %s to %s", ErrOpsInvalid, from, to)
	}
	items, err := e.store.ListInspections(ctx)
	if err != nil {
		return ExportResult{}, err
	}
	result := ExportResult{IDs: []string{}}
	ids := make(chan string, len(items))
	errCh := make(chan error, 1)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for _, item := range items {
		if item.Status != from {
			result.Skipped++
			continue
		}
		wg.Add(1)
		go func(item TowerInspection) {
			defer wg.Done()
			<-start
			release, ok := e.acquire()
			if !ok {
				select {
				case errCh <- fmt.Errorf("%w after %d windows", ErrExportPoolExhausted, result.Exported):
				default:
				}
				return
			}
			defer release()
			if err := e.store.UpdateInspectionStatus(ctx, item.ID, to); err != nil {
				select {
				case errCh <- err:
				default:
				}
				return
			}
			ids <- item.ID
		}(item)
	}
	close(start)
	wg.Wait()
	close(ids)
	for id := range ids {
		result.IDs = append(result.IDs, id)
	}
	result.Exported = len(result.IDs)
	if err := e.finalize(result.IDs); err != nil {
		return result, err
	}
	return result, nil
}

type ExportResult struct {
	Exported int      `json:"exported"`
	Skipped  int      `json:"skipped"`
	IDs      []string `json:"ids"`
}

func inspectionStatusValid(status InspectionStatus) bool {
	switch status {
	case InspectionScheduled, InspectionInProgress, InspectionCompleted, InspectionCancelled:
		return true
	default:
		return false
	}
}
