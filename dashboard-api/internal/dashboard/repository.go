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

func (r *Repository) QueryAggregate(ctx context.Context, req AggregateRequest) ([]AggregatePoint, error) {
	if req.Interval <= 0 {
		return nil, errors.New("interval inválido")
	}
	bucketSeconds := int64(req.Interval.Seconds())

	var rows *sql.Rows
	var err error

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
