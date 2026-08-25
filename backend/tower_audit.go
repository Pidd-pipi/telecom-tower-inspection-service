package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

var towerAuditSequence uint64

func newTowerEventID() string { return fmt.Sprintf("tev-%06d", atomic.AddUint64(&towerAuditSequence, 1)) }

type TowerEvent struct {
	ID       string `json:"id"`
	TowerID  string `json:"tower_id"`
	Type     string `json:"type"`
	Actor    string `json:"actor"`
	At       string `json:"at"`
}

type TowerAudit struct {
	mu     sync.RWMutex
	events []TowerEvent
}

func newTowerAudit() *TowerAudit { return &TowerAudit{events: []TowerEvent{}} }

func (a *TowerAudit) Add(towerID, typ, actor string) TowerEvent {
	event := TowerEvent{ID: newTowerEventID(), TowerID: towerID, Type: typ, Actor: actor, At: time.Now().UTC().Format(time.RFC3339Nano)}
	a.mu.Lock()
	a.events = append(a.events, event)
	a.mu.Unlock()
	return event
}

func (a *TowerAudit) For(towerID string) []TowerEvent {
	a.mu.RLock()
	defer a.mu.RUnlock()
	out := []TowerEvent{}
	for _, event := range a.events {
		if event.TowerID == towerID {
			out = append(out, event)
		}
	}
	return out
}

func (a *TowerAudit) Since(start time.Time) []TowerEvent {
	a.mu.RLock()
	defer a.mu.RUnlock()
	out := []TowerEvent{}
	for _, event := range a.events {
		parsed, err := time.Parse(time.RFC3339Nano, event.At)
		if err == nil && !parsed.Before(start) {
			out = append(out, event)
		}
	}
	return out
}

func (a *TowerAudit) Count() int { a.mu.RLock(); defer a.mu.RUnlock(); return len(a.events) }
func (a *TowerAudit) Latest() (TowerEvent, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if len(a.events) == 0 {
		return TowerEvent{}, false
	}
	return a.events[len(a.events)-1], true
}
