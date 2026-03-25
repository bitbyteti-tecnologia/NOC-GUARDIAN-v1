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

	nodes, err := repo.ListDevices(ctx)
	if err != nil {
		return Response{}, err
	}

	incidentAgg, err := repo.ListIncidentDevices(ctx)
	if err != nil {
		return Response{}, err
	}

	metrics, err := repo.ListLatestMetrics(ctx, []string{"cpu_percent", "mem_used_pct", "disk_used_pct", "agent_heartbeat"})
	if err != nil {
		return Response{}, err
	}

	metricByDevice := map[string]map[string]MetricSnapshot{}
	for _, m := range metrics {
		if _, ok := metricByDevice[m.DeviceID]; !ok {
			metricByDevice[m.DeviceID] = map[string]MetricSnapshot{}
		}
		metricByDevice[m.DeviceID][m.Metric] = m
	}

	nodeByID := map[string]*Node{}
	for i := range nodes {
		n := &nodes[i]
		if n.LastSeen != nil && time.Since(*n.LastSeen) > 5*time.Minute && n.Status == "ok" {
			n.Status = "warning"
		}
		if agg, ok := incidentAgg[n.ID]; ok {
			n.Status = agg.Severity
			n.IncidentCount = agg.Count
		}
		if n.Status == "" {
			n.Status = "ok"
		}

		if mm, ok := metricByDevice[n.ID]; ok {
			if cpu, ok := mm["cpu_percent"]; ok {
				n.Metrics.CPUPercent = &cpu.Value
			}
			if mem, ok := mm["mem_used_pct"]; ok {
				n.Metrics.MemUsedPct = &mem.Value
			}
			if disk, ok := mm["disk_used_pct"]; ok {
				n.Metrics.DiskUsedPct = &disk.Value
			}
			if hb, ok := mm["agent_heartbeat"]; ok && n.LastSeen == nil {
				t := hb.Time
				n.LastSeen = &t
			}
		}
		nodeByID[n.ID] = n
	}

	// identifica root cause: nó com incidente e que é upstream de outro nó com incidente
	childIncident := map[string]bool{}
	for _, e := range edges {
		if incidentAgg[e.Target].Severity != "" {
			childIncident[e.Source] = true
		}
	}
	for id, n := range nodeByID {
		if incidentAgg[id].Severity != "" && childIncident[id] {
			n.Root = true
		}
	}

	log.Printf("%stopology tenant=%s nodes=%d edges=%d took=%s", s.LogPrefix, tenantID, len(nodes), len(edges), time.Since(start))
	return Response{Nodes: nodes, Edges: edges}, nil
}
