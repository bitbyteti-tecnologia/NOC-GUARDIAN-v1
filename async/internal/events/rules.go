package events

import "github.com/bitbyteti/noc-guardian/async/internal/models"

type Rule struct {
	MetricName string
	Threshold  float64
	Severity   string
	EventType  string
	Message    string
}

func (r Rule) Matches(m models.Metric) bool {
	return m.MetricName == r.MetricName
}

func (r Rule) Triggered(value float64) bool {
	return value > r.Threshold
}
