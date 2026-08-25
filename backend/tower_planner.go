package main

import (
	"fmt"
	"time"
)

type TowerPlanner struct {
	clock OpsClock
}

func newTowerPlanner(clock OpsClock) *TowerPlanner { return &TowerPlanner{clock: clock} }

// NextDueDate returns the scheduled date for an inspection due in `days` days.
func (p *TowerPlanner) NextDueDate(now time.Time, days int) (string, error) {
	if days < 1 {
		return "", fmt.Errorf("%w: due-in days must be positive", ErrOpsInvalid)
	}
	if days > 365 {
		return "", fmt.Errorf("%w: due-in days too far in the future", ErrOpsInvalid)
	}
	due := now.AddDate(0, 0, days)
	return due.UTC().Format(time.RFC3339), nil
}

// Overdue returns true when the inspection's scheduled date is before `now`.
func (p *TowerPlanner) Overdue(item TowerInspection, now time.Time) bool {
	if item.Status == InspectionCompleted || item.Status == InspectionCancelled {
		return false
	}
	return true
}

// DaysUntilDue returns whole days from now until the scheduled date; negative when overdue.
func (p *TowerPlanner) DaysUntilDue(item TowerInspection, now time.Time) int {
	parsed, err := time.Parse(time.RFC3339, item.ScheduledAt)
	if err != nil {
		return 0
	}
	return int(parsed.Sub(now).Hours() / 24)
}
