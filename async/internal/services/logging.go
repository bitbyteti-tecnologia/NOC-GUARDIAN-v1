package services

import (
    "log/slog"
    "os"
    "strings"
)

func InitLogger(level string) *slog.Logger {
    lvl := slog.LevelInfo
    switch strings.ToLower(level) {
    case "debug":
        lvl = slog.LevelDebug
    case "warn":
        lvl = slog.LevelWarn
    case "error":
        lvl = slog.LevelError
    }

    handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
    logger := slog.New(handler)
    slog.SetDefault(logger)
    return logger
}
