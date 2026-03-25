package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	RabbitURL               string
	RabbitExchange          string
	RabbitQueue             string
	RabbitRoutingKey        string
	RabbitRetryExchange     string
	RabbitRetryQueue        string
	RabbitRetryRoutingKey   string
	RabbitDeadExchange      string
	RabbitDeadQueue         string
	RabbitDeadRoutingKey    string
	RabbitRetryTTLMS        int
	RabbitPrefetchPerWorker int
	MaxRetries              int

	DBDSN string

	APIAddr string

	CollectorTenantID   string
	CollectorDeviceIDs  []string
	CollectorIntervalMS int
	LogLevel            string
	WorkerConcurrency   int
	WorkerBatchSize     int
	WorkerBatchFlushMS  int
	StatsIntervalSec    int

	CPUThreshold        float64
	MemoryThreshold     float64
	OfflineThresholdSec int
	OfflineCheckSec     int

	CorrelationPollSec   int
	CorrelationWindowSec int
}

func Load() Config {
	return Config{
		RabbitURL:               env("RABBIT_URL", "amqp://guest:guest@localhost:5672/"),
		RabbitExchange:          env("RABBIT_EXCHANGE", "metrics.exchange"),
		RabbitQueue:             env("RABBIT_QUEUE", "metrics.queue"),
		RabbitRoutingKey:        env("RABBIT_ROUTING_KEY", "metrics"),
		RabbitRetryExchange:     env("RABBIT_RETRY_EXCHANGE", "metrics.retry.exchange"),
		RabbitRetryQueue:        env("RABBIT_RETRY_QUEUE", "metrics.retry.queue"),
		RabbitRetryRoutingKey:   env("RABBIT_RETRY_ROUTING_KEY", "metrics.retry"),
		RabbitDeadExchange:      env("RABBIT_DEAD_EXCHANGE", "metrics.dead.exchange"),
		RabbitDeadQueue:         env("RABBIT_DEAD_QUEUE", "metrics.dead.queue"),
		RabbitDeadRoutingKey:    env("RABBIT_DEAD_ROUTING_KEY", "metrics.dead"),
		RabbitRetryTTLMS:        envInt("RABBIT_RETRY_TTL_MS", 10000),
		RabbitPrefetchPerWorker: envInt("RABBIT_PREFETCH_PER_WORKER", 20),
		MaxRetries:              envInt("MAX_RETRIES", 5),

		DBDSN: env("DB_DSN", "postgres://noc:noc@localhost:5432/nocguardian?sslmode=disable"),

		APIAddr: env("API_ADDR", ":8080"),

		CollectorTenantID:   env("COLLECTOR_TENANT_ID", "tenant_demo"),
		CollectorDeviceIDs:  envCSV("COLLECTOR_DEVICE_IDS", "dev-001,dev-002,dev-003"),
		CollectorIntervalMS: envInt("COLLECTOR_INTERVAL_MS", 2000),
		LogLevel:            env("LOG_LEVEL", "info"),
		WorkerConcurrency:   envInt("WORKER_CONCURRENCY", 8),
		WorkerBatchSize:     envInt("WORKER_BATCH_SIZE", 100),
		WorkerBatchFlushMS:  envInt("WORKER_BATCH_FLUSH_MS", 1000),
		StatsIntervalSec:    envInt("STATS_INTERVAL_SEC", 10),

		CPUThreshold:        envFloat("CPU_THRESHOLD", 90),
		MemoryThreshold:     envFloat("MEMORY_THRESHOLD", 85),
		OfflineThresholdSec: envInt("OFFLINE_THRESHOLD_SEC", 120),
		OfflineCheckSec:     envInt("OFFLINE_CHECK_SEC", 30),

		CorrelationPollSec:   envInt("CORRELATION_POLL_SEC", 10),
		CorrelationWindowSec: envInt("CORRELATION_WINDOW_SEC", 60),
	}
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}

func envCSV(key, def string) []string {
	raw := env(key, def)
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func envFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}
