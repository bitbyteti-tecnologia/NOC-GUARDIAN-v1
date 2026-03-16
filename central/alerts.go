// alerts.go
// - Implementa o endpoint de listagem de alertas por tenant:
//   * GET /api/v1/{tenantID}/alerts  -> ListAlertsHandler
//
// Interação no sistema:
// - Lê a tabela "alerts" do DB do tenant (criada em RunTenantMigrations).
// - Esse endpoint alimenta: Dashboard (Bloco 5) e tela /alerts na UI.

package main

import (
    "context"
    "encoding/json"
    "net/http"
    "time"

    "github.com/gorilla/mux"
    "github.com/jackc/pgx/v5"
)

type Alert struct {
    ID        string                 `json:"id"`
    Time      time.Time              `json:"time"`
    Severity  string                 `json:"severity"`
    DeviceID  *string                `json:"device_id,omitempty"`
    Summary   string                 `json:"summary"`
    Details   map[string]any         `json:"details"`
    RCA       map[string]any         `json:"rca"`
    Status    string                 `json:"status"`
}

func ListAlertsHandler(w http.ResponseWriter, r *http.Request) {
    tenantID := mux.Vars(r)["tenantID"]
    dbName, err := ResolveTenantDBName(tenantID)
    if err != nil {
        http.Error(w, "Tenant inválido", 404)
        return
    }

    conn, err := pgx.Connect(context.Background(), TenantDSN(dbName))
    if err != nil {
        http.Error(w, "Conn tenant DB failed: "+err.Error(), 500)
        return
    }
    defer conn.Close(context.Background())

    rows, err := conn.Query(context.Background(), `
        SELECT id::text, time, severity, device_id::text, summary, details, rca, status
        FROM alerts
        ORDER BY time DESC
        LIMIT 200`)
    if err != nil {
        http.Error(w, err.Error(), 500)
        return
    }
    defer rows.Close()

    var out []Alert
    for rows.Next() {
        var a Alert
        var deviceID *string
        var detailsJSON, rcaJSON []byte
        if err := rows.Scan(&a.ID, &a.Time, &a.Severity, &deviceID, &a.Summary, &detailsJSON, &rcaJSON, &a.Status); err != nil {
            http.Error(w, err.Error(), 500)
            return
        }
        a.DeviceID = deviceID
        _ = json.Unmarshal(detailsJSON, &a.Details)
        _ = json.Unmarshal(rcaJSON, &a.RCA)
        out = append(out, a)
    }

    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(out)
}
