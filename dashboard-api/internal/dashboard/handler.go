package dashboard

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

type WriteJSON func(w http.ResponseWriter, v any, err error)

type Router interface {
	Get(pattern string, handlerFn http.HandlerFunc)
}

func RegisterRoutes(r Router, svc *Service, writeJSON WriteJSON) {
	r.Get("/dashboard/series", SeriesHandler(svc, writeJSON))
}

func SeriesHandler(svc *Service, writeJSON WriteJSON) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		q := r.URL.Query()
		tenantID := q.Get("tenant_id")
		metricName := q.Get("metric_name")
		mode := q.Get("mode")
		fill := q.Get("fill")
		from, _ := parseTime(q.Get("from"))
		to, _ := parseTime(q.Get("to"))
		interval, err := parseInterval(q.Get("interval"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		resp, err := svc.AggregateSeries(r.Context(), AggregateRequest{
			TenantID:   tenantID,
			MetricName: metricName,
			Mode:       mode,
			From:       from,
			To:         to,
			Interval:   interval,
			Fill:       fill,
		})
		writeJSON(w, resp, err)
	}
}

func parseInterval(raw string) (time.Duration, error) {
	if strings.TrimSpace(raw) == "" {
		return time.Minute, nil
	}
	return time.ParseDuration(raw)
}

func parseTime(raw string) (time.Time, bool) {
	if strings.TrimSpace(raw) == "" {
		return time.Time{}, false
	}
	if n, err := strconv.ParseInt(raw, 10, 64); err == nil {
		if n > 10_000_000_000 {
			return time.UnixMilli(n).UTC(), true
		}
		return time.Unix(n, 0).UTC(), true
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t.UTC(), true
	}
	return time.Time{}, false
}
