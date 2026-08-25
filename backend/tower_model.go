package main

import (
	"fmt"
	"sync/atomic"
)

type InspectionStatus string

const (
	InspectionScheduled  InspectionStatus = "scheduled"
	InspectionInProgress InspectionStatus = "in_progress"
	InspectionCompleted  InspectionStatus = "completed"
	InspectionCancelled  InspectionStatus = "cancelled"
)

type FindingSeverity string

const (
	FindingSeverityLow    FindingSeverity = "low"
	FindingSeverityMedium FindingSeverity = "medium"
	FindingSeverityHigh   FindingSeverity = "high"
)

type WorkOrderStatus string

const (
	WorkOrderOpen     WorkOrderStatus = "open"
	WorkOrderAssigned WorkOrderStatus = "assigned"
	WorkOrderResolved WorkOrderStatus = "resolved"
)

type TowerInspection struct {
	ID          string            `json:"id"`
	TowerID     string            `json:"tower_id"`
	Region      string            `json:"region"`
	Inspector   string            `json:"inspector"`
	ScheduledAt string            `json:"scheduled_at"`
	CompletedAt string            `json:"completed_at,omitempty"`
	Status      InspectionStatus  `json:"status"`
	FindingIDs  []string          `json:"finding_ids,omitempty"`
}

func (t TowerInspection) Clone() TowerInspection {
	out := t
	out.FindingIDs = append([]string(nil), t.FindingIDs...)
	return out
}

type Finding struct {
	ID           string          `json:"id"`
	TowerID      string          `json:"tower_id"`
	InspectionID string          `json:"inspection_id,omitempty"`
	Severity     FindingSeverity `json:"severity"`
	Category     string          `json:"category"`
	Detail       string          `json:"detail"`
	CreatedAt    string          `json:"created_at"`
}

type WorkOrder struct {
	ID         string          `json:"id"`
	FindingID  string          `json:"finding_id"`
	TowerID    string          `json:"tower_id"`
	Priority   OpsPriority     `json:"priority"`
	Status     WorkOrderStatus `json:"status"`
	Assignee   string          `json:"assignee,omitempty"`
	CreatedAt  string          `json:"created_at"`
	ResolvedAt string          `json:"resolved_at,omitempty"`
}

type RiskProfile struct {
	TowerID      string  `json:"tower_id"`
	Score        int     `json:"score"`
	Level        string  `json:"level"`
	Corrosion    int     `json:"corrosion"`
	WindLoadPct  float64 `json:"wind_load_pct"`
	OpenFindings int     `json:"open_findings"`
}

func riskLevelFor(score int) string {
	switch {
	case score >= 80:
		return "critical"
	case score >= 60:
		return "high"
	case score >= 40:
		return "medium"
	default:
		return "low"
	}
}

var inspectionSequence, findingSequence, orderSequence uint64

func newInspectionID() string { return fmt.Sprintf("ins-%04d", atomic.AddUint64(&inspectionSequence, 1)) }
func newFindingID() string    { return fmt.Sprintf("fnd-%04d", atomic.AddUint64(&findingSequence, 1)) }
func newOrderID() string      { return fmt.Sprintf("ord-%04d", atomic.AddUint64(&orderSequence, 1)) }
