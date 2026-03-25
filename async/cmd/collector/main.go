package main

import (
    "context"
    "os"
    "os/signal"
    "syscall"

    "github.com/bitbyteti/noc-guardian/async/internal/config"
    "github.com/bitbyteti/noc-guardian/async/internal/queue"
    "github.com/bitbyteti/noc-guardian/async/internal/services"
)

func main() {
    cfg := config.Load()
    log := services.InitLogger(cfg.LogLevel)

    mq, err := queue.NewRabbitMQ(cfg)
    if err != nil {
        log.Error("rabbitmq connection failed", "error", err)
        os.Exit(1)
    }
    defer mq.Close()

    collector := services.NewCollector(cfg, mq, log)

    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()

    log.Info("collector started")
    if err := collector.Run(ctx); err != nil && err != context.Canceled {
        log.Error("collector stopped with error", "error", err)
        os.Exit(1)
    }

    log.Info("collector stopped")
}
