package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"
)

type SNMPCredential struct {
	Version      string `json:"version"`
	Community    string `json:"community"`
	Username     string `json:"username"`
	AuthProtocol string `json:"auth_protocol"`
	AuthPassword string `json:"auth_password"`
	PrivProtocol string `json:"priv_protocol"`
	PrivPassword string `json:"priv_password"`
}

type DiscoveryRequest struct {
	TenantID string          `json:"tenant_id"`
	IPs      []string        `json:"ips"`
	SNMP     *SNMPCredential `json:"snmp"`
}

func TriggerDiscovery(tenantID string) error {
	url := getenv("DISCOVERY_URL", "http://discovery:8085") + "/discovery/run"
	payload := DiscoveryRequest{TenantID: tenantID}
	body, _ := json.Marshal(payload)

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("discovery status %d", resp.StatusCode)
	}
	return nil
}

func SeedTenantDiscovery(tenantID, dbName string, ips []string, snmp *SNMPCredential) error {
	conn, err := pgx.Connect(context.Background(), TenantDSN(dbName))
	if err != nil {
		return err
	}
	defer conn.Close(context.Background())

	_, _ = conn.Exec(context.Background(), `
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

ALTER TABLE devices
  ADD COLUMN IF NOT EXISTS ip_address TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS snmp_credential_id UUID;
`)

	var credID *string
	if snmp != nil {
		var id string
		err = conn.QueryRow(context.Background(), `
INSERT INTO snmp_credentials (tenant_id, version, community, username, auth_protocol, auth_password, priv_protocol, priv_password)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id::text`, tenantID, safe(snmp.Version, "v2c"), snmp.Community, snmp.Username, snmp.AuthProtocol, snmp.AuthPassword, snmp.PrivProtocol, snmp.PrivPassword).Scan(&id)
		if err != nil {
			return err
		}
		credID = &id
	}

	for _, ip := range ips {
		ip = strings.TrimSpace(ip)
		if ip == "" {
			continue
		}
		if net.ParseIP(ip) == nil {
			return errors.New("IP inválido: " + ip)
		}
		hostname := ip
		_, err := conn.Exec(context.Background(), `
INSERT INTO devices (hostname, ip, ip_address, type, os, snmp_credential_id)
VALUES ($1, $2, $3, 'network', 'unknown', NULLIF($4,'')::uuid)
ON CONFLICT (hostname) DO UPDATE
  SET ip = EXCLUDED.ip,
      ip_address = EXCLUDED.ip_address,
      snmp_credential_id = COALESCE(EXCLUDED.snmp_credential_id, devices.snmp_credential_id)
`, hostname, ip, ip, deref(credID))
		if err != nil {
			return err
		}
	}
	return nil
}

func DiscoveryRunHandler(w http.ResponseWriter, r *http.Request) {
	var req DiscoveryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if strings.TrimSpace(req.TenantID) == "" {
		http.Error(w, "tenant_id obrigatório", 400)
		return
	}
	if err := TriggerDiscovery(req.TenantID); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

func TenantDiscoveryHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := mux.Vars(r)["tenantID"]
	if strings.TrimSpace(tenantID) == "" {
		http.Error(w, "tenant inválido", http.StatusBadRequest)
		return
	}

	var req DiscoveryRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	if len(req.IPs) > 0 || req.SNMP != nil {
		dbName, err := ResolveTenantDBName(tenantID)
		if err != nil {
			http.Error(w, "Tenant inválido", 404)
			return
		}
		if err := SeedTenantDiscovery(tenantID, dbName, req.IPs, req.SNMP); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	}

	if err := TriggerDiscovery(tenantID); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

func safe(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}

func deref(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
