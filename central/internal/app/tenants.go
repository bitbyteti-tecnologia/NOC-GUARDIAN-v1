// tenants.go
// - Handlers de criação e listagem de Tenants.
// - Ao criar um Tenant:
//   1) insere no master.tenants
//   2) cria DB tenant_<slug>_<sufixo> (isolado por cliente)
//   3) roda migrações Timescale no DB do tenant
//
// Segurança:
// - db_name sanitizado via SlugifyDBName() + sufixo curto UUID (único)
// - Respeita limite de identificador do Postgres (~63)
// Robustez:
// - Se falhar migração: remove tenant do master e tenta DROP DATABASE (compensação)

package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Tenant struct {
	ID     uuid.UUID `json:"id"`
	Name   string    `json:"name"`
	DBName string    `json:"db_name"`
}

func CreateTenantHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string          `json:"name"`
		IPs  []string        `json:"ips,omitempty"`
		SNMP *SNMPCredential `json:"snmp,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		http.Error(w, "name is required", 400)
		return
	}

	tid := uuid.New()

	// db_name humano + seguro: tenant_<slug>_<8chars>
	slug := SlugifyDBName(req.Name)
	short := strings.ReplaceAll(tid.String(), "-", "")
	if len(short) > 8 {
		short = short[:8]
	}

	// Postgres identifier limit ~63
	const maxIdent = 63
	fixed := len("tenant_") + 1 + len(short) // "tenant_" + "_" + short
	maxSlug := maxIdent - fixed
	if maxSlug < 8 {
		maxSlug = 8
	}
	if len(slug) > maxSlug {
		slug = slug[:maxSlug]
	}
	dbName := "tenant_" + slug + "_" + short

	// 1) insere no master
	_, err := MasterConn.Exec(context.Background(),
		"INSERT INTO tenants (id, name, db_name) VALUES ($1,$2,$3)",
		tid, req.Name, dbName)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// 2) cria DB
	if err := CreateTenantDatabase(dbName); err != nil {
		// compensação: remove tenant
		_, _ = MasterConn.Exec(context.Background(), "DELETE FROM tenants WHERE id=$1", tid)
		http.Error(w, "Erro criando DB tenant: "+err.Error(), 500)
		return
	}

	// 3) migrações
	if err := RunTenantMigrations(dbName); err != nil {
		// compensação: remove tenant e dropa DB
		_, _ = MasterConn.Exec(context.Background(), "DELETE FROM tenants WHERE id=$1", tid)
		_, _ = MasterConn.Exec(context.Background(), fmt.Sprintf("DROP DATABASE IF EXISTS %s;", QuoteIdent(dbName)))
		http.Error(w, "Erro migrações tenant: "+err.Error(), 500)
		return
	}

	if len(req.IPs) > 0 || req.SNMP != nil {
		if err := SeedTenantDiscovery(tid.String(), dbName, req.IPs, req.SNMP); err != nil {
			http.Error(w, "Erro onboarding discovery: "+err.Error(), 500)
			return
		}
		_ = TriggerDiscovery(tid.String())
	}

	resp := Tenant{ID: tid, Name: req.Name, DBName: dbName}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func ListTenantsHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := MasterConn.Query(context.Background(),
		"SELECT id, name, db_name FROM tenants ORDER BY created_at DESC")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var out []Tenant
	for rows.Next() {
		var t Tenant
		if err := rows.Scan(&t.ID, &t.Name, &t.DBName); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		out = append(out, t)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func GetTenantHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := muxVar(r, "tenantID")
	if strings.TrimSpace(tenantID) == "" {
		http.Error(w, "tenant inválido", http.StatusBadRequest)
		return
	}

	var out struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		DBName string `json:"db_name"`
	}

	err := MasterConn.QueryRow(context.Background(),
		"SELECT id::text, name, db_name FROM tenants WHERE id=$1",
		tenantID,
	).Scan(&out.ID, &out.Name, &out.DBName)

	if err != nil {
		http.Error(w, "tenant not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func RunTenantMigrations(dbName string) error {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		getenv("MASTER_DB_USER", "guardian"),
		getenv("MASTER_DB_PASS", "guardian_strong_password"),
		getenv("MASTER_DB_HOST", "db"),
		getenv("MASTER_DB_PORT", "5432"),
		dbName,
	)
	conn, err := pgx.Connect(context.Background(), dsn)
	if err != nil {
		return err
	}
	defer conn.Close(context.Background())

	sql := `
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS timescaledb;
CREATE EXTENSION IF NOT EXISTS timescaledb_toolkit;

CREATE TABLE IF NOT EXISTS devices (
  id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
  hostname TEXT NOT NULL UNIQUE,
  ip TEXT NOT NULL DEFAULT '',
  ip_address TEXT NOT NULL DEFAULT '',
  type TEXT NOT NULL DEFAULT 'server',
  os TEXT NOT NULL DEFAULT 'linux',
  vendor TEXT,
  model TEXT,
  snmp_credential_id UUID,
  last_seen TIMESTAMPTZ,
  created_at TIMESTAMPTZ DEFAULT now()
);

ALTER TABLE devices
  ADD COLUMN IF NOT EXISTS ip_address TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS snmp_credential_id UUID;

CREATE TABLE IF NOT EXISTS device_relationships (
  id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
  tenant_id UUID NOT NULL,
  parent_device_id UUID NOT NULL,
  child_device_id UUID NOT NULL,
  relation_type TEXT NOT NULL DEFAULT 'uplink',
  discovered_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, parent_device_id, child_device_id)
);

CREATE INDEX IF NOT EXISTS idx_device_relationships_parent
  ON device_relationships (tenant_id, parent_device_id);

CREATE INDEX IF NOT EXISTS idx_device_relationships_child
  ON device_relationships (tenant_id, child_device_id);

CREATE TABLE IF NOT EXISTS snmp_credentials (
  id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
  tenant_id UUID NOT NULL,
  version TEXT NOT NULL DEFAULT 'v2c',
  community TEXT,
  username TEXT,
  auth_protocol TEXT,
  auth_password TEXT,
  priv_protocol TEXT,
  priv_password TEXT,
  created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS metrics (
  time TIMESTAMPTZ NOT NULL,
  device_id UUID NOT NULL,
  metric TEXT NOT NULL,
  value DOUBLE PRECISION NOT NULL,
  labels JSONB DEFAULT '{}'::jsonb
);

SELECT create_hypertable('metrics','time', if_not_exists => TRUE);

CREATE TABLE IF NOT EXISTS alerts (
  id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
  time TIMESTAMPTZ NOT NULL DEFAULT now(),
  severity TEXT NOT NULL,
  device_id UUID,
  summary TEXT NOT NULL,
  details JSONB DEFAULT '{}'::jsonb,
  rca JSONB DEFAULT '{}'::jsonb,
  status TEXT NOT NULL DEFAULT 'open'
);
`
	_, err = conn.Exec(context.Background(), sql)
	return err
}
