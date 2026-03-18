// rca.go
// - RCA inicial baseado em alertas recentes (sem topologia ainda).
// - Em produção: integrar dependências/topologia e regras específicas.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
)

func RCAHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := muxVar(r, "tenantID")
	if strings.TrimSpace(tenantID) == "" {
		http.Error(w, "tenant inválido", http.StatusBadRequest)
		return
	}

	type RCA struct {
		RootCause       string   `json:"root_cause"`
		Confidence      float64  `json:"confidence"`
		Why             []string `json:"why"`
		Recommendation  string   `json:"recommendation"`
		ImpactedDevices []string `json:"impacted_devices"`
		WindowMinutes   int      `json:"window_minutes"`
		TotalAlerts     int      `json:"total_alerts"`
		BasedOn         string   `json:"based_on"`
	}

	var req struct {
		WindowMinutes int `json:"window_minutes"`
		TopN          int `json:"top_n"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.WindowMinutes <= 0 {
		req.WindowMinutes = 60
	}
	if req.TopN <= 0 {
		req.TopN = 5
	}

	dbName, err := ResolveTenantDBName(tenantID)
	if err != nil {
		http.Error(w, "tenant inválido", http.StatusNotFound)
		return
	}

	dsn := TenantDSN(dbName)
	conn, err := pgx.Connect(context.Background(), dsn)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	defer conn.Close(context.Background())

	var totalAlerts int
	err = conn.QueryRow(context.Background(),
		`SELECT count(*) FROM alerts WHERE time >= now() - ($1::int || ' minutes')::interval`,
		req.WindowMinutes,
	).Scan(&totalAlerts)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	rows, err := conn.Query(context.Background(), `
        WITH recent AS (
            SELECT device_id::text AS device_id,
                   severity,
                   count(*) AS cnt
            FROM alerts
            WHERE time >= now() - ($1::int || ' minutes')::interval
              AND device_id IS NOT NULL
            GROUP BY device_id, severity
        ),
        ranked AS (
            SELECT device_id,
                   sum(cnt) AS total,
                   max(CASE severity
                        WHEN 'critical' THEN 4
                        WHEN 'high' THEN 3
                        WHEN 'medium' THEN 2
                        WHEN 'low' THEN 1
                        ELSE 0
                   END) AS sev_rank
            FROM recent
            GROUP BY device_id
        )
        SELECT device_id, total, sev_rank
        FROM ranked
        ORDER BY total DESC, sev_rank DESC
        LIMIT $2;
    `, req.WindowMinutes, req.TopN)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type row struct {
		Device string
		Total  int
		Sev    int
	}
	var top []row
	for rows.Next() {
		var rrow row
		if err := rows.Scan(&rrow.Device, &rrow.Total, &rrow.Sev); err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		top = append(top, rrow)
	}

	if totalAlerts == 0 || len(top) == 0 {
		resp := RCA{
			RootCause:      "Sem alertas recentes para análise",
			Confidence:     0.1,
			Why:            []string{"Nenhum alerta no período analisado"},
			Recommendation: "Aguardar novos eventos ou ampliar a janela de análise",
			WindowMinutes:  req.WindowMinutes,
			TotalAlerts:    totalAlerts,
			BasedOn:        "alerts",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	topDevice := top[0]
	confidence := 0.2
	if totalAlerts > 0 {
		confidence = float64(topDevice.Total) / float64(totalAlerts)
		if confidence > 0.95 {
			confidence = 0.95
		}
	}

	impacted := make([]string, 0, len(top))
	for _, t := range top {
		impacted = append(impacted, t.Device)
	}

	resp := RCA{
		RootCause:       fmt.Sprintf("Dispositivo %s concentra %d alertas no período", topDevice.Device, topDevice.Total),
		Confidence:      confidence,
		Why:             []string{fmt.Sprintf("Total de alertas no período: %d", totalAlerts)},
		Recommendation:  "Verificar o dispositivo com maior concentração de alertas e seus dependentes imediatos",
		ImpactedDevices: impacted,
		WindowMinutes:   req.WindowMinutes,
		TotalAlerts:     totalAlerts,
		BasedOn:         "alerts",
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
