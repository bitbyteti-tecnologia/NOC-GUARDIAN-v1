package neural

import (
	"time"

	"dashboard-api/internal/intelligence"
	"dashboard-api/internal/topology"
)

type PrimaryIssue struct {
	Title       string `json:"title"`
	Severity    string `json:"severity"`
	Summary     string `json:"summary"`
	RootDevice  string `json:"root_device_id,omitempty"`
	ImpactCount int    `json:"impact_count,omitempty"`
	Source      string `json:"source"` // incident | insight | status
}

type Response struct {
	Intelligence intelligence.IntelligenceResponse `json:"intelligence"`
	Incidents    []intelligence.Incident           `json:"incidents"`
	Topology     topology.Response                 `json:"topology"`
	PrimaryIssue PrimaryIssue                      `json:"primary_issue"`
	GeneratedAt  time.Time                         `json:"generated_at"`
}
