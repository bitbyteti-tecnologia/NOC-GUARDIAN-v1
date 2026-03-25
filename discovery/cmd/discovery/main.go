package main

import (
	"context"
	"log"
	"os"
	"os/signal"
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
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sig
		cancel()
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
