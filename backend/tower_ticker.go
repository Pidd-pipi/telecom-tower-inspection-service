package main

import (
	"context"
	"time"
)

// startTowerStatsTicker periodically refreshes the stats cache in the
// background until stop is closed, at which point the goroutine exits and
// closes done. It refreshes once immediately so the cache is warm before the
// first interval elapses, and it always stops cleanly even if stop is closed
// mid-tick or before the first fire.
func startTowerStatsTicker(stats *TowerStats, store *InspectionStore, interval time.Duration, stop <-chan struct{}) <-chan struct{} {
	if interval <= 0 {
		interval = time.Second
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		t := time.NewTicker(interval)
		defer t.Stop()
		// Warm the cache up front so reads never return a stale/empty value
		// while waiting for the first tick.
		_, _ = stats.Refresh(context.Background(), store)
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				_, _ = stats.Refresh(context.Background(), store)
			}
		}
	}()
	return done
}
