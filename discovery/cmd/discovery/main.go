package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"discovery/internal/config"
	"discovery/internal/discovery"
)

func main() {
	cfg := config.MustLoad()

	if cfg.SNMPCommunity == "" {
		log.Println("[discovery] WARN: SNMP_COMMUNITY vazio; discovery não funcionará.")
	}

	svc := &discovery.Service{
		MasterHost:    cfg.MasterHost,
		MasterPort:    cfg.MasterPort,
		MasterUser:    cfg.MasterUser,
		MasterPass:    cfg.MasterPass,
		MasterDB:      cfg.MasterDB,
		LogPrefix:     "[discovery] ",
		SNMPCommunity: cfg.SNMPCommunity,
		SNMPVersion:   cfg.SNMPVersion,
		SNMPPort:      cfg.SNMPPort,
		SNMPTimeout:   cfg.SNMPTimeout,
		SNMPRetries:   cfg.SNMPRetries,
		SNMPCredKey:   cfg.SNMPCredKey,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sig
		cancel()
	}()

	listen := strings.TrimSpace(os.Getenv("DISCOVERY_LISTEN"))
	if listen == "" {
		listen = ":8085"
	}

	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		})
		mux.HandleFunc("/discovery/run", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			var req struct {
				TenantID string `json:"tenant_id"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			if req.TenantID == "" {
				req.TenantID = r.URL.Query().Get("tenant_id")
			}
			if strings.TrimSpace(req.TenantID) == "" {
				http.Error(w, "tenant_id obrigatório", http.StatusBadRequest)
				return
			}
			if err := svc.RunTenant(r.Context(), req.TenantID); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		})

		log.Printf("[discovery] http listen %s", listen)
		if err := http.ListenAndServe(listen, mux); err != nil {
			log.Printf("[discovery] http error: %v", err)
		}
	}()

	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	svc.RunOnce(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("[discovery] shutdown")
			return
		case <-ticker.C:
			svc.RunOnce(ctx)
		}
	}
}
