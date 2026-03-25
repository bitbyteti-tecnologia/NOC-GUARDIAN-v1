package dashboard

import (
	"context"
	"database/sql"
	"errors"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

type SchemaType string

const (
	SchemaUnknown SchemaType = "unknown"
	SchemaV1      SchemaType = "v1" // time, metric, value, labels
	SchemaV2      SchemaType = "v2" // tenant_id, device_id, metric_name, metric_value, timestamp
)

// TODO: remover suporte ao SchemaV1 após migração completa dos tenants.

func (r *Repository) DetectSchema(ctx context.Context) (SchemaType, error) {
	q := `
SELECT column_name
FROM information_schema.columns
WHERE table_schema = 'public' AND table_name = 'metrics'
`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return SchemaUnknown, err
	}
	defer rows.Close()

	cols := map[string]bool{}
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return SchemaUnknown, err
		}
		cols[c] = true
	}
	if err := rows.Err(); err != nil {
		return SchemaUnknown, err
	}

	if cols["tenant_id"] && cols["metric_name"] && cols["metric_value"] && cols["timestamp"] {
		return SchemaV2, nil
	}
	if cols["metric"] && cols["value"] && cols["time"] {
		return SchemaV1, nil
	}
	return SchemaUnknown, nil
}

func (r *Repository) QueryAggregate(ctx context.Context, req AggregateRequest, schema SchemaType) ([]AggregatePoint, error) {
	if req.Interval <= 0 {
		return nil, errors.New("interval inválido")
	}
	bucketSeconds := int64(req.Interval.Seconds())

	var rows *sql.Rows
	var err error

	switch schema {
	case SchemaV2:
		if req.Mode == "online" {
			onlineSeconds := int64(req.OnlineWindow.Seconds())
			q := `
WITH online AS (
  SELECT device_id
  FROM metrics
  WHERE tenant_id = $2 AND metric_name = $3 AND timestamp >= $5 - ($6 * INTERVAL '1 second')
  GROUP BY device_id
)
SELECT
  to_timestamp(floor(extract(epoch from m.timestamp)/$1)*$1) AS bucket,
  sum(m.metric_value) AS v
FROM metrics m
JOIN online o ON o.device_id = m.device_id
WHERE m.tenant_id = $2
  AND m.metric_name = $3
  AND m.timestamp >= $4 AND m.timestamp <= $5
GROUP BY bucket
ORDER BY bucket;`
			rows, err = r.db.QueryContext(ctx, q, bucketSeconds, req.TenantID, req.MetricName, req.From, req.To, onlineSeconds)
		} else {
			q := `
SELECT
  to_timestamp(floor(extract(epoch from timestamp)/$1)*$1) AS bucket,
  sum(metric_value) AS v
FROM metrics
WHERE tenant_id = $2
  AND metric_name = $3
  AND timestamp >= $4 AND timestamp <= $5
GROUP BY bucket
ORDER BY bucket;`
			rows, err = r.db.QueryContext(ctx, q, bucketSeconds, req.TenantID, req.MetricName, req.From, req.To)
		}
	case SchemaV1:
		if req.Mode == "online" {
			onlineSeconds := int64(req.OnlineWindow.Seconds())
			q := `
WITH online AS (
  SELECT device_id
  FROM metrics
  WHERE metric = $2 AND time >= $4 - ($5 * INTERVAL '1 second')
  GROUP BY device_id
)
SELECT
  to_timestamp(floor(extract(epoch from m.time)/$1)*$1) AS bucket,
  sum(m.value) AS v
FROM metrics m
JOIN online o ON o.device_id = m.device_id
WHERE m.metric = $2
  AND m.time >= $3 AND m.time <= $4
GROUP BY bucket
ORDER BY bucket;`
			rows, err = r.db.QueryContext(ctx, q, bucketSeconds, req.MetricName, req.From, req.To, onlineSeconds)
		} else {
			q := `
SELECT
  to_timestamp(floor(extract(epoch from time)/$1)*$1) AS bucket,
  sum(value) AS v
FROM metrics
WHERE metric = $2
  AND time >= $3 AND time <= $4
GROUP BY bucket
ORDER BY bucket;`
			rows, err = r.db.QueryContext(ctx, q, bucketSeconds, req.MetricName, req.From, req.To)
		}
	default:
		return nil, errors.New("schema metrics desconhecido")
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	points := make([]AggregatePoint, 0)
	for rows.Next() {
		var p AggregatePoint
		if err := rows.Scan(&p.Timestamp, &p.Value); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}
