package main

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
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

func (e *InspectionExporter) acquire(ctx context.Context) (release func(), ok bool) {
	select {
	case e.tokens <- struct{}{}:
		var once bool
		return func() {
			if !once {
				once = true
				<-e.tokens
			}
		}, true
	case <-ctx.Done():
		return nil, false
	default:
		return nil, false
	}
}

// recordAudit appends a single audit event per exported inspection. Callers
// must not also call finalize-style helpers that would double-write.
func (e *InspectionExporter) recordAudit(id string) {
	e.audit.Add(id, "exported", "system")
}

// Export moves every inspection currently in `from` status into `to` status.
//
// It is safe to call with hundreds of eligible inspections and under a short
// request-scoped deadline: worker goroutines never block on unbuffered error
// channels, the context is honored at every step, and each exported
// inspection is recorded in the audit log exactly once.
func (e *InspectionExporter) Export(ctx context.Context, from, to InspectionStatus) (ExportResult, error) {
	if !inspectionStatusValid(from) || !inspectionStatusValid(to) || from == to {
		return ExportResult{}, fmt.Errorf("%w: from %s to %s", ErrOpsInvalid, from, to)
	}
	items, err := e.store.ListInspections(ctx)
	if err != nil {
		return ExportResult{}, err
	}

	result := ExportResult{IDs: []string{}}
	// Collect exported IDs and the first error under a mutex so worker
	// goroutines never block on a channel nobody reads (which previously
	// deadlocked the whole export once any worker errored or the context
	// expired mid-run).
	var (
		mu        sync.Mutex
		firstErr  error
		once      sync.Once
		exported  int32
		skipped   int32
	)

	// errCh is buffered to the number of workers so a worker reporting an
	// error never blocks waiting for a reader; it is drained after the
	// workers join.
	errCh := make(chan error, len(items)+1)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for _, item := range items {
		item := item
		if item.Status != from {
			atomic.AddInt32(&skipped, 1)
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			release, ok := e.acquire(ctx)
			if !ok {
				errCh <- fmt.Errorf("%w after %d windows", ErrExportPoolExhausted, atomic.LoadInt32(&exported))
				return
			}
			defer release()
			if err := e.store.UpdateInspectionStatus(ctx, item.ID, to); err != nil {
				errCh <- err
				return
			}
			e.recordAudit(item.ID)
			mu.Lock()
			result.IDs = append(result.IDs, item.ID)
			mu.Unlock()
			atomic.AddInt32(&exported, 1)
		}()
	}
	close(start)
	wg.Wait()
	close(errCh)
	for e := range errCh {
		once.Do(func() { firstErr = e })
	}

	result.Exported = int(atomic.LoadInt32(&exported))
	result.Skipped = int(atomic.LoadInt32(&skipped))
	// Sort IDs for deterministic output regardless of goroutine scheduling.
	sort.Strings(result.IDs)

	if firstErr != nil {
		return result, firstErr
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
