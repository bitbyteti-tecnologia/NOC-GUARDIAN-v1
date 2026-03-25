package main

import (
    "context"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/bitbyteti/noc-guardian/async/internal/api"
    "github.com/bitbyteti/noc-guardian/async/internal/config"
    "github.com/bitbyteti/noc-guardian/async/internal/db"
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

    srv := api.NewServer(store, log)

    httpServer := &http.Server{
        Addr:         cfg.APIAddr,
        Handler:      srv.Routes(),
        ReadTimeout:  5 * time.Second,
        WriteTimeout: 10 * time.Second,
        IdleTimeout:  60 * time.Second,
    }

    go func() {
        <-ctx.Done()
        shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()
        _ = httpServer.Shutdown(shutdownCtx)
    }()

    log.Info("api started", "addr", cfg.APIAddr)
    if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        log.Error("api stopped with error", "error", err)
        os.Exit(1)
    }

    log.Info("api stopped")
}
