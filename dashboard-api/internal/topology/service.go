package topology

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"time"
)

type TenantOpener func(ctx context.Context, tenantID string) (*sql.DB, string, error)

type Service struct {
	OpenTenant    TenantOpener
	LogPrefix     string
	EnforceTenant bool
}

func (s *Service) Build(ctx context.Context, tenantID string) (Response, error) {
	if tenantID == "" {
		return Response{}, errors.New("tenant_id é obrigatório")
	}
	start := time.Now()

	db, _, err := s.OpenTenant(ctx, tenantID)
	if err != nil {
		return Response{}, err
	}
	defer db.Close()

	repo := NewRepository(db)
	edges, err := repo.ListRelationships(ctx)
	if err != nil {
		return Response{}, err
	}

	nodes, err := repo.ListDevicesFromMetrics(ctx)
	if err != nil {
		return Response{}, err
	}

	incidentSev, err := repo.ListIncidentDevices(ctx)
	if err != nil {
		return Response{}, err
	}

	nodeByID := map[string]*Node{}
	for i := range nodes {
		n := &nodes[i]
		if sev, ok := incidentSev[n.ID]; ok {
			n.Status = sev
		} else {
			n.Status = "ok"
		}
		nodeByID[n.ID] = n
	}

	// identifica root cause: nó com incidente e que é upstream de outro nó com incidente
	childIncident := map[string]bool{}
	for _, e := range edges {
		if incidentSev[e.Target] != "" {
			childIncident[e.Source] = true
		}
	}
	for id, n := range nodeByID {
		if incidentSev[id] != "" && childIncident[id] {
			n.Root = true
		}
	}

	log.Printf("%stopology tenant=%s nodes=%d edges=%d took=%s", s.LogPrefix, tenantID, len(nodes), len(edges), time.Since(start))
	return Response{Nodes: nodes, Edges: edges}, nil
}
