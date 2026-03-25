package dashboard

import "time"

type AggregateRequest struct {
	TenantID     string
	MetricName   string
	Mode         string
	From         time.Time
	To           time.Time
	Interval     time.Duration
	OnlineWindow time.Duration
	Fill         string
	MaxPoints    int
}

type AggregatePoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

type AggregateResponse struct {
	TenantID   string           `json:"tenant_id"`
	MetricName string           `json:"metric_name"`
	Mode       string           `json:"mode"`
	From       time.Time        `json:"from"`
	To         time.Time        `json:"to"`
	Interval   string           `json:"interval"`
	Fill       string           `json:"fill,omitempty"`
	Points     []AggregatePoint `json:"points"`
}
