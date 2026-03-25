package events

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) GetActive(ctx context.Context, tenantID, deviceID, eventType string) (*Event, error) {
	row := r.pool.QueryRow(
		ctx,
		`SELECT id, event_id, tenant_id, device_id, event_type, severity, message, metadata, first_seen, last_seen, status, created_at, updated_at
         FROM events
         WHERE tenant_id = $1 AND device_id = $2 AND event_type = $3 AND status = 'active'
         LIMIT 1`,
		tenantID,
		deviceID,
		eventType,
	)

	var e Event
	var metadataJSON []byte
	if err := row.Scan(&e.ID, &e.EventID, &e.TenantID, &e.DeviceID, &e.EventType, &e.Severity, &e.Message, &metadataJSON, &e.FirstSeen, &e.LastSeen, &e.Status, &e.CreatedAt, &e.UpdatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &e.Metadata); err != nil {
			return nil, err
		}
	}
	return &e, nil
}

func (r *Repository) UpsertActive(ctx context.Context, e *Event) (bool, error) {
	existing, err := r.GetActive(ctx, e.TenantID, e.DeviceID, e.EventType)
	if err != nil {
		return false, err
	}

	now := time.Now().UTC()
	metadataJSON, err := json.Marshal(e.Metadata)
	if err != nil {
		return false, err
	}

	if existing == nil {
		_, err := r.pool.Exec(
			ctx,
			`INSERT INTO events (event_id, tenant_id, device_id, event_type, severity, message, metadata, first_seen, last_seen, status, created_at, updated_at)
             VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'active', $10, $10)`,
			e.EventID,
			e.TenantID,
			e.DeviceID,
			e.EventType,
			e.Severity,
			e.Message,
			metadataJSON,
			now,
			now,
			now,
		)
		if err != nil {
			return false, err
		}
		return true, nil
	}

	_, err = r.pool.Exec(
		ctx,
		`UPDATE events
         SET severity = $1, message = $2, metadata = $3, last_seen = $4, updated_at = $4
         WHERE id = $5`,
		e.Severity,
		e.Message,
		metadataJSON,
		now,
		existing.ID,
	)
	if err != nil {
		return false, err
	}
	return false, nil
}

func (r *Repository) ResolveActive(ctx context.Context, tenantID, deviceID, eventType string) (bool, error) {
	now := time.Now().UTC()
	cmd, err := r.pool.Exec(
		ctx,
		`UPDATE events
         SET status = 'resolved', last_seen = $1, updated_at = $1
         WHERE tenant_id = $2 AND device_id = $3 AND event_type = $4 AND status = 'active'`,
		now,
		tenantID,
		deviceID,
		eventType,
	)
	if err != nil {
		return false, err
	}
	return cmd.RowsAffected() > 0, nil
}

type ListFilter struct {
	TenantID string
	Status   string
	Severity string
}

func (r *Repository) List(ctx context.Context, filter ListFilter) ([]Event, error) {
	query := `SELECT id, event_id, tenant_id, device_id, event_type, severity, message, metadata, first_seen, last_seen, status, created_at, updated_at
        FROM events
        WHERE tenant_id = $1`
	args := []any{filter.TenantID}

	idx := 2
	if filter.Status != "" {
		query += " AND status = $" + strconv.Itoa(idx)
		args = append(args, filter.Status)
		idx++
	}
	if filter.Severity != "" {
		query += " AND severity = $" + strconv.Itoa(idx)
		args = append(args, filter.Severity)
		idx++
	}
	query += " ORDER BY created_at DESC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]Event, 0)
	for rows.Next() {
		var e Event
		var metadataJSON []byte
		if err := rows.Scan(&e.ID, &e.EventID, &e.TenantID, &e.DeviceID, &e.EventType, &e.Severity, &e.Message, &metadataJSON, &e.FirstSeen, &e.LastSeen, &e.Status, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &e.Metadata); err != nil {
				return nil, err
			}
		}
		events = append(events, e)
	}
	return events, rows.Err()
}
