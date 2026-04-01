// devices.go
// - Implementa os endpoints de inventário de dispositivos por tenant:
//   * GET  /api/v1/{tenantID}/devices   -> ListDevicesHandler
//   * POST /api/v1/{tenantID}/devices   -> RegisterDeviceHandler
//
// Interação no sistema:
// - Usa o MASTER DB p/ resolver o DB específico do tenant (via ResolveTenantDBName).
// - Conecta ao DB do tenant (TimescaleDB habilitado) usando TenantDSN.
// - Lê/escreve na tabela "devices" do DB do tenant.
// - Este inventário é usado pelo Proxy/Agents para correlacionar métricas e pelo RCA.

package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"
)

type Device struct {
	ID       string `json:"id,omitempty"`
	Hostname string `json:"hostname"`
	IP       string `json:"ip,omitempty"`
	IPAddr   string `json:"ip_address,omitempty"`
	Type     string `json:"type"` // server, switch, router, ap, storage, etc.
	Vendor   string `json:"vendor,omitempty"`
	Model    string `json:"model,omitempty"`
	SNMPCred string `json:"snmp_credential_id,omitempty"`
}

func ListDevicesHandler(w http.ResponseWriter, r *http.Request) {
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

	rows, err := conn.Query(context.Background(),
		"SELECT id::text, hostname, COALESCE(ip_address, ip)::text, type, vendor, model, snmp_credential_id::text FROM devices ORDER BY created_at DESC")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var out []Device
	for rows.Next() {
		var d Device
		if err := rows.Scan(&d.ID, &d.Hostname, &d.IPAddr, &d.Type, &d.Vendor, &d.Model, &d.SNMPCred); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		d.IP = d.IPAddr
		out = append(out, d)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func RegisterDeviceHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := mux.Vars(r)["tenantID"]
	dbName, err := ResolveTenantDBName(tenantID)
	if err != nil {
		http.Error(w, "Tenant inválido", 404)
		return
	}

	var req Device
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	// Valida IP válido
	ipStr := req.IPAddr
	if ipStr == "" {
		ipStr = req.IP
	}
	if ip := net.ParseIP(ipStr); ip == nil {
		http.Error(w, "IP inválido", 400)
		return
	}
	if req.Hostname == "" || req.Type == "" {
		http.Error(w, "hostname e type são obrigatórios", 400)
		return
	}

	conn, err := pgx.Connect(context.Background(), TenantDSN(dbName))
	if err != nil {
		http.Error(w, "Conn tenant DB failed: "+err.Error(), 500)
		return
	}
	defer conn.Close(context.Background())

	var id string
	err = conn.QueryRow(context.Background(), `
        INSERT INTO devices (hostname, ip, ip_address, type, vendor, model, snmp_credential_id)
        VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7,'')::uuid)
        RETURNING id::text`,
		req.Hostname, ipStr, ipStr, req.Type, req.Vendor, req.Model, req.SNMPCred,
	).Scan(&id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Erro insert device: %v", err), 500)
		return
	}

	req.ID = id
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(req)
}
