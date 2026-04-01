package metrics

import (
	"regexp"
	"strings"
)

var nonAlphaNum = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func sanitizeMetricKey(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "unknown"
	}
	s = strings.ToLower(s)
	s = nonAlphaNum.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	if s == "" {
		return "unknown"
	}
	return s
}

func boolToFloat(v bool) float64 {
	if v {
		return 1
	}
	return 0
}

func clampPct(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}
