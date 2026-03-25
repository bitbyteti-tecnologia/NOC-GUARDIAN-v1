package incidents

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) tableExists(ctx context.Context, name string) (bool, error) {
	row := r.db.QueryRowContext(ctx, `SELECT to_regclass($1)`, name)
	var reg sql.NullString
	if err := row.Scan(&reg); err != nil {
		return false, err
	}
	return reg.Valid, nil
}

func (r *Repository) GetIncident(ctx context.Context, incidentID string) (Incident, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, incident_id, tenant_id, root_device_id, root_event_type, severity, title, description, status, impact_count, created_at, updated_at
FROM incidents WHERE incident_id = $1`, incidentID)
	var inc Incident
	if err := row.Scan(
		&inc.ID,
		&inc.IncidentID,
		&inc.TenantID,
		&inc.RootDevice,
		&inc.RootEvent,
		&inc.Severity,
		&inc.Title,
		&inc.Description,
		&inc.Status,
		&inc.ImpactCount,
		&inc.CreatedAt,
		&inc.UpdatedAt,
	); err != nil {
		return Incident{}, err
	}
	return inc, nil
}

func (r *Repository) ListEvents(ctx context.Context, incidentID string) ([]Event, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT e.event_id, e.device_id, e.event_type, e.severity, e.status, e.message, e.first_seen, e.last_seen, e.created_at, e.updated_at
FROM incident_events ie
JOIN events e ON e.event_id = ie.event_id
WHERE ie.incident_id = $1
ORDER BY e.first_seen ASC`, incidentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Event, 0)
	for rows.Next() {
		var ev Event
		if err := rows.Scan(&ev.EventID, &ev.DeviceID, &ev.EventType, &ev.Severity, &ev.Status, &ev.Message, &ev.FirstSeen, &ev.LastSeen, &ev.CreatedAt, &ev.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, ev)
	}
	return out, rows.Err()
}

type MetricsSchema string

const (
	MetricsUnknown MetricsSchema = "unknown"
	MetricsV1      MetricsSchema = "v1"
	MetricsV2      MetricsSchema = "v2"
)

func (r *Repository) DetectMetricsSchema(ctx context.Context) (MetricsSchema, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT column_name
FROM information_schema.columns
WHERE table_schema='public' AND table_name='metrics'`)
	if err != nil {
		return MetricsUnknown, err
	}
	defer rows.Close()

	cols := map[string]bool{}
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return MetricsUnknown, err
		}
		cols[c] = true
	}
	if err := rows.Err(); err != nil {
		return MetricsUnknown, err
	}

	if cols["metric_name"] && cols["metric_value"] && cols["timestamp"] {
		return MetricsV2, nil
	}
	if cols["metric"] && cols["value"] && cols["time"] {
		return MetricsV1, nil
	}
	return MetricsUnknown, nil
}

func (r *Repository) RecentMetrics(ctx context.Context, schema MetricsSchema, deviceID string, from, to time.Time, metrics []string) ([]MetricSeries, error) {
	if deviceID == "" || len(metrics) == 0 {
		return []MetricSeries{}, nil
	}

	var rows *sql.Rows
	var err error
	if schema == MetricsV2 {
		rows, err = r.db.QueryContext(ctx, `
SELECT metric_name, timestamp, metric_value
FROM metrics
WHERE device_id = $1 AND metric_name = ANY($2) AND timestamp >= $3 AND timestamp <= $4
ORDER BY metric_name, timestamp`, deviceID, metrics, from, to)
	} else if schema == MetricsV1 {
		rows, err = r.db.QueryContext(ctx, `
SELECT metric, time, value
FROM metrics
WHERE device_id = $1 AND metric = ANY($2) AND time >= $3 AND time <= $4
ORDER BY metric, time`, deviceID, metrics, from, to)
	} else {
		return []MetricSeries{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	seriesMap := map[string][]MetricPoint{}
	for rows.Next() {
		var metric string
		var ts time.Time
		var v float64
		if err := rows.Scan(&metric, &ts, &v); err != nil {
			return nil, err
		}
		seriesMap[metric] = append(seriesMap[metric], MetricPoint{T: ts, V: v})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]MetricSeries, 0, len(seriesMap))
	for metric, pts := range seriesMap {
		out = append(out, MetricSeries{Metric: metric, Points: pts})
	}
	return out, nil
}

func (r *Repository) EnsureIncidentsTables(ctx context.Context) error {
	exists, err := r.tableExists(ctx, "incidents")
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("incidents table not found")
	}
	return nil
}
