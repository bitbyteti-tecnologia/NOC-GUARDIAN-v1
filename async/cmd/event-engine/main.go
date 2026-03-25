package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/bitbyteti/noc-guardian/async/internal/alerts"
	"github.com/bitbyteti/noc-guardian/async/internal/config"
	"github.com/bitbyteti/noc-guardian/async/internal/db"
	"github.com/bitbyteti/noc-guardian/async/internal/events"
	"github.com/bitbyteti/noc-guardian/async/internal/queue"
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

	mq, err := queue.NewRabbitMQ(cfg)
	if err != nil {
		log.Error("rabbitmq connection failed", "error", err)
		os.Exit(1)
	}
	defer mq.Close()

	eventsRepo := events.NewRepository(store.Pool())
	alertsRepo := alerts.NewRepository(store.Pool())

	engine := events.NewEngine(cfg, store, eventsRepo, alertsRepo, mq, log)

	log.Info("event engine started")
	if err := engine.Run(ctx); err != nil && err != context.Canceled {
		log.Error("event engine stopped with error", "error", err)
		os.Exit(1)
	}
	log.Info("event engine stopped")
}
