package neural

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"time"

	"dashboard-api/internal/intelligence"
	"dashboard-api/internal/topology"
)

type TenantOpener func(ctx context.Context, tenantID string) (*sql.DB, string, error)

type Service struct {
	OpenTenant    TenantOpener
	LogPrefix     string
	EnforceTenant bool

	Intelligence *intelligence.Service
	Topology     *topology.Service
}

func (s *Service) Build(ctx context.Context, tenantID string) (Response, error) {
	if tenantID == "" {
		return Response{}, errors.New("tenant_id é obrigatório")
	}
	start := time.Now()

	var intel intelligence.IntelligenceResponse
	var incidents []intelligence.Incident
	var topo topology.Response

	if s.Intelligence == nil {
		return Response{}, errors.New("intelligence service not configured")
	}
	intel, err := s.Intelligence.Build(ctx, tenantID)
	if err != nil {
		return Response{}, err
	}

	if s.OpenTenant != nil {
		db, _, err := s.OpenTenant(ctx, tenantID)
		if err == nil {
			repo := intelligence.NewRepository(db)
			incidents, _ = repo.ListActiveIncidents(ctx, tenantID, 50)
			_ = db.Close()
		}
	}

	if s.Topology != nil {
		if t, err := s.Topology.Build(ctx, tenantID); err == nil {
			topo = t
		}
	}

	primary := buildPrimaryIssue(intel, incidents)

	log.Printf("%sneural tenant=%s incidents=%d nodes=%d took=%s",
		s.LogPrefix, tenantID, len(incidents), len(topo.Nodes), time.Since(start))

	return Response{
		Intelligence: intel,
		Incidents:    incidents,
		Topology:     topo,
		PrimaryIssue: primary,
		GeneratedAt:  time.Now().UTC(),
	}, nil
}

func buildPrimaryIssue(intel intelligence.IntelligenceResponse, incidents []intelligence.Incident) PrimaryIssue {
	if len(intel.TopIncidents) > 0 {
		inc := intel.TopIncidents[0]
		title := inc.Title
		if title == "" {
			title = inc.RootEvent
		}
		if title == "" {
			title = "Incidente crítico"
		}
		summary := "Incidente prioritário requer atenção imediata."
		if inc.ImpactCount > 0 {
			summary = "Afetando múltiplos dispositivos na infraestrutura."
		}
		return PrimaryIssue{
			Title:       title,
			Severity:    inc.Severity,
			Summary:     summary,
			RootDevice:  inc.RootDevice,
			ImpactCount: inc.ImpactCount,
			Source:      "incident",
		}
	}

	if len(intel.Insights) > 0 {
		ins := intel.Insights[0]
		return PrimaryIssue{
			Title:    "Insight prioritário",
			Severity: ins.Severity,
			Summary:  ins.Message,
			Source:   "insight",
		}
	}

	status := intel.Status
	if status == "" {
		status = "healthy"
	}
	return PrimaryIssue{
		Title:    "Ambiente estável",
		Severity: status,
		Summary:  "Nenhum problema crítico identificado.",
		Source:   "status",
	}
}
