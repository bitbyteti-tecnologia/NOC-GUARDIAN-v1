package main

import (
    "context"
    "database/sql"
    "encoding/json"
    "errors"
    "fmt"
    "log"
    "net/http"
    "os"
    "strconv"
    "strings"
    "time"

    "github.com/go-chi/chi/v5"
    _ "github.com/jackc/pgx/v5/stdlib"
)

type Config struct {
    ListenAddr     string
    MasterHost     string
    MasterPort     string
    MasterUser     string
    MasterPass     string
    MasterDB       string
    DashboardSecret string
}

func getenv(k, def string) string {
    v := strings.TrimSpace(os.Getenv(k))
    if v == "" {
        return def
    }
    return v
}

func mustConfig() Config {
    cfg := Config{
        ListenAddr:     getenv("DASH_LISTEN", ":8090"),
        MasterHost:     getenv("MASTER_DB_HOST", "db"),
        MasterPort:     getenv("MASTER_DB_PORT", "5432"),
        MasterUser:     getenv("MASTER_DB_USER", "guardian"),
        MasterPass:     getenv("MASTER_DB_PASS", ""),
        MasterDB:       getenv("MASTER_DB_NAME", "guardian_master"),
        DashboardSecret: getenv("DASHBOARD_SECRET", ""),
    }
    if cfg.DashboardSecret == "" {
        log.Println("[WARN] DASHBOARD_SECRET vazio; endpoints ficarão sem proteção por header.")
    }
    return cfg
}

func dsn(host, port, user, pass, dbname string) string {
    // SSL desabilitado porque a comunicação é interna na rede Docker
    return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
        urlenc(user), urlenc(pass), host, port, dbname)
}

func urlenc(s string) string {
    // simples: o pgx aceita caracteres comuns, mas isso evita problemas com # etc.
    return strings.ReplaceAll(s, "#", "%23")
}

func openDB(ctx context.Context, dsn string) (*sql.DB, error) {
    db, err := sql.Open("pgx", dsn)
    if err != nil {
        return nil, err
    }
    db.SetMaxOpenConns(10)
    db.SetMaxIdleConns(5)
    db.SetConnMaxLifetime(10 * time.Minute)

    // ping com timeout
    c, cancel := context.WithTimeout(ctx, 3*time.Second)
    defer cancel()
    if err := db.PingContext(c); err != nil {
        _ = db.Close()
        return nil, err
    }
    return db, nil
}

func main() {
    cfg := mustConfig()

    r := chi.NewRouter()

    // Middleware simples de proteção (via Nginx)
    r.Use(func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
            if cfg.DashboardSecret != "" {
                if req.Header.Get("X-Dashboard-Secret") != cfg.DashboardSecret {
                    http.Error(w, "forbidden", http.StatusForbidden)
                    return
                }
            }
            next.ServeHTTP(w, req)
        })
    })

    r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })

    // Endpoints esperados pelo UI
    r.Get("/api/v1/tenants/{tenantId}/dashboard/summary", func(w http.ResponseWriter, r *http.Request) {
        tenantId := chi.URLParam(r, "tenantId")
        resp, err := handleSummary(r.Context(), cfg, tenantId)
        writeJSON(w, resp, err)
    })

    r.Get("/api/v1/tenants/{tenantId}/dashboard/hosts", func(w http.ResponseWriter, r *http.Request) {
        tenantId := chi.URLParam(r, "tenantId")
        resp, err := handleHosts(r.Context(), cfg, tenantId)
        writeJSON(w, resp, err)
    })

    r.Get("/api/v1/tenants/{tenantId}/dashboard/series", func(w http.ResponseWriter, r *http.Request) {
        tenantId := chi.URLParam(r, "tenantId")
        hostname := r.URL.Query().Get("hostname")
        metric := r.URL.Query().Get("metric")
        window := r.URL.Query().Get("window") // "1h" ou "24h"
        if hostname == "" || metric == "" {
            http.Error(w, "hostname e metric são obrigatórios", http.StatusBadRequest)
            return
        }
        if window == "" {
            window = "24h"
        }
        resp, err := handleSeries(r.Context(), cfg, tenantId, hostname, metric, window)
        writeJSON(w, resp, err)
    })

    r.Get("/api/v1/tenants/{tenantId}/dashboard/host/{hostname}/inventory/latest", HostInventoryLatestHandler)

    log.Printf("dashboard-api listening on %s\n", cfg.ListenAddr)
    if err := http.ListenAndServe(cfg.ListenAddr, r); err != nil {
        log.Fatal(err)
    }
}

func writeJSON(w http.ResponseWriter, v any, err error) {
    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    if err != nil {
        code := http.StatusInternalServerError
        if errors.Is(err, sql.ErrNoRows) {
            code = http.StatusNotFound
        }
        log.Printf("[ERROR] %v\n", err)
        http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), code)
        return
    }
    enc := json.NewEncoder(w)
    enc.SetIndent("", "  ")
    _ = enc.Encode(v)
}

func tenantDBName(ctx context.Context, cfg Config, tenantId string) (string, error) {
    master, err := openDB(ctx, dsn(cfg.MasterHost, cfg.MasterPort, cfg.MasterUser, cfg.MasterPass, cfg.MasterDB))
    if err != nil {
        return "", fmt.Errorf("master db: %w", err)
    }
    defer master.Close()

    var dbname string
    q := `SELECT db_name FROM public.tenants WHERE id = $1`
    c, cancel := context.WithTimeout(ctx, 2*time.Second)
    defer cancel()
    if err := master.QueryRowContext(c, q, tenantId).Scan(&dbname); err != nil {
        return "", err
    }
    return dbname, nil
}

func openTenant(ctx context.Context, cfg Config, tenantId string) (*sql.DB, string, error) {
    dbname, err := tenantDBName(ctx, cfg, tenantId)
    if err != nil {
        return nil, "", err
    }
    tenant, err := openDB(ctx, dsn(cfg.MasterHost, cfg.MasterPort, cfg.MasterUser, cfg.MasterPass, dbname))
    if err != nil {
        return nil, "", fmt.Errorf("tenant db %s: %w", dbname, err)
    }
    return tenant, dbname, nil
}

// ---------- handlers ----------

type SummaryResp struct {
    TotalHosts        int       `json:"total_hosts"`
    Online            int       `json:"online"`
    Offline           int       `json:"offline"`
    LastAnyHeartbeat  time.Time `json:"last_any_heartbeat"`
}

func handleSummary(ctx context.Context, cfg Config, tenantId string) (SummaryResp, error) {
    tdb, _, err := openTenant(ctx, cfg, tenantId)
    if err != nil {
        return SummaryResp{}, err
    }
    defer tdb.Close()

    q := `
WITH last_seen AS (
  SELECT
    labels->>'hostname' AS hostname,
    max(time) AS last_heartbeat
  FROM public.metrics
  WHERE metric = 'agent_heartbeat'
  GROUP BY labels->>'hostname'
)
SELECT
  count(*) AS total_hosts,
  count(*) FILTER (WHERE now() - last_heartbeat <= interval '2 minutes') AS online,
  count(*) FILTER (WHERE now() - last_heartbeat >  interval '2 minutes') AS offline,
  coalesce(max(last_heartbeat), now()) AS last_any_heartbeat
FROM last_seen;`

    var out SummaryResp
    c, cancel := context.WithTimeout(ctx, 3*time.Second)
    defer cancel()
    err = tdb.QueryRowContext(c, q).Scan(&out.TotalHosts, &out.Online, &out.Offline, &out.LastAnyHeartbeat)
    return out, err
}

type HostRow struct {
    Hostname    string    `json:"hostname"`
    OS          string    `json:"os"`
    AgentID     string    `json:"agent_id"`
    LastSeen    time.Time `json:"last_seen"`
    CPU         *float64  `json:"cpu_percent"`
    Mem         *float64  `json:"mem_used_pct"`
    Disk        *float64  `json:"disk_used_pct"`
    Status      string    `json:"status"`
}

func handleHosts(ctx context.Context, cfg Config, tenantId string) ([]HostRow, error) {
    tdb, _, err := openTenant(ctx, cfg, tenantId)
    if err != nil {
        return nil, err
    }
    defer tdb.Close()

    q := `
WITH last_per_metric AS (
  SELECT DISTINCT ON (labels->>'hostname', metric)
    labels->>'hostname' AS hostname,
    labels->>'os'       AS os,
    labels->>'agent_id' AS agent_id,
    metric,
    time,
    value
  FROM public.metrics
  WHERE metric IN ('cpu_percent','mem_used_pct','disk_used_pct','agent_heartbeat')
  ORDER BY (labels->>'hostname'), metric, time DESC
),
pivot AS (
  SELECT
    hostname,
    max(os)       AS os,
    max(agent_id) AS agent_id,
    max(time) FILTER (WHERE metric='agent_heartbeat') AS last_seen,
    max(value) FILTER (WHERE metric='cpu_percent')    AS cpu_percent,
    max(value) FILTER (WHERE metric='mem_used_pct')   AS mem_used_pct,
    max(value) FILTER (WHERE metric='disk_used_pct')  AS disk_used_pct
  FROM last_per_metric
  GROUP BY hostname
)
SELECT
  hostname, os, agent_id, last_seen, cpu_percent, mem_used_pct, disk_used_pct,
  CASE WHEN now() - last_seen <= interval '2 minutes' THEN 'ONLINE' ELSE 'OFFLINE' END AS status
FROM pivot
ORDER BY status DESC, hostname;`

    c, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    rows, err := tdb.QueryContext(c, q)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var out []HostRow
    for rows.Next() {
        var r HostRow
        if err := rows.Scan(&r.Hostname, &r.OS, &r.AgentID, &r.LastSeen, &r.CPU, &r.Mem, &r.Disk, &r.Status); err != nil {
            return nil, err
        }
        out = append(out, r)
    }
    return out, rows.Err()
}

type SeriesPoint struct {
    T time.Time `json:"t"`
    V float64   `json:"v"`
}
type SeriesResp struct {
    Hostname string       `json:"hostname"`
    Metric   string       `json:"metric"`
    Window   string       `json:"window"`
    Points   []SeriesPoint `json:"points"`
}

func parseWindow(w string) (time.Duration, error) {
    w = strings.TrimSpace(strings.ToLower(w))
    if strings.HasSuffix(w, "h") {
        n, err := strconv.Atoi(strings.TrimSuffix(w, "h"))
        if err != nil || n <= 0 {
            return 0, fmt.Errorf("window inválida: %s", w)
        }
        return time.Duration(n) * time.Hour, nil
    }
    if strings.HasSuffix(w, "m") {
        n, err := strconv.Atoi(strings.TrimSuffix(w, "m"))
        if err != nil || n <= 0 {
            return 0, fmt.Errorf("window inválida: %s", w)
        }
        return time.Duration(n) * time.Minute, nil
    }
    return 0, fmt.Errorf("window inválida: %s (use 1h, 24h, 15m etc)", w)
}

func handleSeries(ctx context.Context, cfg Config, tenantId, hostname, metricName, window string) (SeriesResp, error) {
    tdb, _, err := openTenant(ctx, cfg, tenantId)
    if err != nil {
        return SeriesResp{}, err
    }
    defer tdb.Close()

    dur, err := parseWindow(window)
    if err != nil {
        return SeriesResp{}, err
    }

    // bucket: 1m para 24h, 10s para 1h (ajuste se quiser)
    bucket := "1 minute"
    if dur <= 2*time.Hour {
        bucket = "10 seconds"
    }

    q := fmt.Sprintf(`
SELECT
  time_bucket('%s', time) AS bucket,
  avg(value) AS v
FROM public.metrics
WHERE metric = $1
  AND labels->>'hostname' = $2
  AND time >= now() - $3::interval
GROUP BY bucket
ORDER BY bucket;`, bucket)

    intervalStr := fmt.Sprintf("%d seconds", int(dur.Seconds()))

    c, cancel := context.WithTimeout(ctx, 8*time.Second)
    defer cancel()
    rows, err := tdb.QueryContext(c, q, metricName, hostname, intervalStr)
    if err != nil {
        return SeriesResp{}, err
    }
    defer rows.Close()

    var pts []SeriesPoint
    for rows.Next() {
        var p SeriesPoint
        if err := rows.Scan(&p.T, &p.V); err != nil {
            return SeriesResp{}, err
        }
        pts = append(pts, p)
    }

    return SeriesResp{
        Hostname: hostname,
        Metric:   metricName,
        Window:   window,
        Points:   pts,
    }, rows.Err()
}
