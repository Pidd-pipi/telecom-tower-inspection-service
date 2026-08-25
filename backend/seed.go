package main

func seedTowers() []TowerInfo {
	return []TowerInfo{
		{ID: "TWR-118", Region: "Kanto North", TowerType: "lattice", WindLoadPct: 34, Corrosion: 2},
		{ID: "TWR-204", Region: "Kanto Coast", TowerType: "monopole", WindLoadPct: 58, Corrosion: 4},
		{ID: "TWR-301", Region: "Chubu Hills", TowerType: "guyed", WindLoadPct: 71, Corrosion: 6},
		{ID: "TWR-405", Region: "Kyushu South", TowerType: "lattice", WindLoadPct: 43, Corrosion: 3},
	}
}

func seedInspections() []TowerInspection {
	return []TowerInspection{
		{ID: "ins-0001", TowerID: "TWR-118", Region: "Kanto North", Inspector: "hanako", ScheduledAt: "2026-09-01T09:00:00Z", Status: InspectionScheduled, FindingIDs: []string{}},
		{ID: "ins-0002", TowerID: "TWR-204", Region: "Kanto Coast", Inspector: "taro", ScheduledAt: "2026-08-28T09:00:00Z", Status: InspectionInProgress, FindingIDs: []string{}},
		{ID: "ins-0003", TowerID: "TWR-301", Region: "Chubu Hills", Inspector: "yuki", ScheduledAt: "2026-08-20T09:00:00Z", CompletedAt: "2026-08-21T15:00:00Z", Status: InspectionCompleted, FindingIDs: []string{"fnd-0001"}},
		{ID: "ins-0004", TowerID: "TWR-405", Region: "Kyushu South", Inspector: "sora", ScheduledAt: "2026-09-05T09:00:00Z", Status: InspectionScheduled, FindingIDs: []string{}},
	}
}

func seedFindings() []Finding {
	return []Finding{
		{ID: "fnd-0001", TowerID: "TWR-301", InspectionID: "ins-0003", Severity: FindingSeverityHigh, Category: "corrosion", Detail: "anchor bolt corrosion at base", CreatedAt: "2026-08-21T14:30:00Z"},
	}
}

func seedCounters() {
	inspectionSequence = 4
	findingSequence = 1
	orderSequence = 0
}

func seedOpsRecords() []OpsRecord {
	return []OpsRecord{
		{ID: "ops-0001", Subject: "Tower TWR-204 repair dispatch", Owner: "ops-lead", Status: OpsStatusActive, Priority: OpsPriorityHigh, Revision: 1, Labels: map[string]string{"site": "Kanto Coast"}},
		{ID: "ops-0002", Subject: "Tower TWR-301 follow-up survey", Owner: "ops-lead", Status: OpsStatusQueued, Priority: OpsPriorityNormal, Revision: 1, Labels: map[string]string{"site": "Chubu Hills"}},
	}
}
