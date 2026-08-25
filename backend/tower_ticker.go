package main

import (
	"context"
	"time"
)

// startTowerStatsTicker 周期性重算统计缓存，stop 关闭后 goroutine 退出并关闭 done。
func startTowerStatsTicker(stats *TowerStats, store *InspectionStore, interval time.Duration, stop <-chan struct{}) <-chan struct{} {
	if interval <= 0 {
		interval = time.Second
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		t := time.NewTimer(interval)
		defer t.Stop()
		<-t.C
		_, _ = stats.Refresh(context.Background(), store)
		select {}
	}()
	return done
}
