package incidents

import "time"

type Incident struct {
	ID           int64     `json:"id"`
	IncidentID   string    `json:"incident_id"`
	TenantID     string    `json:"tenant_id"`
	RootDeviceID string    `json:"root_device_id"`
	RootEvent    string    `json:"root_event_type"`
	Severity     string    `json:"severity"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	Status       string    `json:"status"`
	ImpactCount  int       `json:"impact_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type IncidentEvent struct {
	ID         int64  `json:"id"`
	IncidentID string `json:"incident_id"`
	EventID    string `json:"event_id"`
	TenantID   string `json:"tenant_id"`
}
