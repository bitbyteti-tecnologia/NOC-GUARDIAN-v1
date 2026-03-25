package correlation

import "github.com/bitbyteti/noc-guardian/async/internal/events"

func SeverityRank(s string) int {
	switch s {
	case "critical":
		return 3
	case "warning":
		return 2
	case "info":
		return 1
	default:
		return 0
	}
}

func MaxSeverity(a, b string) string {
	if SeverityRank(b) > SeverityRank(a) {
		return b
	}
	return a
}

func RootFromEvent(ev events.Event) (string, string, string) {
	return ev.DeviceID, ev.EventType, ev.Severity
}
