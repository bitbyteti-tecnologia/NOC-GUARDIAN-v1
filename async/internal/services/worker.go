package services

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/bitbyteti/noc-guardian/async/internal/config"
	"github.com/bitbyteti/noc-guardian/async/internal/db"
	"github.com/bitbyteti/noc-guardian/async/internal/models"
	"github.com/bitbyteti/noc-guardian/async/internal/observability"
	"github.com/bitbyteti/noc-guardian/async/internal/queue"
)

type Worker struct {
	cfg      config.Config
	store    *db.Store
	mq       *queue.RabbitMQ
	log      *slog.Logger
	counters *observability.Counters
}

func NewWorker(cfg config.Config, store *db.Store, mq *queue.RabbitMQ, log *slog.Logger) *Worker {
	return &Worker{
		cfg:      cfg,
		store:    store,
		mq:       mq,
		log:      log,
		counters: observability.NewCounters(),
	}
}

func (w *Worker) Run(ctx context.Context) error {
	prefetch := w.cfg.RabbitPrefetchPerWorker * w.cfg.WorkerConcurrency
	if prefetch <= 0 {
		prefetch = 20
	}
	deliveries, err := w.mq.Consume(prefetch)
	if err != nil {
		return err
	}

	jobs := make(chan amqp.Delivery)

	for i := 0; i < w.cfg.WorkerConcurrency; i++ {
		go w.workerLoop(ctx, jobs, i)
	}

	go w.logStats(ctx)

	for {
		select {
		case d, ok := <-deliveries:
			if !ok {
				return errors.New("delivery channel closed")
			}
			jobs <- d
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (w *Worker) workerLoop(ctx context.Context, jobs <-chan amqp.Delivery, id int) {
	buffer := make([]amqp.Delivery, 0, w.cfg.WorkerBatchSize)
	metrics := make([]models.Metric, 0, w.cfg.WorkerBatchSize)
	flushTicker := time.NewTicker(time.Duration(w.cfg.WorkerBatchFlushMS) * time.Millisecond)
	defer flushTicker.Stop()

	flush := func() {
		if len(buffer) == 0 {
			return
		}
		start := time.Now()
		if err := w.store.InsertMetricsBatch(ctx, metrics); err != nil {
			w.handleBatchFailure(ctx, buffer, err)
			buffer = buffer[:0]
			metrics = metrics[:0]
			return
		}

		for _, d := range buffer {
			if err := d.Ack(false); err != nil {
				w.log.Error("ack failed", "worker", id, "error", err)
			}
			w.counters.IncProcessed()
		}
		w.counters.ObserveDuration(time.Since(start))
		buffer = buffer[:0]
		metrics = metrics[:0]
	}

	for {
		select {
		case d, ok := <-jobs:
			if !ok {
				flush()
				return
			}
			m, err := w.decodeAndValidate(d)
			if err != nil {
				w.counters.IncFailed()
				_ = d.Ack(false)
				continue
			}
			w.log.Debug("message consumed",
				"event_id", m.EventID,
				"tenant_id", m.TenantID,
				"device_id", m.DeviceID,
				"metric_name", m.MetricName,
			)
			buffer = append(buffer, d)
			metrics = append(metrics, m)
			if len(buffer) >= w.cfg.WorkerBatchSize {
				flush()
			}
		case <-flushTicker.C:
			flush()
		case <-ctx.Done():
			flush()
			return
		}
	}
}

func (w *Worker) decodeAndValidate(d amqp.Delivery) (models.Metric, error) {
	var m models.Metric
	if err := json.Unmarshal(d.Body, &m); err != nil {
		w.log.Error("invalid payload", "error", err)
		return models.Metric{}, err
	}

	if err := validateMetric(m); err != nil {
		w.log.Error("metric validation failed", "error", err)
		return models.Metric{}, err
	}

	m.MetricName = strings.ToLower(strings.TrimSpace(m.MetricName))
	m.Timestamp = m.Timestamp.UTC()

	return m, nil
}

func (w *Worker) handleBatchFailure(ctx context.Context, deliveries []amqp.Delivery, cause error) {
	w.log.Error("batch insert failed", "error", cause, "count", len(deliveries))
	for _, d := range deliveries {
		if err := w.handleFailure(ctx, d, cause); err != nil {
			w.log.Error("message failure handling failed", "error", err)
		}
	}
}

func (w *Worker) handleFailure(ctx context.Context, d amqp.Delivery, cause error) error {
	retries := queue.RetryCount(d.Headers)
	if retries < w.cfg.MaxRetries {
		headers := queue.WithRetryCount(d.Headers, retries+1)
		if err := w.mq.PublishRetry(ctx, d.Body, headers); err != nil {
			w.log.Error("failed to publish retry", "error", err)
			_ = d.Nack(false, true)
			return err
		}
		_ = d.Ack(false)
		w.counters.IncRetried()
		w.log.Warn("message sent to retry", "retry", retries+1, "error", cause)
		return nil
	}

	headers := queue.WithRetryCount(d.Headers, retries)
	if err := w.mq.PublishDead(ctx, d.Body, headers); err != nil {
		w.log.Error("failed to publish dead-letter", "error", err)
		_ = d.Nack(false, true)
		return err
	}
	_ = d.Ack(false)
	w.counters.IncDead()
	w.log.Error("message sent to dead-letter", "error", cause)
	return nil
}

func (w *Worker) logStats(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(w.cfg.StatsIntervalSec) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			snap := w.counters.Snapshot()
			w.log.Info("worker stats",
				"processed", snap.Processed,
				"failed", snap.Failed,
				"retried", snap.Retried,
				"dead", snap.Dead,
				"avg_ms", snap.AvgMS,
			)
		case <-ctx.Done():
			return
		}
	}
}

func validateMetric(m models.Metric) error {
	if m.EventID == "" {
		return errors.New("event_id is required")
	}
	if m.TenantID == "" || m.DeviceID == "" || m.MetricName == "" {
		return errors.New("tenant_id, device_id, and metric_name are required")
	}
	if m.Timestamp.IsZero() {
		return errors.New("timestamp is required")
	}
	return nil
}

func BuildMetric(tenantID, deviceID, metricName string, value float64, labels map[string]string, eventID string) models.Metric {
	return models.Metric{
		EventID:     eventID,
		TenantID:    tenantID,
		DeviceID:    deviceID,
		MetricName:  metricName,
		MetricValue: value,
		Labels:      labels,
		Timestamp:   time.Now().UTC(),
	}
}
