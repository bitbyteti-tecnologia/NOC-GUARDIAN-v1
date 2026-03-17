package main

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "time"

    "github.com/jackc/pgx/v5"
)

type AgentSummary struct {
    DeviceID         string     `json:"device_id"`
    AgentIDOriginal  string     `json:"agent_id_original"`
    Hostname         string     `json:"hostname"`
    OS               string     `json:"os"`
    Version          string     `json:"version"`
    DiskPath         string     `json:"disk_path"`
    IP               string     `json:"ip"`
    LastSeen         *time.Time `json:"last_seen"`
    Online           bool       `json:"online"`

    CPUPercent  *float64 `json:"cpu_percent"`
    MemUsedPct  *float64 `json:"mem_used_pct"`
    DiskUsedPct *float64 `json:"disk_used_pct"`
}

func labelStr(labels map[string]any, key string) string {
    v, ok := labels[key]
    if !ok || v == nil {
        return ""
    }
    if s, ok := v.(string); ok {
        return s
    }
    b, _ := json.Marshal(v)
    return string(b)
}

// GET /api/v1/{tenantID}/agents
func AgentsListHandler(w http.ResponseWriter, r *http.Request) {
    tenantID := muxVar(r, "tenantID")
    dbName, err := ResolveTenantDBName(tenantID)
    if err != nil {
        http.Error(w, "Tenant inválido", http.StatusNotFound)
        return
    }

    conn, err := pgx.Connect(context.Background(), TenantDSN(dbName))
    if err != nil {
        http.Error(w, "Conn tenant DB failed", http.StatusInternalServerError)
        return
    }
    defer conn.Close(context.Background())

    now := time.Now().UTC()
    onlineWindow := 2 * time.Minute

    heartbeatMetric := "agent_heartbeat"

    // ✅ Query heartbeat parametrizada
    rows, err := conn.Query(context.Background(), `
        SELECT DISTINCT ON (device_id)
            device_id::text AS device_id,
            time AS last_seen,
            labels::text AS labels_text
        FROM metrics
        WHERE metric = $1
        ORDER BY device_id, time DESC
    `, heartbeatMetric)
    if err != nil {
        http.Error(w, fmt.Sprintf("query heartbeat failed: %v", err), http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    agents := map[string]*AgentSummary{}
    for rows.Next() {
        var deviceID string
        var lastSeen time.Time
        var labelsText *string

        if err := rows.Scan(&deviceID, &lastSeen, &labelsText); err != nil {
            http.Error(w, fmt.Sprintf("scan heartbeat failed: %v", err), http.StatusInternalServerError)
            return
        }

        labels := map[string]any{}
        if labelsText != nil && *labelsText != "" {
            _ = json.Unmarshal([]byte(*labelsText), &labels)
        }

        a := &AgentSummary{
            DeviceID:        deviceID,
            Hostname:        labelStr(labels, "hostname"),
            OS:              labelStr(labels, "os"),
            Version:         labelStr(labels, "version"),
            DiskPath:        labelStr(labels, "disk_path"),
            IP:              labelStr(labels, "ip"),
            AgentIDOriginal: labelStr(labels, "original_agent_id"),
            LastSeen:        &lastSeen,
            Online:          now.Sub(lastSeen.UTC()) <= onlineWindow,
        }
        if a.Hostname == "" {
            a.Hostname = deviceID
        }
        agents[deviceID] = a
    }

    // ✅ Query latest metrics parametrizada (nada de literais no SQL)
    metricList := []string{"cpu_percent", "mem_used_pct", "disk_used_pct"}

    rows2, err := conn.Query(context.Background(), `
        SELECT DISTINCT ON (device_id, metric)
            device_id::text AS device_id,
            metric,
            value
        FROM metrics
        WHERE metric = ANY($1)
        ORDER BY device_id, metric, time DESC
    `, metricList)
    if err != nil {
        http.Error(w, fmt.Sprintf("query latest metrics failed: %v", err), http.StatusInternalServerError)
        return
    }
    defer rows2.Close()

    for rows2.Next() {
        var deviceID, metric string
        var value float64
        if err := rows2.Scan(&deviceID, &metric, &value); err != nil {
            http.Error(w, fmt.Sprintf("scan latest metrics failed: %v", err), http.StatusInternalServerError)
            return
        }

        a, ok := agents[deviceID]
        if !ok {
            a = &AgentSummary{DeviceID: deviceID, Hostname: deviceID, Online: false}
            agents[deviceID] = a
        }

        switch metric {
        case "cpu_percent":
            a.CPUPercent = &value
        case "mem_used_pct":
            a.MemUsedPct = &value
        case "disk_used_pct":
            a.DiskUsedPct = &value
        }
    }

    out := make([]*AgentSummary, 0, len(agents))
    for _, a := range agents {
        out = append(out, a)
    }

    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(out)
}
