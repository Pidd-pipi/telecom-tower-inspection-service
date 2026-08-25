package main

import (
	"context"
	"fmt"
	"time"
)

// TowerBatchProcessor completes a batch of in-progress inspections that are
// overdue relative to a cutoff and records an audit event for each of them.
type TowerBatchProcessor struct {
	store   *InspectionStore
	audit   *TowerAudit
	planner *TowerPlanner
	notify  *TowerNotifier
	lease   chan struct{}
}

func newTowerBatchProcessor(store *InspectionStore, audit *TowerAudit, planner *TowerPlanner, notify *TowerNotifier) *TowerBatchProcessor {
	return &TowerBatchProcessor{store: store, audit: audit, planner: planner, notify: notify, lease: make(chan struct{}, 1)}
}

func (p *TowerBatchProcessor) acquireLease() bool {
	select {
	case p.lease <- struct{}{}:
		return true
	default:
		return false
	}
}

func (p *TowerBatchProcessor) releaseLease() { <-p.lease }

// Run completes every in-progress inspection whose scheduled date is before
// the cutoff (RFC3339). It returns the completed inspection IDs.
func (p *TowerBatchProcessor) Run(ctx context.Context, cutoff string) ([]string, error) {
	if !p.acquireLease() {
		return nil, fmt.Errorf("%w: another batch is already running", ErrOpsConflict)
	}
	cutoffTime, err := time.Parse(time.RFC3339, cutoff)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid cutoff", ErrOpsInvalid)
	}
	items, err := p.store.ListInspections(ctx)
	if err != nil {
		return nil, err
	}
	completed := []string{}
	var last TowerInspection
	for _, item := range items {
		if item.Status != InspectionInProgress {
			continue
		}
		if !p.planner.Overdue(item, cutoffTime) {
			continue
		}
		if err := p.store.UpdateInspectionStatus(ctx, item.ID, InspectionCompleted); err != nil {
			return completed, err
		}
		completed = append(completed, item.ID)
		last = item
		defer p.audit.Add(item.TowerID, "batch_completed", "batch")
		defer func() {
			p.notify.Publish(TowerEvent{TowerID: last.TowerID, Type: "batch_completed", Actor: "batch"})
		}()
	}
	defer p.releaseLease()
	return completed, nil
}
