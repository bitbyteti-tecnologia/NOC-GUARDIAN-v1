package db

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bitbyteti/noc-guardian/async/internal/models"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(ctx context.Context, dsn string) (*Store, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() {
	if s.pool != nil {
		s.pool.Close()
	}
}

func (s *Store) Pool() *pgxpool.Pool {
	return s.pool
}

func (s *Store) InsertMetricsBatch(ctx context.Context, metrics []models.Metric) error {
	if len(metrics) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	now := time.Now().UTC()
	for _, m := range metrics {
		labelsJSON, err := json.Marshal(m.Labels)
		if err != nil {
			return err
		}
		batch.Queue(
			`INSERT INTO metrics (event_id, tenant_id, device_id, metric_name, metric_value, labels, timestamp, created_at)
             VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
             ON CONFLICT (event_id) DO NOTHING`,
			m.EventID,
			m.TenantID,
			m.DeviceID,
			m.MetricName,
			m.MetricValue,
			labelsJSON,
			m.Timestamp,
			now,
		)
	}
	br := s.pool.SendBatch(ctx, batch)
	defer br.Close()
	for i := 0; i < len(metrics); i++ {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) QueryMetrics(ctx context.Context, tenantID string, deviceID *string) ([]models.Metric, error) {
	var rows pgx.Rows
	var err error

	if deviceID != nil {
		rows, err = s.pool.Query(
			ctx,
			`SELECT event_id, tenant_id, device_id, metric_name, metric_value, labels, timestamp
             FROM metrics
             WHERE tenant_id = $1 AND device_id = $2
             ORDER BY timestamp ASC`,
			tenantID,
			*deviceID,
		)
	} else {
		rows, err = s.pool.Query(
			ctx,
			`SELECT event_id, tenant_id, device_id, metric_name, metric_value, labels, timestamp
             FROM metrics
             WHERE tenant_id = $1
             ORDER BY timestamp ASC`,
			tenantID,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	metrics := make([]models.Metric, 0)
	for rows.Next() {
		var m models.Metric
		var labelsJSON []byte
		if err := rows.Scan(&m.EventID, &m.TenantID, &m.DeviceID, &m.MetricName, &m.MetricValue, &labelsJSON, &m.Timestamp); err != nil {
			return nil, err
		}
		if len(labelsJSON) > 0 {
			if err := json.Unmarshal(labelsJSON, &m.Labels); err != nil {
				return nil, err
			}
		}
		metrics = append(metrics, m)
	}
	return metrics, rows.Err()
}

type DeviceLastSeen struct {
	TenantID string
	DeviceID string
	LastSeen time.Time
}

func (s *Store) ListStaleDevices(ctx context.Context, threshold time.Duration) ([]DeviceLastSeen, error) {
	seconds := int64(threshold.Seconds())
	rows, err := s.pool.Query(
		ctx,
		`SELECT tenant_id, device_id, MAX(timestamp) as last_seen
         FROM metrics
         GROUP BY tenant_id, device_id
         HAVING MAX(timestamp) < NOW() - ($1 * INTERVAL '1 second')`,
		seconds,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]DeviceLastSeen, 0)
	for rows.Next() {
		var item DeviceLastSeen
		if err := rows.Scan(&item.TenantID, &item.DeviceID, &item.LastSeen); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}
