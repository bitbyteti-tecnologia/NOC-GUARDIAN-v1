// metrics.go
// - Recebe ingest de métricas do Proxy em lote (HTTP POST).
// - Aceita também payload do Agent (objeto) e converte para batch.
// - Valida tenant e grava em hypertable.
// - device_id no banco é UUID: se agent_id não for UUID, gera UUID determinístico.

package app

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
)

type MetricPoint struct {
	Time     string         `json:"time"`
	DeviceID string         `json:"device_id"`
	Metric   string         `json:"metric"`
	Value    float64        `json:"value"`
	Labels   map[string]any `json:"labels"`
}

// Payload do agente
type agentMetric struct {
	T string  `json:"t"`
	V float64 `json:"v"`
}

type AgentPayload struct {
	AgentID  string                 `json:"agent_id"`
	Hostname string                 `json:"hostname"`
	OS       string                 `json:"os"`
	Version  string                 `json:"version"`
	DiskPath string                 `json:"disk_path"`
	Metrics  map[string]agentMetric `json:"metrics"`
}

var uuidRe = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func isUUID(s string) bool {
	return uuidRe.MatchString(s)
}

// Gera UUID determinístico baseado em string (estável) - estilo UUIDv5
// (não depende de lib externa)
func uuidFromName(name string) string {
	sum := sha1.Sum([]byte("nocguardian-agent:" + name))
	b := sum[:16]

	// version 5
	b[6] = (b[6] & 0x0f) | 0x50
	// variant RFC4122
	b[8] = (b[8] & 0x3f) | 0x80

	hexs := hex.EncodeToString(b)
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hexs[0:8], hexs[8:12], hexs[12:16], hexs[16:20], hexs[20:32])
}

// Converte payload do agente para batch []MetricPoint (formato do banco)
func agentToBatch(ap AgentPayload) ([]MetricPoint, error) {
	labelsBase := map[string]any{
		"agent_id":  ap.AgentID,
		"hostname":  ap.Hostname,
		"os":        ap.OS,
		"version":   ap.Version,
		"disk_path": ap.DiskPath,
		"source":    "agent",
	}

	// device_id precisa ser UUID no banco
	deviceID := ap.AgentID
	if deviceID == "" {
		deviceID = ap.Hostname
	}
	if deviceID == "" {
		deviceID = "unknown"
	}

	// se não for UUID, gera determinístico e guarda original
	if !isUUID(deviceID) {
		labelsBase["original_agent_id"] = deviceID
		deviceID = uuidFromName(deviceID)
	}

	out := make([]MetricPoint, 0, len(ap.Metrics))
	for name, mv := range ap.Metrics {
		ts := mv.T
		if ts == "" {
			ts = time.Now().UTC().Format(time.RFC3339)
		} else {
			if _, err := time.Parse(time.RFC3339, ts); err != nil {
				return nil, fmt.Errorf("invalid metric timestamp for %s: %v", name, err)
			}
		}

		labels := make(map[string]any, len(labelsBase)+1)
		for k, v := range labelsBase {
			labels[k] = v
		}
		labels["metric_name"] = name

		out = append(out, MetricPoint{
			Time:     ts,
			DeviceID: deviceID,
			Metric:   name,
			Value:    mv.V,
			Labels:   labels,
		})
	}
	return out, nil
}

func MetricsIngestHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := muxVar(r, "tenantID")
	dbName, err := ResolveTenantDBName(tenantID)
	if err != nil {
		http.Error(w, "Tenant inválido", http.StatusNotFound)
		return
	}

	// Validação opcional de API Key
	apiKey := r.Header.Get("X-API-Key")
	if apiKey != "" {
		if err := ValidateApiKey(r.Context(), tenantID, apiKey); err != nil {
			http.Error(w, "Chave de API inválida", http.StatusUnauthorized)
			return
		}
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body error", http.StatusBadRequest)
		return
	}

	// 1) Tenta ler como batch []MetricPoint (formato original)
	var batch []MetricPoint
	if err := json.Unmarshal(body, &batch); err != nil {
		// 2) Se falhar, tenta ler como payload do agente (objeto)
		var ap AgentPayload
		if err2 := json.Unmarshal(body, &ap); err2 != nil {
			http.Error(w, fmt.Sprintf("decode error (batch/agent): %v", err), http.StatusBadRequest)
			return
		}
		converted, err3 := agentToBatch(ap)
		if err3 != nil {
			http.Error(w, fmt.Sprintf("agent payload error: %v", err3), http.StatusBadRequest)
			return
		}
		batch = converted
	}

	if len(batch) == 0 {
		http.Error(w, "empty batch", http.StatusBadRequest)
		return
	}

	// Conecta ao DB do tenant e insere em batch
	dsn := TenantDSN(dbName)
	conn, err := pgx.Connect(context.Background(), dsn)
	if err != nil {
		http.Error(w, "Conn tenant DB failed", http.StatusInternalServerError)
		return
	}
	defer conn.Close(context.Background())

	for _, p := range batch {
		labelsJSON, _ := json.Marshal(p.Labels)
		_, err := conn.Exec(context.Background(),
			`INSERT INTO metrics (time, device_id, metric, value, labels) 
             VALUES ($1, $2, $3, $4, $5)`,
			p.Time, p.DeviceID, p.Metric, p.Value, string(labelsJSON))
		if err != nil {
			http.Error(w, fmt.Sprintf("Insert error: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Atualiza metadados do host na tabela devices
	if len(batch) > 0 {
		var hostname, os, ip string
		if v, ok := batch[0].Labels["hostname"].(string); ok {
			hostname = v
		}
		if v, ok := batch[0].Labels["os"].(string); ok {
			os = v
		}
		if v, ok := batch[0].Labels["ip"].(string); ok {
			ip = v
		}

		if hostname != "" {
			_, _ = conn.Exec(context.Background(), `
                INSERT INTO devices (hostname, ip, type, os, last_seen)
                VALUES ($1, $2, 'server', $3, now())
                ON CONFLICT (hostname) DO UPDATE SET
                    os = EXCLUDED.os,
                    last_seen = now(),
                    ip = CASE WHEN EXCLUDED.ip != '' THEN EXCLUDED.ip ELSE devices.ip END
            `, hostname, ip, os)
		}
	}

	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte("accepted"))
}

func ResolveTenantDBName(tenantID string) (string, error) {
	var dbName string
	err := MasterConn.QueryRow(context.Background(),
		"SELECT db_name FROM tenants WHERE id=$1", tenantID).Scan(&dbName)
	return dbName, err
}

func TenantDSN(dbName string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		getenv("MASTER_DB_USER", "guardian"),
		getenv("MASTER_DB_PASS", "guardian_strong_password"),
		getenv("MASTER_DB_HOST", "db"),
		getenv("MASTER_DB_PORT", "5432"),
		dbName,
	)
}
