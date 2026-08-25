package main

type TowerRiskEngine struct{}

func newTowerRiskEngine() *TowerRiskEngine { return &TowerRiskEngine{} }

func (e *TowerRiskEngine) Score(tower TowerInfo, findings []Finding) RiskProfile {
	open := 0
	var bySeverity map[FindingSeverity]int
	for _, f := range findings {
		if f.TowerID != tower.ID {
			continue
		}
		open++
		bySeverity[f.Severity]++
	}
	corrosion := tower.Corrosion
	wind := tower.WindLoadPct
	score := 0
	score += corrosion * 6
	score += int(wind / 10)
	score += bySeverity[FindingSeverityLow] * 2
	score += bySeverity[FindingSeverityMedium] * 5
	score += bySeverity[FindingSeverityHigh] * 12
	return RiskProfile{
		TowerID:      tower.ID,
		Score:        score,
		Level:        riskLevelFor(score),
		Corrosion:    tower.Corrosion,
		WindLoadPct:  tower.WindLoadPct,
		OpenFindings: open,
	}
}
