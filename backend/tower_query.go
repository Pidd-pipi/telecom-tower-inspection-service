package main

import "strings"

type InspectionQuery struct {
	TowerID  string
	Region   string
	Status   InspectionStatus
	Severity FindingSeverity
	Page     int
	PageSize int
}

type InspectionPage struct {
	Items    []TowerInspection `json:"items"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
	Total    int               `json:"total"`
	HasNext  bool              `json:"has_next"`
}

func inspectionMatches(item TowerInspection, q InspectionQuery) bool {
	if q.TowerID != "" && item.TowerID != q.TowerID {
		return false
	}
	if q.Region != "" && !strings.EqualFold(item.Region, q.Region) {
		return false
	}
	if q.Status != "" && item.Status != q.Status {
		return false
	}
	return true
}

func filterInspections(items []TowerInspection, q InspectionQuery) []TowerInspection {
	out := make([]TowerInspection, 0, len(items))
	for _, item := range items {
		if inspectionMatches(item, q) {
			out = append(out, item)
		}
	}
	return out
}

func findingMatches(f Finding, q InspectionQuery) bool {
	if q.TowerID != "" && f.TowerID != q.TowerID {
		return false
	}
	if q.Severity != "" && f.Severity != q.Severity {
		return false
	}
	return true
}

func filterFindings(items []Finding, q InspectionQuery) []Finding {
	out := make([]Finding, 0, len(items))
	for _, item := range items {
		if findingMatches(item, q) {
			out = append(out, item)
		}
	}
	return out
}

func towerQueryDefaults(q InspectionQuery) InspectionQuery {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 {
		q.PageSize = 25
	}
	if q.PageSize > 200 {
		q.PageSize = 200
	}
	return q
}

func towerBounds(total, page, size int) (int, int) {
	q := towerQueryDefaults(InspectionQuery{Page: page, PageSize: size})
	start := (q.Page - 1) * q.PageSize
	if start < 0 {
		start = 0
	}
	if start > total {
		start = total
	}
	end := start + q.PageSize
	if end > total {
		end = total
	}
	return start, end
}

func towerPage(items []TowerInspection, q InspectionQuery) InspectionPage {
	filtered := filterInspections(items, q)
	q = towerQueryDefaults(q)
	start, end := towerBounds(len(filtered), q.Page, q.PageSize)
	return InspectionPage{Items: filtered[start:end], Page: q.Page, PageSize: q.PageSize, Total: len(filtered), HasNext: end < len(filtered)}
}
