package incidents

import (
	"context"
	"errors"
	"fmt"
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

func (r *Repository) GetOpenByDevice(ctx context.Context, tenantID, deviceID string) (*Incident, error) {
	row := r.pool.QueryRow(
		ctx,
		`SELECT id, incident_id, tenant_id, root_device_id, root_event_type, severity, title, description, status, impact_count, created_at, updated_at
         FROM incidents
         WHERE tenant_id = $1 AND root_device_id = $2 AND status IN ('open','investigating')
         LIMIT 1`,
		tenantID,
		deviceID,
	)
	return scanIncident(row)
}

func (r *Repository) GetOpenByEventTypeWindow(ctx context.Context, tenantID, eventType string, window time.Duration) (*Incident, error) {
	row := r.pool.QueryRow(
		ctx,
		`SELECT id, incident_id, tenant_id, root_device_id, root_event_type, severity, title, description, status, impact_count, created_at, updated_at
         FROM incidents
         WHERE tenant_id = $1 AND root_event_type = $2 AND status IN ('open','investigating') AND updated_at >= NOW() - ($3 * INTERVAL '1 second')
         ORDER BY updated_at DESC
         LIMIT 1`,
		tenantID,
		eventType,
		int64(window.Seconds()),
	)
	return scanIncident(row)
}

func (r *Repository) Create(ctx context.Context, inc Incident) (*Incident, error) {
	row := r.pool.QueryRow(
		ctx,
		`INSERT INTO incidents (incident_id, tenant_id, root_device_id, root_event_type, severity, title, description, status, impact_count, created_at, updated_at)
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
         RETURNING id, incident_id, tenant_id, root_device_id, root_event_type, severity, title, description, status, impact_count, created_at, updated_at`,
		inc.IncidentID,
		inc.TenantID,
		inc.RootDeviceID,
		inc.RootEvent,
		inc.Severity,
		inc.Title,
		inc.Description,
		inc.Status,
		inc.ImpactCount,
	)
	return scanIncident(row)
}

func (r *Repository) AddEvent(ctx context.Context, incidentID, eventID, tenantID string) (bool, error) {
	cmd, err := r.pool.Exec(
		ctx,
		`INSERT INTO incident_events (incident_id, event_id, tenant_id)
         VALUES ($1, $2, $3)
         ON CONFLICT (tenant_id, event_id) DO NOTHING`,
		incidentID,
		eventID,
		tenantID,
	)
	if err != nil {
		return false, err
	}
	return cmd.RowsAffected() > 0, nil
}

func (r *Repository) UpdateIncident(ctx context.Context, inc Incident) error {
	_, err := r.pool.Exec(
		ctx,
		`UPDATE incidents
         SET root_device_id = $1, root_event_type = $2, severity = $3, title = $4, description = $5, status = $6, impact_count = $7, updated_at = NOW()
         WHERE incident_id = $8`,
		inc.RootDeviceID,
		inc.RootEvent,
		inc.Severity,
		inc.Title,
		inc.Description,
		inc.Status,
		inc.ImpactCount,
		inc.IncidentID,
	)
	return err
}

func (r *Repository) Recompute(ctx context.Context, incidentID string) (Incident, error) {
	var inc Incident
	row := r.pool.QueryRow(
		ctx,
		`SELECT id, incident_id, tenant_id, root_device_id, root_event_type, severity, title, description, status, impact_count, created_at, updated_at
         FROM incidents
         WHERE incident_id = $1`,
		incidentID,
	)
	scanned, err := scanIncident(row)
	if err != nil {
		return inc, err
	}
	inc = *scanned

	var rootDevice string
	var rootEvent string
	var severity string
	var firstSeen time.Time
	var impactCount int

	row = r.pool.QueryRow(
		ctx,
		`WITH ordered AS (
             SELECT e.device_id, e.event_type, e.severity, e.first_seen
             FROM incident_events ie
             JOIN events e ON e.event_id = ie.event_id
             WHERE ie.incident_id = $1
             ORDER BY
               CASE e.severity WHEN 'critical' THEN 3 WHEN 'warning' THEN 2 WHEN 'info' THEN 1 ELSE 0 END DESC,
               e.first_seen ASC
             LIMIT 1
         )
         SELECT device_id, event_type, severity, first_seen FROM ordered`,
		incidentID,
	)
	if err := row.Scan(&rootDevice, &rootEvent, &severity, &firstSeen); err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return inc, err
		}
	}

	row = r.pool.QueryRow(
		ctx,
		`SELECT COUNT(DISTINCT e.device_id)
         FROM incident_events ie
         JOIN events e ON e.event_id = ie.event_id
         WHERE ie.incident_id = $1`,
		incidentID,
	)
	if err := row.Scan(&impactCount); err != nil {
		return inc, err
	}

	if rootDevice != "" {
		inc.RootDeviceID = rootDevice
		inc.RootEvent = rootEvent
		inc.Severity = severity
		inc.ImpactCount = impactCount
	}

	return inc, nil
}

func (r *Repository) List(ctx context.Context, tenantID, status string) ([]Incident, error) {
	query := `SELECT id, incident_id, tenant_id, root_device_id, root_event_type, severity, title, description, status, impact_count, created_at, updated_at
        FROM incidents WHERE tenant_id = $1`
	args := []any{tenantID}
	if status != "" {
		query += " AND status = $2"
		args = append(args, status)
	}
	query += " ORDER BY updated_at DESC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]Incident, 0)
	for rows.Next() {
		inc, err := scanIncident(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *inc)
	}
	return list, rows.Err()
}

func (r *Repository) GetDetail(ctx context.Context, incidentID string) (Incident, []string, error) {
	inc, err := r.getByIncidentID(ctx, incidentID)
	if err != nil {
		return Incident{}, nil, err
	}

	rows, err := r.pool.Query(
		ctx,
		`SELECT event_id FROM incident_events WHERE incident_id = $1`,
		incidentID,
	)
	if err != nil {
		return Incident{}, nil, err
	}
	defer rows.Close()

	events := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return Incident{}, nil, err
		}
		events = append(events, id)
	}
	return *inc, events, rows.Err()
}

func (r *Repository) ResolveIfNoActiveEvents(ctx context.Context, incidentID string) (bool, error) {
	row := r.pool.QueryRow(
		ctx,
		`SELECT COUNT(*)
         FROM incident_events ie
         JOIN events e ON e.event_id = ie.event_id
         WHERE ie.incident_id = $1 AND e.status = 'active'`,
		incidentID,
	)
	var activeCount int
	if err := row.Scan(&activeCount); err != nil {
		return false, err
	}
	if activeCount > 0 {
		return false, nil
	}

	cmd, err := r.pool.Exec(
		ctx,
		`UPDATE incidents SET status = 'resolved', updated_at = NOW() WHERE incident_id = $1`,
		incidentID,
	)
	if err != nil {
		return false, err
	}
	return cmd.RowsAffected() > 0, nil
}

func (r *Repository) ListOpen(ctx context.Context) ([]Incident, error) {
	rows, err := r.pool.Query(
		ctx,
		`SELECT id, incident_id, tenant_id, root_device_id, root_event_type, severity, title, description, status, impact_count, created_at, updated_at
         FROM incidents WHERE status IN ('open','investigating')`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]Incident, 0)
	for rows.Next() {
		inc, err := scanIncident(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *inc)
	}
	return list, rows.Err()
}

func (r *Repository) getByIncidentID(ctx context.Context, incidentID string) (*Incident, error) {
	row := r.pool.QueryRow(
		ctx,
		`SELECT id, incident_id, tenant_id, root_device_id, root_event_type, severity, title, description, status, impact_count, created_at, updated_at
         FROM incidents WHERE incident_id = $1`,
		incidentID,
	)
	return scanIncident(row)
}

type scanner interface {
	Scan(dest ...any) error
}

func scanIncident(row scanner) (*Incident, error) {
	var inc Incident
	if err := row.Scan(
		&inc.ID,
		&inc.IncidentID,
		&inc.TenantID,
		&inc.RootDeviceID,
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
	return &inc, nil
}

func BuildTitle(eventType, deviceID string) string {
	return fmt.Sprintf("Incidente: %s em %s", eventType, deviceID)
}

func BuildDescription(eventType string, impact int) string {
	if impact <= 1 {
		return fmt.Sprintf("Evento correlacionado: %s", eventType)
	}
	return fmt.Sprintf("Eventos correlacionados (%d dispositivos): %s", impact, eventType)
}
