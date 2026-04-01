package config

import (
	"bufio"
	"errors"
	"os"
	"strings"
)

// Config representa a configuração mínima do agente.
type Config struct {
	ServerURL string // ex: https://nocguardian.bitbyteti.tec.br
	TenantID  string // UUID do tenant
	AgentID   string // ID gerado na primeira execução
	IngestURL string // opcional (override), ex: https://.../api/v1/<tenant>/metrics/ingest
	Services  []string // nomes de serviços a monitorar (comma-separated)
	PingTargets []string // alvos para ping (comma-separated)
}

// Load carrega um arquivo simples no formato "chave: valor".
func Load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	cfg := &Config{}

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// formato esperado: key: value
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		k := strings.TrimSpace(parts[0])
		// Remove BOM if present (common on Windows UTF-8 files)
		k = strings.TrimPrefix(k, "\ufeff")
		v := strings.TrimSpace(parts[1])
		v = strings.Trim(v, `"'`)

		switch strings.ToLower(k) {
		case "server_url":
			cfg.ServerURL = v
		case "tenant_id":
			cfg.TenantID = v
		case "agent_id":
			cfg.AgentID = v
		case "ingest_url":
			cfg.IngestURL = v
		case "services":
			cfg.Services = splitList(v)
		case "ping_targets":
			cfg.PingTargets = splitList(v)
		}
	}

	if err := sc.Err(); err != nil {
		return nil, err
	}

	if cfg.ServerURL == "" {
		return nil, errors.New("server_url is required")
	}
	if cfg.TenantID == "" {
		return nil, errors.New("tenant_id is required")
	}

	return cfg, nil
}

func splitList(v string) []string {
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t == "" {
			continue
		}
		out = append(out, t)
	}
	return out
}
