package topology

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

func (r *Repository) ListDevicesFromMetrics(ctx context.Context) ([]Node, error) {
	exists, err := r.tableExists(ctx, "metrics")
	if err != nil || !exists {
		return []Node{}, err
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT device_id::text AS device_id, max(labels->>'hostname') AS hostname
FROM metrics
GROUP BY device_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Node, 0)
	for rows.Next() {
		var id, name sql.NullString
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		label := id.String
		if name.Valid && name.String != "" {
			label = name.String
		}
		out = append(out, Node{ID: id.String, Label: label, Status: "unknown"})
	}
	return out, rows.Err()
}

func (r *Repository) ListIncidentDevices(ctx context.Context) (map[string]string, error) {
	exists, err := r.tableExists(ctx, "incident_events")
	if err != nil || !exists {
		return map[string]string{}, err
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT e.device_id::text, max(e.severity)
FROM incident_events ie
JOIN events e ON e.event_id = ie.event_id
WHERE e.status = 'active'
GROUP BY e.device_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]string)
	for rows.Next() {
		var deviceID, sev string
		if err := rows.Scan(&deviceID, &sev); err != nil {
			return nil, err
		}
		out[deviceID] = sev
	}
	return out, rows.Err()
}

var errMissing = errors.New("missing topology data")
