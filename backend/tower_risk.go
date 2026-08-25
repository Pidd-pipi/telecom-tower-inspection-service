package main

type TowerRiskEngine struct{}

func newTowerRiskEngine() *TowerRiskEngine { return &TowerRiskEngine{} }

// Score computes a 0-100 risk profile for a tower from its open findings.
// Only findings belonging to this tower are counted, and the total is capped at 100.
func (e *TowerRiskEngine) Score(tower TowerInfo, findings []Finding) RiskProfile {
	open := 0
	bySeverity := map[FindingSeverity]int{}
	for _, f := range findings {
		if f.TowerID != tower.ID {
			continue
		}
		open++
		bySeverity[f.Severity]++
	}
	corrosion := tower.Corrosion
	if corrosion < 0 {
		corrosion = 0
	}
	if corrosion > 10 {
		corrosion = 10
	}
	wind := tower.WindLoadPct
	if wind < 0 {
		wind = 0
	}
	if wind > 100 {
		wind = 100
	}
	score := 0
	score += corrosion * 6
	score += int(wind / 10)
	score += bySeverity[FindingSeverityLow] * 2
	score += bySeverity[FindingSeverityMedium] * 5
	score += bySeverity[FindingSeverityHigh] * 12
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}
	return RiskProfile{
		TowerID:      tower.ID,
		Score:        score,
		Level:        riskLevelFor(score),
		Corrosion:    tower.Corrosion,
		WindLoadPct:  tower.WindLoadPct,
		OpenFindings: open,
	}
}
