package main

import (
	"context"
	"sort"
	"sync"
)

type InspectionStore struct {
	mu          sync.RWMutex
	inspections []TowerInspection
	findings    map[string]Finding
	orders      map[string]WorkOrder
	towers      map[string]TowerInfo
}

type TowerInfo struct {
	ID          string
	Region      string
	TowerType   string
	WindLoadPct float64
	Corrosion   int
}

func newInspectionStore(towers []TowerInfo) *InspectionStore {
	s := &InspectionStore{
		inspections: []TowerInspection{},
		findings:    map[string]Finding{},
		orders:      map[string]WorkOrder{},
		towers:      map[string]TowerInfo{},
	}
	for _, t := range towers {
		s.towers[t.ID] = t
	}
	return s
}

func (s *InspectionStore) Tower(id string) (TowerInfo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.towers[id]
	return t, ok
}

func (s *InspectionStore) Towers() []TowerInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]TowerInfo, 0, len(s.towers))
	for _, t := range s.towers {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (s *InspectionStore) AddInspection(ctx context.Context, item TowerInspection) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.inspections {
		if existing.ID == item.ID {
			return ErrOpsConflict
		}
	}
	s.inspections = append(s.inspections, item)
	return nil
}

func (s *InspectionStore) GetInspection(ctx context.Context, id string) (TowerInspection, error) {
	if err := ctx.Err(); err != nil {
		return TowerInspection{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, item := range s.inspections {
		if item.ID == id {
			return item.Clone(), nil
		}
	}
	return TowerInspection{}, ErrOpsNotFound
}

func (s *InspectionStore) ListInspections(ctx context.Context) ([]TowerInspection, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.inspections, nil
}

func (s *InspectionStore) UpdateInspectionStatus(ctx context.Context, id string, status InspectionStatus) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.inspections {
		if s.inspections[i].ID == id {
			s.inspections[i].Status = status
			if status == InspectionCompleted {
				s.inspections[i].CompletedAt = timeNowOps()
			}
			return nil
		}
	}
	return ErrOpsNotFound
}

func (s *InspectionStore) AddFinding(ctx context.Context, finding Finding) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.findings[finding.ID]; ok {
		return ErrOpsConflict
	}
	s.findings[finding.ID] = finding
	return nil
}

func (s *InspectionStore) GetFinding(ctx context.Context, id string) (Finding, error) {
	if err := ctx.Err(); err != nil {
		return Finding{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	f, ok := s.findings[id]
	if !ok {
		return Finding{}, ErrOpsNotFound
	}
	return f, nil
}

func (s *InspectionStore) ListFindings(ctx context.Context) ([]Finding, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Finding, 0, len(s.findings))
	for _, f := range s.findings {
		out = append(out, f)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (s *InspectionStore) AddOrder(ctx context.Context, order WorkOrder) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.orders[order.ID]; ok {
		return ErrOpsConflict
	}
	s.orders[order.ID] = order
	return nil
}

func (s *InspectionStore) GetOrder(ctx context.Context, id string) (WorkOrder, error) {
	if err := ctx.Err(); err != nil {
		return WorkOrder{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	o, ok := s.orders[id]
	if !ok {
		return WorkOrder{}, ErrOpsNotFound
	}
	return o, nil
}

func (s *InspectionStore) ListOrders(ctx context.Context) ([]WorkOrder, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]WorkOrder, 0, len(s.orders))
	for _, o := range s.orders {
		out = append(out, o)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (s *InspectionStore) UpdateOrderStatus(ctx context.Context, id string, status WorkOrderStatus) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.orders[id]
	if !ok {
		return ErrOpsNotFound
	}
	o.Status = status
	if status == WorkOrderResolved {
		o.ResolvedAt = timeNowOps()
	}
	s.orders[id] = o
	return nil
}
