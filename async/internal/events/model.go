package events

import "time"

type Event struct {
	ID        int64          `json:"id"`
	EventID   string         `json:"event_id"`
	TenantID  string         `json:"tenant_id"`
	DeviceID  string         `json:"device_id"`
	EventType string         `json:"event_type"`
	Severity  string         `json:"severity"`
	Message   string         `json:"message"`
	Metadata  map[string]any `json:"metadata"`
	FirstSeen time.Time      `json:"first_seen"`
	LastSeen  time.Time      `json:"last_seen"`
	Status    string         `json:"status"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}
