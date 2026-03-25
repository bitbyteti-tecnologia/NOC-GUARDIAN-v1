package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/bitbyteti/noc-guardian/async/internal/config"
	"github.com/bitbyteti/noc-guardian/async/internal/correlation"
	"github.com/bitbyteti/noc-guardian/async/internal/db"
	"github.com/bitbyteti/noc-guardian/async/internal/events"
	"github.com/bitbyteti/noc-guardian/async/internal/incidents"
	"github.com/bitbyteti/noc-guardian/async/internal/services"
)

func main() {
	cfg := config.Load()
	log := services.InitLogger(cfg.LogLevel)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store, err := db.NewStore(ctx, cfg.DBDSN)
	if err != nil {
		log.Error("db connection failed", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	eventsRepo := events.NewRepository(store.Pool())
	incidentsRepo := incidents.NewRepository(store.Pool())

	engine := correlation.NewEngine(
		correlation.Config{
			PollIntervalSec: cfg.CorrelationPollSec,
			WindowSec:       cfg.CorrelationWindowSec,
		},
		eventsRepo,
		incidentsRepo,
		log,
	)

	log.Info("correlation engine started")
	if err := engine.Run(ctx); err != nil && err != context.Canceled {
		log.Error("correlation engine stopped with error", "error", err)
		os.Exit(1)
	}
	log.Info("correlation engine stopped")
}
