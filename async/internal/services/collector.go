package services

import (
	"context"
	"log/slog"
	"math/rand"
	"time"

	"github.com/google/uuid"

	"github.com/bitbyteti/noc-guardian/async/internal/config"
	"github.com/bitbyteti/noc-guardian/async/internal/queue"
)

type Collector struct {
	cfg config.Config
	mq  *queue.RabbitMQ
	log *slog.Logger
}

func NewCollector(cfg config.Config, mq *queue.RabbitMQ, log *slog.Logger) *Collector {
	return &Collector{cfg: cfg, mq: mq, log: log}
}

func (c *Collector) Run(ctx context.Context) error {
	ticker := time.NewTicker(time.Duration(c.cfg.CollectorIntervalMS) * time.Millisecond)
	defer ticker.Stop()

	metrics := []string{"cpu_usage", "memory_usage", "latency"}
	rand.Seed(time.Now().UnixNano())

	for {
		select {
		case <-ticker.C:
			for _, deviceID := range c.cfg.CollectorDeviceIDs {
				for _, metricName := range metrics {
					value := randomValue(metricName)
					labels := map[string]string{"source": "collector"}
					eventID := uuid.NewString()
					m := BuildMetric(c.cfg.CollectorTenantID, deviceID, metricName, value, labels, eventID)
					if err := c.mq.PublishMetric(ctx, m, nil); err != nil {
						c.log.Error("publish failed", "error", err)
						return err
					}
					c.log.Info("metric published",
						"event_id", m.EventID,
						"tenant_id", m.TenantID,
						"device_id", m.DeviceID,
						"metric_name", m.MetricName,
						"metric_value", m.MetricValue,
					)
				}
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func randomValue(metricName string) float64 {
	switch metricName {
	case "cpu_usage":
		return 20 + rand.Float64()*70
	case "memory_usage":
		return 30 + rand.Float64()*60
	case "latency":
		return 5 + rand.Float64()*200
	default:
		return rand.Float64() * 100
	}
}
