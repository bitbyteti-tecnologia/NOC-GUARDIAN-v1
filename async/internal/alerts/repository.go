package alerts

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) CreateIfNotExists(ctx context.Context, alert Alert) error {
	_, err := r.pool.Exec(
		ctx,
		`INSERT INTO alerts (tenant_id, event_id, alert_type, severity, message, acknowledged, created_at)
         VALUES ($1, $2, $3, $4, $5, false, NOW())
         ON CONFLICT (event_id, alert_type) DO NOTHING`,
		alert.TenantID,
		alert.EventID,
		alert.AlertType,
		alert.Severity,
		alert.Message,
	)
	return err
}

func (r *Repository) List(ctx context.Context, tenantID string, acknowledged *bool) ([]Alert, error) {
	query := `SELECT id, tenant_id, event_id, alert_type, severity, message, acknowledged, created_at
        FROM alerts
        WHERE tenant_id = $1`
	args := []any{tenantID}

	if acknowledged != nil {
		query += " AND acknowledged = $2"
		args = append(args, *acknowledged)
	}
	query += " ORDER BY created_at DESC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	alerts := make([]Alert, 0)
	for rows.Next() {
		var a Alert
		if err := rows.Scan(&a.ID, &a.TenantID, &a.EventID, &a.AlertType, &a.Severity, &a.Message, &a.Acknowledged, &a.CreatedAt); err != nil {
			return nil, err
		}
		alerts = append(alerts, a)
	}
	return alerts, rows.Err()
}

func (r *Repository) Ack(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `UPDATE alerts SET acknowledged = true WHERE id = $1`, id)
	return err
}
