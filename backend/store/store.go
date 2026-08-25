package store

import (
	"errors"
	"example.com/telecom-tower-inspection-service/domain"
	"sync"
)

var ErrNotFound = errors.New("tower not found")

type Store struct {
	mu    sync.RWMutex
	items []domain.Tower
}

func New() *Store {
	return &Store{items: []domain.Tower{{ID: "TWR-118", Region: "Kanto North", TowerType: "lattice", LastInspection: "2026-08-12", WindLoadPct: 34, CorrosionScore: 2, Status: "clear", UpdatedAt: "2026-08-21T08:40:00Z"}, {ID: "TWR-204", Region: "Kanto Coast", TowerType: "monopole", LastInspection: "2026-07-29", WindLoadPct: 58, CorrosionScore: 4, Status: "review", UpdatedAt: "2026-08-21T08:35:00Z"}}}
}
func (s *Store) List() []domain.Tower {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]domain.Tower, len(s.items))
	copy(result, s.items)
	return result
}
func (s *Store) UpdateStatus(id, status, updatedAt string) (domain.Tower, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.items {
		if s.items[i].ID == id {
			s.items[i].Status, s.items[i].UpdatedAt = status, updatedAt
			return s.items[i], nil
		}
	}
	return domain.Tower{}, ErrNotFound
}
