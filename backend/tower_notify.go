package main

import "sync"

type TowerNotifier struct {
	mu          sync.Mutex
	subscribers map[string][]chan TowerEvent
}

func newTowerNotifier() *TowerNotifier {
	return &TowerNotifier{subscribers: map[string][]chan TowerEvent{}}
}

func (n *TowerNotifier) Subscribe(towerID string) <-chan TowerEvent {
	ch := make(chan TowerEvent, 8)
	n.mu.Lock()
	defer n.mu.Unlock()
	n.subscribers[towerID] = append(n.subscribers[towerID], ch)
	return ch
}

func (n *TowerNotifier) Unsubscribe(towerID string, ch <-chan TowerEvent) {
	n.mu.Lock()
	defer n.mu.Unlock()
	subs := n.subscribers[towerID]
	for i, candidate := range subs {
		if candidate == ch {
			n.subscribers[towerID] = append(subs[:i], subs[i+1:]...)
			close(candidate)
			return
		}
	}
}

func (n *TowerNotifier) Publish(event TowerEvent) {
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, ch := range n.subscribers[event.TowerID] {
		select {
		case ch <- event:
		default:
		}
	}
}

func (n *TowerNotifier) SubscriberCount(towerID string) int {
	n.mu.Lock()
	defer n.mu.Unlock()
	return len(n.subscribers[towerID])
}

func (n *TowerNotifier) TotalSubscribers() int {
	n.mu.Lock()
	defer n.mu.Unlock()
	total := 0
	for _, subs := range n.subscribers {
		total += len(subs)
	}
	return total
}
