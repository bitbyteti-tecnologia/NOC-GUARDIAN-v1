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
