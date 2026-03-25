package intelligence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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

func (r *Repository) columnExists(ctx context.Context, table, col string) (bool, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT 1 FROM information_schema.columns
WHERE table_schema='public' AND table_name=$1 AND column_name=$2
LIMIT 1`, table, col)
	var one int
	if err := row.Scan(&one); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

type MetricsSchema string

const (
	MetricsUnknown MetricsSchema = "unknown"
	MetricsV1      MetricsSchema = "v1" // time, metric, value, device_id
	MetricsV2      MetricsSchema = "v2" // timestamp, metric_name, metric_value, device_id, tenant_id
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

func (r *Repository) ListActiveIncidents(ctx context.Context, tenantID string, limit int) ([]Incident, error) {
	exists, err := r.tableExists(ctx, "incidents")
	if err != nil || !exists {
		return []Incident{}, err
	}
	hasTenant, err := r.columnExists(ctx, "incidents", "tenant_id")
	if err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = 50
	}

	base := `
SELECT id, incident_id, tenant_id, root_device_id, root_event_type, severity, title, description, status, impact_count, created_at, updated_at
FROM incidents
WHERE status IN ('open','investigating')`
	args := []any{}
	if hasTenant {
		base += " AND tenant_id = $1"
		args = append(args, tenantID)
	}
	base += " ORDER BY updated_at DESC LIMIT " + fmt.Sprintf("%d", limit)

	rows, err := r.db.QueryContext(ctx, base, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Incident, 0)
	for rows.Next() {
		var inc Incident
		if err := rows.Scan(
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
			return nil, err
		}
		out = append(out, inc)
	}
	return out, rows.Err()
}

func (r *Repository) CountIncidentsInWindow(ctx context.Context, tenantID string, from, to time.Time) (int, error) {
	exists, err := r.tableExists(ctx, "incidents")
	if err != nil || !exists {
		return 0, err
	}
	hasTenant, err := r.columnExists(ctx, "incidents", "tenant_id")
	if err != nil {
		return 0, err
	}

	q := `SELECT count(*) FROM incidents WHERE created_at >= $1 AND created_at <= $2`
	args := []any{from, to}
	if hasTenant {
		q += " AND tenant_id = $3"
		args = append(args, tenantID)
	}
	row := r.db.QueryRowContext(ctx, q, args...)
	var c int
	if err := row.Scan(&c); err != nil {
		return 0, err
	}
	return c, nil
}

func (r *Repository) AvgMetric(ctx context.Context, schema MetricsSchema, tenantID string, metricNames []string, from, to time.Time) (float64, bool, error) {
	if len(metricNames) == 0 {
		return 0, false, nil
	}
	switch schema {
	case MetricsV2:
		q := `SELECT avg(metric_value) FROM metrics WHERE metric_name = ANY($1) AND timestamp >= $2 AND timestamp <= $3`
		args := []any{metricNames, from, to}
		if tenantID != "" {
			hasTenant, err := r.columnExists(ctx, "metrics", "tenant_id")
			if err != nil {
				return 0, false, err
			}
			if hasTenant {
				q += " AND tenant_id = $4"
				args = append(args, tenantID)
			}
		}
		row := r.db.QueryRowContext(ctx, q, args...)
		var v sql.NullFloat64
		if err := row.Scan(&v); err != nil {
			return 0, false, err
		}
		return v.Float64, v.Valid, nil
	case MetricsV1:
		q := `SELECT avg(value) FROM metrics WHERE metric = ANY($1) AND time >= $2 AND time <= $3`
		row := r.db.QueryRowContext(ctx, q, metricNames, from, to)
		var v sql.NullFloat64
		if err := row.Scan(&v); err != nil {
			return 0, false, err
		}
		return v.Float64, v.Valid, nil
	default:
		return 0, false, nil
	}
}

func (r *Repository) CountMetricAbove(ctx context.Context, schema MetricsSchema, tenantID string, metricName string, threshold float64, from, to time.Time) (int, error) {
	if metricName == "" {
		return 0, nil
	}
	switch schema {
	case MetricsV2:
		q := `SELECT count(*) FROM metrics WHERE metric_name = $1 AND metric_value > $2 AND timestamp >= $3 AND timestamp <= $4`
		args := []any{metricName, threshold, from, to}
		hasTenant, err := r.columnExists(ctx, "metrics", "tenant_id")
		if err != nil {
			return 0, err
		}
		if hasTenant {
			q += " AND tenant_id = $5"
			args = append(args, tenantID)
		}
		row := r.db.QueryRowContext(ctx, q, args...)
		var c int
		if err := row.Scan(&c); err != nil {
			return 0, err
		}
		return c, nil
	case MetricsV1:
		q := `SELECT count(*) FROM metrics WHERE metric = $1 AND value > $2 AND time >= $3 AND time <= $4`
		row := r.db.QueryRowContext(ctx, q, metricName, threshold, from, to)
		var c int
		if err := row.Scan(&c); err != nil {
			return 0, err
		}
		return c, nil
	default:
		return 0, nil
	}
}
