package main

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "strconv"
    "strings"
    "time"

    "github.com/jackc/pgx/v5"
)

type MetricLatest struct {
    Time   time.Time       `json:"time"`
    Metric string          `json:"metric"`
    Value  float64         `json:"value"`
    Labels json.RawMessage `json:"labels"`
}

type MetricBucket struct {
    Bucket time.Time `json:"bucket"`
    Avg    float64   `json:"avg"`
    Min    float64   `json:"min"`
    Max    float64   `json:"max"`
    Count  int64     `json:"count"`
}

// GET /api/v1/{tenantID}/metrics/latest?device_id=...&metric=a,b,c
func MetricsLatestHandler(w http.ResponseWriter, r *http.Request) {
    tenantID := muxVar(r, "tenantID")
    dbName, err := ResolveTenantDBName(tenantID)
    if err != nil {
        http.Error(w, "Tenant inválido", http.StatusNotFound)
        return
    }

    deviceID := r.URL.Query().Get("device_id")
    metrics := r.URL.Query().Get("metric")
    if deviceID == "" || metrics == "" {
        http.Error(w, "device_id and metric are required", http.StatusBadRequest)
        return
    }

    metricList := strings.Split(metrics, ",")
    for i := range metricList {
        metricList[i] = strings.TrimSpace(metricList[i])
    }
    if len(metricList) == 0 {
        http.Error(w, "metric list is empty", http.StatusBadRequest)
        return
    }

    conn, err := pgx.Connect(context.Background(), TenantDSN(dbName))
    if err != nil {
        http.Error(w, "Conn tenant DB failed", http.StatusInternalServerError)
        return
    }
    defer conn.Close(context.Background())

    // Parametrizado: device_id e lista de métricas
    rows, err := conn.Query(context.Background(), `
        SELECT DISTINCT ON (metric)
            time, metric, value, labels::text
        FROM metrics
        WHERE device_id::text = $1
          AND metric = ANY($2)
        ORDER BY metric, time DESC
    `, deviceID, metricList)
    if err != nil {
        http.Error(w, fmt.Sprintf("query latest failed: %v", err), http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    out := []MetricLatest{}
    for rows.Next() {
        var t time.Time
        var m string
        var v float64
        var labelsText string
        if err := rows.Scan(&t, &m, &v, &labelsText); err != nil {
            http.Error(w, fmt.Sprintf("scan latest failed: %v", err), http.StatusInternalServerError)
            return
        }
        out = append(out, MetricLatest{
            Time:   t,
            Metric: m,
            Value:  v,
            Labels: json.RawMessage(labelsText),
        })
    }

    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(out)
}

// GET /api/v1/{tenantID}/metrics/range?device_id=...&metric=cpu_percent&from=RFC3339&to=RFC3339&step_sec=60
func MetricsRangeHandler(w http.ResponseWriter, r *http.Request) {
    tenantID := muxVar(r, "tenantID")
    dbName, err := ResolveTenantDBName(tenantID)
    if err != nil {
        http.Error(w, "Tenant inválido", http.StatusNotFound)
        return
    }

    deviceID := r.URL.Query().Get("device_id")
    metric := r.URL.Query().Get("metric")
    fromStr := r.URL.Query().Get("from")
    toStr := r.URL.Query().Get("to")
    stepStr := r.URL.Query().Get("step_sec")

    if deviceID == "" || metric == "" || fromStr == "" || toStr == "" {
        http.Error(w, "device_id, metric, from, to are required (RFC3339)", http.StatusBadRequest)
        return
    }

    fromT, err := time.Parse(time.RFC3339, fromStr)
    if err != nil {
        http.Error(w, "invalid from (RFC3339)", http.StatusBadRequest)
        return
    }
    toT, err := time.Parse(time.RFC3339, toStr)
    if err != nil {
        http.Error(w, "invalid to (RFC3339)", http.StatusBadRequest)
        return
    }
    if toT.Before(fromT) {
        http.Error(w, "to must be >= from", http.StatusBadRequest)
        return
    }

    conn, err := pgx.Connect(context.Background(), TenantDSN(dbName))
    if err != nil {
        http.Error(w, "Conn tenant DB failed", http.StatusInternalServerError)
        return
    }
    defer conn.Close(context.Background())

    // Bucket agregado (Timescale)
    if stepStr != "" {
        stepSec, err := strconv.Atoi(stepStr)
        if err != nil || stepSec <= 0 {
            http.Error(w, "invalid step_sec", http.StatusBadRequest)
            return
        }

        rows, err := conn.Query(context.Background(), `
            SELECT
                time_bucket(($5 * interval '1 second'), time) AS bucket,
                avg(value) AS avg,
                min(value) AS min,
                max(value) AS max,
                count(*) AS count
            FROM metrics
            WHERE device_id::text = $1
              AND metric = $2
              AND time >= $3
              AND time <= $4
            GROUP BY bucket
            ORDER BY bucket ASC
        `, deviceID, metric, fromT, toT, stepSec)
        if err != nil {
            http.Error(w, fmt.Sprintf("range bucket query failed: %v", err), http.StatusInternalServerError)
            return
        }
        defer rows.Close()

        out := []MetricBucket{}
        for rows.Next() {
            var b MetricBucket
            if err := rows.Scan(&b.Bucket, &b.Avg, &b.Min, &b.Max, &b.Count); err != nil {
                http.Error(w, fmt.Sprintf("scan bucket failed: %v", err), http.StatusInternalServerError)
                return
            }
            out = append(out, b)
        }

        w.Header().Set("Content-Type", "application/json")
        _ = json.NewEncoder(w).Encode(out)
        return
    }

    // Sem bucket: pontos crus
    type Point struct {
        Time  time.Time `json:"time"`
        Value float64   `json:"value"`
    }
    rows, err := conn.Query(context.Background(), `
        SELECT time, value
        FROM metrics
        WHERE device_id::text = $1
          AND metric = $2
          AND time >= $3
          AND time <= $4
        ORDER BY time ASC
        LIMIT 5000
    `, deviceID, metric, fromT, toT)
    if err != nil {
        http.Error(w, fmt.Sprintf("range raw query failed: %v", err), http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    out := []Point{}
    for rows.Next() {
        var p Point
        if err := rows.Scan(&p.Time, &p.Value); err != nil {
            http.Error(w, fmt.Sprintf("scan raw failed: %v", err), http.StatusInternalServerError)
            return
        }
        out = append(out, p)
    }

    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(out)
}
