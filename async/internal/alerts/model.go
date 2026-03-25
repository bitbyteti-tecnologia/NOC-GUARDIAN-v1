package alerts

import "time"

type Alert struct {
	ID           int64     `json:"id"`
	TenantID     string    `json:"tenant_id"`
	EventID      string    `json:"event_id"`
	AlertType    string    `json:"alert_type"`
	Severity     string    `json:"severity"`
	Message      string    `json:"message"`
	Acknowledged bool      `json:"acknowledged"`
	CreatedAt    time.Time `json:"created_at"`
}
