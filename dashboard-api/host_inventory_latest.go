package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/lib/pq"
)

type HostInventoryLatestResponse struct {
	Hostname    string          `json:"hostname"`
	AgentID     *string         `json:"agent_id,omitempty"`
	OS          *string         `json:"os,omitempty"`
	CollectedAt time.Time       `json:"collected_at"`
	Data        json.RawMessage `json:"data"`
}

func HostInventoryLatestHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	hostname := chi.URLParam(r, "hostname")
	if tenantID == "" || hostname == "" {
		http.Error(w, "tenantId/hostname missing", http.StatusBadRequest)
		return
	}

	masterDBName := invFirstEnv("MASTER_DB_NAME", "DB_NAME", "POSTGRES_DB")
	if masterDBName == "" {
		masterDBName = "guardian_master"
	}
	masterDB, err := invOpenDBWithName(masterDBName)
	if err != nil {
		log.Printf("[INV-GET] master db error: %v", err)
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	defer masterDB.Close()

	dbName, err := invTenantDBName(masterDB, tenantID)
	if err != nil {
		log.Printf("[INV-GET] tenant lookup error tenant=%s: %v", tenantID, err)
		http.Error(w, "tenant not found", http.StatusNotFound)
		return
	}

	tenantDB, err := invOpenDBWithName(dbName)
	if err != nil {
		log.Printf("[INV-GET] open tenant db error tenant=%s db=%s: %v", tenantID, dbName, err)
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	defer tenantDB.Close()

	var resp HostInventoryLatestResponse
	var agentID sql.NullString
	var osName sql.NullString
	var data []byte

	err = tenantDB.QueryRow(`
		SELECT hostname, agent_id, os, collected_at, data
		FROM public.host_inventory_latest
		WHERE hostname = $1
		   OR hostname = (SELECT hostname_inventory FROM public.host_aliases WHERE hostname_metrics = $1)
	`, hostname).Scan(&resp.Hostname, &agentID, &osName, &resp.CollectedAt, &data)

	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if isUndefinedTable(err) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("[INV-GET] query error tenant=%s db=%s host=%s: %v", tenantID, dbName, hostname, err)
		http.Error(w, "db query error", http.StatusInternalServerError)
		return
	}

	if agentID.Valid {
		resp.AgentID = &agentID.String
	}
	if osName.Valid {
		resp.OS = &osName.String
	}
	resp.Data = json.RawMessage(data)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// ---------- DB helpers (isolados para não depender do resto do dashboard-api) ----------

func invTenantDBName(master *sql.DB, tenantID string) (string, error) {
	var dbName sql.NullString
	// Não fixa schema (evita erro se search_path/schema diferirem do esperado)
	err := master.QueryRow(`SELECT db_name FROM tenants WHERE id = $1`, tenantID).Scan(&dbName)
	if err != nil {
		return "", err
	}
	if !dbName.Valid || dbName.String == "" {
		return "", errors.New("db_name empty")
	}
	return dbName.String, nil
}

func invOpenDBWithName(dbName string) (*sql.DB, error) {
	host := invFirstEnv("PGHOST", "POSTGRES_HOST", "DB_HOST")
	port := invFirstEnv("PGPORT", "POSTGRES_PORT", "DB_PORT")
	user := invFirstEnv("PGUSER", "POSTGRES_USER", "DB_USER")
	pass := invFirstEnv("PGPASSWORD", "POSTGRES_PASSWORD", "DB_PASSWORD", "MASTER_DB_PASS")

	if host == "" {
		host = "db"
	}
	if port == "" {
		port = "5432"
	}
	if user == "" {
		user = "guardian"
	}

	sslmode := invFirstEnv("PGSSLMODE", "POSTGRES_SSLMODE")
	if sslmode == "" {
		sslmode = "disable"
	}

	dsn := "host=" + host + " port=" + port + " user=" + user + " dbname=" + dbName + " sslmode=" + sslmode
	if pass != "" {
		dsn += " password=" + pass
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(8)
	db.SetMaxIdleConns(8)
	db.SetConnMaxLifetime(30 * time.Minute)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func invFirstEnv(keys ...string) string {
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return ""
}

func isUndefinedTable(err error) bool {
	if err == nil {
		return false
	}
	if pgErr, ok := err.(*pq.Error); ok {
		return pgErr.Code == "42P01"
	}
	return false
}
