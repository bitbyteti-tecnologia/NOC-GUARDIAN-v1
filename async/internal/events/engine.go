package events

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/bitbyteti/noc-guardian/async/internal/alerts"
	"github.com/bitbyteti/noc-guardian/async/internal/config"
	"github.com/bitbyteti/noc-guardian/async/internal/db"
	"github.com/bitbyteti/noc-guardian/async/internal/models"
	"github.com/bitbyteti/noc-guardian/async/internal/observability"
	"github.com/bitbyteti/noc-guardian/async/internal/queue"
)

type Engine struct {
	cfg      config.Config
	store    *db.Store
	events   *Repository
	alerts   *alerts.Repository
	mq       *queue.RabbitMQ
	log      *slog.Logger
	counters *observability.EventCounters
	rules    []Rule
}

func NewEngine(cfg config.Config, store *db.Store, eventsRepo *Repository, alertsRepo *alerts.Repository, mq *queue.RabbitMQ, log *slog.Logger) *Engine {
	rules := []Rule{
		{MetricName: "cpu_usage", Threshold: cfg.CPUThreshold, Severity: "critical", EventType: "cpu_high", Message: "CPU usage above threshold"},
		{MetricName: "memory_usage", Threshold: cfg.MemoryThreshold, Severity: "warning", EventType: "memory_high", Message: "Memory usage above threshold"},
	}
	return &Engine{
		cfg:      cfg,
		store:    store,
		events:   eventsRepo,
		alerts:   alertsRepo,
		mq:       mq,
		log:      log,
		counters: observability.NewEventCounters(),
		rules:    rules,
	}
}

func (e *Engine) Run(ctx context.Context) error {
	prefetch := e.cfg.RabbitPrefetchPerWorker
	if prefetch <= 0 {
		prefetch = 20
	}
	deliveries, err := e.mq.Consume(prefetch)
	if err != nil {
		return err
	}

	go e.offlineChecker(ctx)
	go e.logStats(ctx)

	for {
		select {
		case d, ok := <-deliveries:
			if !ok {
				return errors.New("delivery channel closed")
			}
			if err := e.processDelivery(ctx, d); err != nil {
				e.log.Error("event engine processing failed", "error", err)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (e *Engine) processDelivery(ctx context.Context, d amqp.Delivery) error {
	var m models.Metric
	if err := json.Unmarshal(d.Body, &m); err != nil {
		e.log.Error("invalid metric payload", "error", err)
		_ = d.Ack(false)
		return err
	}

	if m.TenantID == "" || m.DeviceID == "" || m.MetricName == "" {
		_ = d.Ack(false)
		return errors.New("metric missing required fields")
	}

	if err := e.applyRules(ctx, m); err != nil {
		return e.handleFailure(ctx, d, err)
	}

	if err := e.resolveOfflineIfRecovered(ctx, m); err != nil {
		e.log.Error("offline resolve failed", "error", err)
	}

	if err := d.Ack(false); err != nil {
		return err
	}
	return nil
}

func (e *Engine) applyRules(ctx context.Context, m models.Metric) error {
	for _, rule := range e.rules {
		if !rule.Matches(m) {
			continue
		}

		if rule.Triggered(m.MetricValue) {
			event := &Event{
				EventID:   uuid.NewString(),
				TenantID:  m.TenantID,
				DeviceID:  m.DeviceID,
				EventType: rule.EventType,
				Severity:  rule.Severity,
				Message:   rule.Message,
				Metadata: map[string]any{
					"metric_name":  m.MetricName,
					"metric_value": m.MetricValue,
					"threshold":    rule.Threshold,
					"labels":       m.Labels,
				},
			}

			created, err := e.events.UpsertActive(ctx, event)
			if err != nil {
				return err
			}
			if created {
				e.counters.IncCreated()
				e.log.Info("event created",
					"tenant_id", event.TenantID,
					"device_id", event.DeviceID,
					"event_type", event.EventType,
					"severity", event.Severity,
				)
				if event.Severity == "critical" {
					if err := e.alerts.CreateIfNotExists(ctx, alerts.Alert{
						TenantID:  event.TenantID,
						EventID:   event.EventID,
						AlertType: event.EventType,
						Severity:  event.Severity,
						Message:   event.Message,
					}); err != nil {
						return err
					}
					e.counters.IncAlerts()
					e.log.Info("alert generated",
						"tenant_id", event.TenantID,
						"event_id", event.EventID,
						"event_type", event.EventType,
					)
				}
			} else {
				e.counters.IncUpdated()
				e.log.Info("event updated",
					"tenant_id", event.TenantID,
					"device_id", event.DeviceID,
					"event_type", event.EventType,
				)
			}
		} else {
			resolved, err := e.events.ResolveActive(ctx, m.TenantID, m.DeviceID, rule.EventType)
			if err != nil {
				return err
			}
			if resolved {
				e.counters.IncResolved()
				e.log.Info("event resolved",
					"tenant_id", m.TenantID,
					"device_id", m.DeviceID,
					"event_type", rule.EventType,
				)
			}
		}
	}
	return nil
}

func (e *Engine) resolveOfflineIfRecovered(ctx context.Context, m models.Metric) error {
	resolved, err := e.events.ResolveActive(ctx, m.TenantID, m.DeviceID, "device_offline")
	if err != nil {
		return err
	}
	if resolved {
		e.counters.IncResolved()
		e.log.Info("device recovered",
			"tenant_id", m.TenantID,
			"device_id", m.DeviceID,
		)
	}
	return nil
}

func (e *Engine) offlineChecker(ctx context.Context) {
	interval := time.Duration(e.cfg.OfflineCheckSec) * time.Second
	threshold := time.Duration(e.cfg.OfflineThresholdSec) * time.Second
	if interval <= 0 || threshold <= 0 {
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			stale, err := e.store.ListStaleDevices(ctx, threshold)
			if err != nil {
				e.log.Error("offline scan failed", "error", err)
				continue
			}
			for _, item := range stale {
				event := &Event{
					EventID:   uuid.NewString(),
					TenantID:  item.TenantID,
					DeviceID:  item.DeviceID,
					EventType: "device_offline",
					Severity:  "critical",
					Message:   "Device offline (no metrics)",
					Metadata: map[string]any{
						"last_seen": item.LastSeen,
					},
				}

				created, err := e.events.UpsertActive(ctx, event)
				if err != nil {
					e.log.Error("offline event upsert failed", "error", err)
					continue
				}
				if created {
					e.counters.IncCreated()
					e.log.Info("offline event created",
						"tenant_id", event.TenantID,
						"device_id", event.DeviceID,
					)
					if err := e.alerts.CreateIfNotExists(ctx, alerts.Alert{
						TenantID:  event.TenantID,
						EventID:   event.EventID,
						AlertType: event.EventType,
						Severity:  event.Severity,
						Message:   event.Message,
					}); err != nil {
						e.log.Error("offline alert create failed", "error", err)
						continue
					}
					e.counters.IncAlerts()
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func (e *Engine) handleFailure(ctx context.Context, d amqp.Delivery, cause error) error {
	retries := queue.RetryCount(d.Headers)
	if retries < e.cfg.MaxRetries {
		headers := queue.WithRetryCount(d.Headers, retries+1)
		if err := e.mq.PublishRetry(ctx, d.Body, headers); err != nil {
			e.log.Error("failed to publish retry", "error", err)
			_ = d.Nack(false, true)
			return err
		}
		_ = d.Ack(false)
		e.log.Warn("event engine message sent to retry", "retry", retries+1, "error", cause)
		return nil
	}

	headers := queue.WithRetryCount(d.Headers, retries)
	if err := e.mq.PublishDead(ctx, d.Body, headers); err != nil {
		e.log.Error("failed to publish dead-letter", "error", err)
		_ = d.Nack(false, true)
		return err
	}
	_ = d.Ack(false)
	e.log.Error("event engine message sent to dead-letter", "error", cause)
	return nil
}

func (e *Engine) logStats(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(e.cfg.StatsIntervalSec) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			snap := e.counters.Snapshot()
			e.log.Info("event engine stats",
				"events_created", snap.Created,
				"events_updated", snap.Updated,
				"events_resolved", snap.Resolved,
				"alerts_created", snap.Alerts,
			)
		case <-ctx.Done():
			return
		}
	}
}
