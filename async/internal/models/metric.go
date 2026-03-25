package models

import "time"

type Metric struct {
	EventID     string            `json:"event_id"`
	TenantID    string            `json:"tenant_id"`
	DeviceID    string            `json:"device_id"`
	MetricName  string            `json:"metric_name"`
	MetricValue float64           `json:"metric_value"`
	Labels      map[string]string `json:"labels"`
	Timestamp   time.Time         `json:"timestamp"`
}
