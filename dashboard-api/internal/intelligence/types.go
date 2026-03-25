package intelligence

import "time"

type Insight struct {
	Type          string   `json:"type"`
	Message       string   `json:"message"`
	Severity      string   `json:"severity"`
	DeviceID      string   `json:"device_id,omitempty"`
	Metric        string   `json:"metric,omitempty"`
	ChangePercent *float64 `json:"change_percent,omitempty"`
	Context       string   `json:"context,omitempty"`
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
	Priority    int `json:"priority_score"`
}

type IntelligenceResponse struct {
	HealthScore     int              `json:"health_score"`
	Status          string           `json:"status"`
	Trend           string           `json:"trend"`
	Summary         string           `json:"summary"`
	TopIncidents    []Incident       `json:"top_incidents"`
	Insights        []Insight        `json:"insights"`
	Recommendations []Recommendation `json:"recommendations"`
}
