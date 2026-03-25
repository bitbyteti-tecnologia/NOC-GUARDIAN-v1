package topology

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

func (r *Repository) ListRelationships(ctx context.Context) ([]Edge, error) {
	exists, err := r.tableExists(ctx, "device_relationships")
	if err != nil || !exists {
		return []Edge{}, err
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT parent_device_id::text, child_device_id::text, relation_type
FROM device_relationships`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Edge, 0)
	for rows.Next() {
		var e Edge
		if err := rows.Scan(&e.Source, &e.Target, &e.RelationType); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *Repository) ListDevices(ctx context.Context) ([]Node, error) {
	exists, err := r.tableExists(ctx, "devices")
	if err != nil || !exists {
		return []Node{}, err
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT id::text, hostname, last_seen
FROM devices
ORDER BY hostname`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Node, 0)
	for rows.Next() {
		var id, name sql.NullString
		var lastSeen sql.NullTime
		if err := rows.Scan(&id, &name, &lastSeen); err != nil {
			return nil, err
		}
		label := id.String
		if name.Valid && name.String != "" {
			label = name.String
		}
		n := Node{ID: id.String, Label: label, Status: "unknown"}
		if lastSeen.Valid {
			t := lastSeen.Time
			n.LastSeen = &t
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

type MetricSnapshot struct {
	DeviceID string
	Metric   string
	Value    float64
	Time     time.Time
}

func (r *Repository) ListLatestMetrics(ctx context.Context, metrics []string) ([]MetricSnapshot, error) {
	exists, err := r.tableExists(ctx, "metrics")
	if err != nil || !exists {
		return []MetricSnapshot{}, err
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT DISTINCT ON (device_id, metric)
  device_id::text, metric, value, time
FROM metrics
WHERE metric = ANY($1)
ORDER BY device_id, metric, time DESC`, metrics)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]MetricSnapshot, 0)
	for rows.Next() {
		var m MetricSnapshot
		if err := rows.Scan(&m.DeviceID, &m.Metric, &m.Value, &m.Time); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

type IncidentAgg struct {
	Severity string
	Count    int
}

func (r *Repository) ListIncidentDevices(ctx context.Context) (map[string]IncidentAgg, error) {
	exists, err := r.tableExists(ctx, "incident_events")
	if err != nil || !exists {
		return map[string]IncidentAgg{}, err
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT e.device_id::text, max(e.severity), count(*)
FROM incident_events ie
JOIN events e ON e.event_id = ie.event_id
WHERE e.status = 'active'
GROUP BY e.device_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]IncidentAgg)
	for rows.Next() {
		var deviceID, sev string
		var cnt int
		if err := rows.Scan(&deviceID, &sev, &cnt); err != nil {
			return nil, err
		}
		out[deviceID] = IncidentAgg{Severity: sev, Count: cnt}
	}
	return out, rows.Err()
}

var errMissing = errors.New("missing topology data")
