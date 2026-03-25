package incidents

import "time"

type Incident struct {
	ID          int64     `json:"id"`
	IncidentID  string    `json:"incident_id"`
	TenantID    string    `json:"tenant_id"`
	RootDevice  string    `json:"root_device_id"`
	RootEvent   string    `json:"root_event_type"`
	Severity    string    `json:"severity"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	ImpactCount int       `json:"impact_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Event struct {
	EventID   string    `json:"event_id"`
	DeviceID  string    `json:"device_id"`
	EventType string    `json:"event_type"`
	Severity  string    `json:"severity"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TimelineItem struct {
	Ts        time.Time `json:"ts"`
	Type      string    `json:"type"`
	EventID   string    `json:"event_id"`
	EventType string    `json:"event_type"`
	Severity  string    `json:"severity"`
	DeviceID  string    `json:"device_id"`
}

type DeviceItem struct {
	DeviceID string `json:"device_id"`
	Status   string `json:"status"`
}

type MetricPoint struct {
	T time.Time `json:"t"`
	V float64   `json:"v"`
}

type MetricSeries struct {
	Metric string        `json:"metric"`
	Points []MetricPoint `json:"points"`
}

type DetailsResponse struct {
	Incident Incident       `json:"incident"`
	Events   []Event        `json:"events"`
	Timeline []TimelineItem `json:"timeline"`
	Devices  []DeviceItem   `json:"devices"`
	Metrics  []MetricSeries `json:"metrics"`
}
