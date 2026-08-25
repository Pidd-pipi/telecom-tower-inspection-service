package domain

type Tower struct {
	ID             string  `json:"id"`
	Region         string  `json:"region"`
	TowerType      string  `json:"tower_type"`
	LastInspection string  `json:"last_inspection"`
	WindLoadPct    float64 `json:"wind_load_pct"`
	CorrosionScore int     `json:"corrosion_score"`
	Status         string  `json:"status"`
	UpdatedAt      string  `json:"updated_at"`
}
