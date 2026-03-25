package intelligence

import "time"

type Insight struct {
	Type     string `json:"type"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
}

type Recommendation struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type Incident struct {
	ID          int64
	IncidentID  string
	TenantID    string
	RootDevice  string
	RootEvent   string
	Severity    string
	Title       string
	Description string
	Status      string
	ImpactCount int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type IntelligenceResponse struct {
	HealthScore     int              `json:"health_score"`
	Status          string           `json:"status"`
	TopIncidents    []Incident       `json:"top_incidents"`
	Insights        []Insight        `json:"insights"`
	Recommendations []Recommendation `json:"recommendations"`
}
