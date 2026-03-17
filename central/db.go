// db.go
// - Conecta no MASTER DB.
// - Migrações idempotentes: extensões, tabelas, colunas, índices.
// - Cria users, tenants, refresh_tokens, password_resets.

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var MasterConn *pgxpool.Pool

func InitMasterDB() error {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		getenv("MASTER_DB_USER", "guardian"),
		getenv("MASTER_DB_PASS", "guardian_strong_password"),
		getenv("MASTER_DB_HOST", "db"),
		getenv("MASTER_DB_PORT", "5432"),
		getenv("MASTER_DB_NAME", "guardian_master"),
	)
	var err error
	MasterConn, err = pgxpool.New(context.Background(), dsn)
	if err != nil {
		return err
	}

	// Testa conexão
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return MasterConn.Ping(ctx)
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func RunMasterMigrations() error {
	sql := `
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Usuários globais e por tenant
CREATE TABLE IF NOT EXISTS users (
  id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
  email TEXT UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,
  role TEXT NOT NULL DEFAULT 'admin',
  created_at TIMESTAMPTZ DEFAULT now()
);
ALTER TABLE users
  ADD COLUMN IF NOT EXISTS tenant_id UUID NULL;
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname='public' AND indexname='idx_users_tenant_id') THEN
    CREATE INDEX idx_users_tenant_id ON users(tenant_id);
  END IF;
END$$;
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conrelid='public.users'::regclass AND conname='users_email_key'
  ) THEN
    ALTER TABLE users ADD CONSTRAINT users_email_key UNIQUE (email);
  END IF;
END$$;

-- Tenants
CREATE TABLE IF NOT EXISTS tenants (
  id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
  name TEXT NOT NULL,
  db_name TEXT UNIQUE NOT NULL,
  created_at TIMESTAMPTZ DEFAULT now()
);

-- Refresh tokens (sessão de longa duração, cookie HttpOnly)
CREATE TABLE IF NOT EXISTS refresh_tokens (
  id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL,
  user_agent TEXT,
  ip TEXT,
  expires_at TIMESTAMPTZ NOT NULL,
  revoked_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ DEFAULT now()
);
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname='public' AND indexname='idx_rt_user') THEN
    CREATE INDEX idx_rt_user ON refresh_tokens(user_id);
  END IF;
END$$;
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname='public' AND indexname='idx_rt_expires') THEN
    CREATE INDEX idx_rt_expires ON refresh_tokens(expires_at);
  END IF;
END$$;

-- Password reset (token de recuperação)
CREATE TABLE IF NOT EXISTS password_resets (
  id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  used_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ DEFAULT now()
);
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname='public' AND indexname='idx_pr_user') THEN
    CREATE INDEX idx_pr_user ON password_resets(user_id);
  END IF;
END$$;
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname='public' AND indexname='idx_pr_expires') THEN
    CREATE INDEX idx_pr_expires ON password_resets(expires_at);
  END IF;
END$$;

-- API Keys (Para agentes e integrações)
CREATE TABLE IF NOT EXISTS api_keys (
  id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  key_prefix TEXT NOT NULL,
  key_hash TEXT NOT NULL,
  name TEXT,
  created_at TIMESTAMPTZ DEFAULT now(),
  last_used_at TIMESTAMPTZ
);
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname='public' AND indexname='idx_ak_tenant') THEN
    CREATE INDEX idx_ak_tenant ON api_keys(tenant_id);
  END IF;
END$$;
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname='public' AND indexname='idx_ak_prefix') THEN
    CREATE INDEX idx_ak_prefix ON api_keys(key_prefix);
  END IF;
END$$;
`
	_, err := MasterConn.Exec(context.Background(), sql)
	return err
}

func CreateTenantDatabase(dbName string) error {
	_, err := MasterConn.Exec(context.Background(), fmt.Sprintf("CREATE DATABASE %s;", QuoteIdent(dbName)))
	return err
}

func QuoteIdent(s string) string {
	return `"` + s + `"`
}
