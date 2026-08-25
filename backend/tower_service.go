package main

import (
	"context"
	"fmt"
	"strings"
)

type InspectionService struct {
	store   *InspectionStore
	audit   *TowerAudit
	planner *TowerPlanner
	risk    *TowerRiskEngine
	clock   OpsClock
}

func newInspectionService(store *InspectionStore) *InspectionService {
	return &InspectionService{
		store:   store,
		audit:   newTowerAudit(),
		planner: newTowerPlanner(newOpsClock()),
		risk:    newTowerRiskEngine(),
		clock:   newOpsClock(),
	}
}

type ScheduleRequest struct {
	TowerID   string `json:"tower_id"`
	Inspector string `json:"inspector"`
	Region    string `json:"region"`
	DueInDays int    `json:"due_in_days"`
}

func (s *InspectionService) Schedule(ctx context.Context, req ScheduleRequest) (TowerInspection, error) {
	if strings.TrimSpace(req.TowerID) == "" || strings.TrimSpace(req.Inspector) == "" {
		return TowerInspection{}, fmt.Errorf("%w: tower id and inspector are required", ErrOpsInvalid)
	}
	if _, ok := s.store.Tower(req.TowerID); !ok {
		return TowerInspection{}, fmt.Errorf("%w: unknown tower %s", ErrOpsNotFound, req.TowerID)
	}
	due, err := s.planner.NextDueDate(s.clock.Now(), req.DueInDays)
	if err != nil {
		return TowerInspection{}, err
	}
	item := TowerInspection{
		ID:          newInspectionID(),
		TowerID:     req.TowerID,
		Region:      req.Region,
		Inspector:   req.Inspector,
		ScheduledAt: due,
		Status:      InspectionScheduled,
		FindingIDs:  []string{},
	}
	if err := s.store.AddInspection(ctx, item); err != nil {
		return TowerInspection{}, err
	}
	s.audit.Add(item.ID, "scheduled", req.Inspector)
	return item, nil
}

func (s *InspectionService) Start(ctx context.Context, id string) (TowerInspection, error) {
	item, err := s.store.GetInspection(ctx, id)
	if err != nil {
		return TowerInspection{}, err
	}
	if item.Status != InspectionScheduled {
		return TowerInspection{}, fmt.Errorf("%w: cannot start a %s inspection", ErrOpsTransition, item.Status)
	}
	if err := s.store.UpdateInspectionStatus(ctx, id, InspectionInProgress); err != nil {
		return TowerInspection{}, err
	}
	s.audit.Add(id, "started", item.Inspector)
	item.Status = InspectionInProgress
	return item, nil
}

func (s *InspectionService) Complete(ctx context.Context, id string) (TowerInspection, error) {
	item, err := s.store.GetInspection(ctx, id)
	if err != nil {
		return TowerInspection{}, err
	}
	if item.Status != InspectionInProgress {
		return TowerInspection{}, fmt.Errorf("%w: cannot complete a %s inspection", ErrOpsTransition, item.Status)
	}
	if err := s.store.UpdateInspectionStatus(ctx, id, InspectionCompleted); err != nil {
		return TowerInspection{}, err
	}
	s.audit.Add(id, "completed", item.Inspector)
	item.Status = InspectionCompleted
	return item, nil
}

type RecordFindingRequest struct {
	TowerID      string         `json:"tower_id"`
	InspectionID string         `json:"inspection_id"`
	Severity     FindingSeverity `json:"severity"`
	Category     string         `json:"category"`
	Detail       string         `json:"detail"`
}

func (s *InspectionService) RecordFinding(ctx context.Context, req RecordFindingRequest) (Finding, error) {
	if strings.TrimSpace(req.TowerID) == "" {
		return Finding{}, fmt.Errorf("%w: tower id required", ErrOpsInvalid)
	}
	if !findingSeverityValid(req.Severity) {
		return Finding{}, fmt.Errorf("%w: severity must be low, medium, or high", ErrOpsInvalid)
	}
	finding := Finding{
		ID:           newFindingID(),
		TowerID:      req.TowerID,
		InspectionID: req.InspectionID,
		Severity:     req.Severity,
		Category:     req.Category,
		Detail:       req.Detail,
		CreatedAt:    s.clock.Stamp(),
	}
	if err := s.store.AddFinding(ctx, finding); err != nil {
		return Finding{}, err
	}
	s.audit.Add(req.TowerID, "finding_recorded", "inspector")
	if req.Severity == FindingSeverityHigh {
		order := WorkOrder{
			ID:        newOrderID(),
			FindingID: finding.ID,
			TowerID:   req.TowerID,
			Priority:  OpsPriorityHigh,
			Status:    WorkOrderOpen,
			CreatedAt: s.clock.Stamp(),
		}
		if err := s.store.AddOrder(ctx, order); err != nil {
			return Finding{}, err
		}
		s.audit.Add(req.TowerID, "order_generated", "system")
	}
	return finding, nil
}

func (s *InspectionService) AssignOrder(ctx context.Context, id, assignee string) (WorkOrder, error) {
	order, err := s.store.GetOrder(ctx, id)
	if err != nil {
		return WorkOrder{}, err
	}
	if order.Status != WorkOrderOpen {
		return WorkOrder{}, fmt.Errorf("%w: order is %s", ErrOpsTransition, order.Status)
	}
	if err := s.store.UpdateOrderStatus(ctx, id, WorkOrderAssigned); err != nil {
		return WorkOrder{}, err
	}
	order.Status = WorkOrderAssigned
	order.Assignee = assignee
	s.audit.Add(order.TowerID, "order_assigned", assignee)
	return order, nil
}

func (s *InspectionService) ResolveOrder(ctx context.Context, id string) (WorkOrder, error) {
	order, err := s.store.GetOrder(ctx, id)
	if err != nil {
		return WorkOrder{}, err
	}
	if order.Status != WorkOrderAssigned {
		return WorkOrder{}, fmt.Errorf("%w: order is %s", ErrOpsTransition, order.Status)
	}
	if err := s.store.UpdateOrderStatus(ctx, id, WorkOrderResolved); err != nil {
		return WorkOrder{}, err
	}
	order.Status = WorkOrderResolved
	s.audit.Add(order.TowerID, "order_resolved", order.Assignee)
	return order, nil
}

func (s *InspectionService) Risk(ctx context.Context, towerID string) (RiskProfile, error) {
	tower, ok := s.store.Tower(towerID)
	if !ok {
		return RiskProfile{}, fmt.Errorf("%w: unknown tower %s", ErrOpsNotFound, towerID)
	}
	findings, err := s.store.ListFindings(ctx)
	if err != nil {
		return RiskProfile{}, err
	}
	return s.risk.Score(tower, findings), nil
}

func (s *InspectionService) Audit(id string) []TowerEvent { return s.audit.For(id) }
func (s *InspectionService) Store() *InspectionStore      { return s.store }

// findingSeverityValid reports whether a finding severity is one of the allowed values.
func findingSeverityValid(s FindingSeverity) bool {
	switch s {
	case FindingSeverityLow, FindingSeverityMedium, FindingSeverityHigh:
		return true
	default:
		return false
	}
}
