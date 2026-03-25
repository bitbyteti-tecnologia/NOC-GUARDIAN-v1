package intelligence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"sort"
	"time"
)

type TenantOpener func(ctx context.Context, tenantID string) (*sql.DB, string, error)

type Service struct {
	OpenTenant   TenantOpener
	LogPrefix    string
	OnlineWindow time.Duration
}

func (s *Service) Build(ctx context.Context, tenantID string) (IntelligenceResponse, error) {
	if tenantID == "" {
		return IntelligenceResponse{}, errors.New("tenant_id é obrigatório")
	}
	start := time.Now()

	db, _, err := s.OpenTenant(ctx, tenantID)
	if err != nil {
		return IntelligenceResponse{}, err
	}
	defer db.Close()

	repo := NewRepository(db)
	schema, _ := repo.DetectMetricsSchema(ctx)

	incidents, err := repo.ListActiveIncidents(ctx, tenantID, 200)
	if err != nil {
		return IntelligenceResponse{}, err
	}

	score := computeHealthScore(incidents)
	status := StatusFromScore(score)
	top := topIncidents(incidents, 5)

	insights := make([]Insight, 0)
	recommendations := make([]Recommendation, 0)

	insights = append(insights, buildLatencyInsight(ctx, repo, schema, tenantID)...)
	insights = append(insights, buildCpuInsight(ctx, repo, schema, tenantID)...)
	insights = append(insights, buildIncidentBurstInsight(ctx, repo, tenantID)...)

	for _, inc := range top {
		recommendations = append(recommendations, Recommendation{
			Type:    inc.RootEvent,
			Message: RecommendForEvent(inc.RootEvent),
		})
	}

	trend := computeTrend(ctx, repo, tenantID)
	summary := buildSummary(score, status, trend, top, insights)

	log.Printf("%sintelligence tenant=%s score=%d incidents=%d took=%s",
		s.LogPrefix, tenantID, score, len(incidents), time.Since(start))

	return IntelligenceResponse{
		HealthScore:     score,
		Status:          status,
		Trend:           trend,
		Summary:         summary,
		TopIncidents:    top,
		Insights:        insights,
		Recommendations: recommendations,
	}, nil
}

func computeHealthScore(incidents []Incident) int {
	score := 100
	penaltyByCategory := map[string]int{}
	for _, inc := range incidents {
		penalty := SeverityWeight(inc.Severity)
		impact := inc.ImpactCount
		if impact < 1 {
			impact = 1
		}
		if impact > 5 {
			impact = 5
		}
		penalty = penalty * impact

		duration := time.Since(inc.CreatedAt)
		switch {
		case duration > 6*time.Hour:
			penalty -= 20
		case duration > 2*time.Hour:
			penalty -= 10
		case duration > 30*time.Minute:
			penalty -= 5
		}

		// limitar penalidade por categoria para evitar distorção
		cat := inc.RootEvent
		penaltyByCategory[cat] += penalty
		if penaltyByCategory[cat] < -40 {
			penaltyByCategory[cat] = -40
		}
	}
	for _, v := range penaltyByCategory {
		score += v
	}
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}

func topIncidents(incidents []Incident, limit int) []Incident {
	if limit <= 0 {
		limit = 5
	}
	sorted := make([]Incident, 0, len(incidents))
	sorted = append(sorted, incidents...)
	sort.Slice(sorted, func(i, j int) bool {
		pi := priorityScore(sorted[i])
		pj := priorityScore(sorted[j])
		if pi == pj {
			return sorted[i].UpdatedAt.After(sorted[j].UpdatedAt)
		}
		return pi > pj
	})
	for i := range sorted {
		sorted[i].Priority = priorityScore(sorted[i])
	}
	if len(sorted) > limit {
		sorted = sorted[:limit]
	}
	return sorted
}

func priorityScore(inc Incident) int {
	sev := SeverityRank(inc.Severity) * 100
	impact := inc.ImpactCount * 10
	dur := int(time.Since(inc.CreatedAt).Minutes())
	return sev + impact + dur
}

func buildLatencyInsight(ctx context.Context, repo *Repository, schema MetricsSchema, tenantID string) []Insight {
	window := 15 * time.Minute
	now := time.Now().UTC()

	metricNames := []string{"wan_latency_ms", "wan_latency", "wan_latency_avg_ms"}
	cur, ok1, err := repo.AvgMetric(ctx, schema, tenantID, metricNames, now.Add(-window), now)
	if err != nil || !ok1 {
		return nil
	}
	prev, ok2, err := repo.AvgMetric(ctx, schema, tenantID, metricNames, now.Add(-2*window), now.Add(-window))
	if err != nil || !ok2 || prev <= 0 {
		return nil
	}

	delta := (cur - prev) / prev
	if delta >= 0.30 {
		percent := delta * 100
		return []Insight{{
			Type:          "anomaly",
			Message:       "Latência média aumentou mais de 30% nas últimas janelas",
			Severity:      "warning",
			Metric:        "wan_latency_ms",
			ChangePercent: &percent,
			Context:       "Comparação entre as duas últimas janelas de 15m",
		}}
	}
	return nil
}

func buildCpuInsight(ctx context.Context, repo *Repository, schema MetricsSchema, tenantID string) []Insight {
	window := 30 * time.Minute
	now := time.Now().UTC()

	metricName := "cpu_percent"
	count, err := repo.CountMetricAbove(ctx, schema, tenantID, metricName, 90, now.Add(-window), now)
	if err != nil {
		return nil
	}
	if count >= 10 {
		return []Insight{{
			Type:     "anomaly",
			Message:  "CPU alta recorrente detectada nas últimas 30 minutos",
			Severity: "warning",
			Metric:   "cpu_percent",
			Context:  "Mais de 10 amostras acima de 90%",
		}}
	}
	return nil
}

func buildIncidentBurstInsight(ctx context.Context, repo *Repository, tenantID string) []Insight {
	window := 10 * time.Minute
	now := time.Now().UTC()

	count, err := repo.CountIncidentsInWindow(ctx, tenantID, now.Add(-window), now)
	if err != nil {
		return nil
	}
	if count >= 5 {
		return []Insight{{
			Type:     "instability",
			Message:  "Muitos incidentes em curto intervalo de tempo",
			Severity: "warning",
			Context:  ">=5 incidentes em 10 minutos",
		}}
	}
	return nil
}

func computeTrend(ctx context.Context, repo *Repository, tenantID string) string {
	now := time.Now().UTC()
	cur, err := repo.CountIncidentsInWindow(ctx, tenantID, now.Add(-30*time.Minute), now)
	if err != nil {
		return "stable"
	}
	prev, err := repo.CountIncidentsInWindow(ctx, tenantID, now.Add(-60*time.Minute), now.Add(-30*time.Minute))
	if err != nil {
		return "stable"
	}
	if cur > prev {
		return "degrading"
	}
	if cur < prev {
		return "improving"
	}
	return "stable"
}

func buildSummary(score int, status, trend string, top []Incident, insights []Insight) string {
	incCount := len(top)
	insCount := len(insights)
	base := "Ambiente estável."
	if status == "warning" {
		base = "Ambiente requer atenção."
	}
	if status == "critical" {
		base = "Ambiente crítico com alto risco."
	}
	return fmt.Sprintf("%s Health score %d. Trend %s. Top incidentes: %d. Insights ativos: %d.",
		base, score, trend, incCount, insCount)
}
