package correlation

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/bitbyteti/noc-guardian/async/internal/events"
	"github.com/bitbyteti/noc-guardian/async/internal/incidents"
	"github.com/bitbyteti/noc-guardian/async/internal/observability"
)

type Engine struct {
	cfg           Config
	eventsRepo    *events.Repository
	incidentsRepo *incidents.Repository
	log           *slog.Logger
	counters      *observability.CorrelationCounters
	lastPoll      time.Time
}

type Config struct {
	PollIntervalSec int
	WindowSec       int
}

func NewEngine(cfg Config, eventsRepo *events.Repository, incidentsRepo *incidents.Repository, log *slog.Logger) *Engine {
	return &Engine{
		cfg:           cfg,
		eventsRepo:    eventsRepo,
		incidentsRepo: incidentsRepo,
		log:           log,
		counters:      observability.NewCorrelationCounters(),
		lastPoll:      time.Now().UTC().Add(-time.Duration(cfg.WindowSec) * time.Second),
	}
}

func (e *Engine) Run(ctx context.Context) error {
	poll := time.Duration(e.cfg.PollIntervalSec) * time.Second
	if poll <= 0 {
		poll = 10 * time.Second
	}
	window := time.Duration(e.cfg.WindowSec) * time.Second
	if window <= 0 {
		window = 60 * time.Second
	}

	ticker := time.NewTicker(poll)
	defer ticker.Stop()

	for {
		if err := e.cycle(ctx, window); err != nil {
			e.log.Error("correlation cycle failed", "error", err)
		}
		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (e *Engine) cycle(ctx context.Context, window time.Duration) error {
	since := e.lastPoll
	e.lastPoll = time.Now().UTC()

	eventsList, err := e.eventsRepo.ListActiveUpdatedSince(ctx, since)
	if err != nil {
		return err
	}
	for _, ev := range eventsList {
		if err := e.correlateEvent(ctx, ev, window); err != nil {
			e.log.Error("correlate event failed", "event_id", ev.EventID, "error", err)
		}
	}

	return e.resolveIncidents(ctx)
}

func (e *Engine) correlateEvent(ctx context.Context, ev events.Event, window time.Duration) error {
	inc, err := e.findIncident(ctx, ev, window)
	if err != nil {
		return err
	}

	if inc == nil {
		rootDevice, rootEvent, rootSeverity := RootFromEvent(ev)
		newInc, err := e.incidentsRepo.Create(ctx, incidents.Incident{
			IncidentID:   uuid.NewString(),
			TenantID:     ev.TenantID,
			RootDeviceID: rootDevice,
			RootEvent:    rootEvent,
			Severity:     rootSeverity,
			Title:        incidents.BuildTitle(rootEvent, rootDevice),
			Description:  incidents.BuildDescription(rootEvent, 1),
			Status:       "open",
			ImpactCount:  1,
		})
		if err != nil {
			return err
		}
		inc = newInc
		e.counters.IncCreated()
		e.log.Info("incident created", "incident_id", inc.IncidentID, "tenant_id", inc.TenantID, "root_event", inc.RootEvent)
	} else {
		e.counters.IncUpdated()
	}

	added, err := e.incidentsRepo.AddEvent(ctx, inc.IncidentID, ev.EventID, ev.TenantID)
	if err != nil {
		return err
	}
	if !added {
		return nil
	}

	recalc, err := e.incidentsRepo.Recompute(ctx, inc.IncidentID)
	if err != nil {
		return err
	}
	if recalc.IncidentID != "" {
		recalc.Title = incidents.BuildTitle(recalc.RootEvent, recalc.RootDeviceID)
		recalc.Description = incidents.BuildDescription(recalc.RootEvent, recalc.ImpactCount)
		recalc.Status = "open"
		if err := e.incidentsRepo.UpdateIncident(ctx, recalc); err != nil {
			return err
		}
		e.log.Info("incident updated",
			"incident_id", recalc.IncidentID,
			"severity", recalc.Severity,
			"impact", recalc.ImpactCount,
		)
	}

	return nil
}

func (e *Engine) findIncident(ctx context.Context, ev events.Event, window time.Duration) (*incidents.Incident, error) {
	if ev.DeviceID != "" {
		inc, err := e.incidentsRepo.GetOpenByDevice(ctx, ev.TenantID, ev.DeviceID)
		if err == nil && inc != nil {
			return inc, nil
		}
		if err != nil {
			return nil, err
		}
	}

	if ev.EventType != "" {
		return e.incidentsRepo.GetOpenByEventTypeWindow(ctx, ev.TenantID, ev.EventType, window)
	}
	return nil, nil
}

func (e *Engine) resolveIncidents(ctx context.Context) error {
	open, err := e.incidentsRepo.ListOpen(ctx)
	if err != nil {
		return err
	}
	for _, inc := range open {
		resolved, err := e.incidentsRepo.ResolveIfNoActiveEvents(ctx, inc.IncidentID)
		if err != nil {
			e.log.Error("resolve incident failed", "incident_id", inc.IncidentID, "error", err)
			continue
		}
		if resolved {
			e.counters.IncResolved()
			e.log.Info("incident resolved", "incident_id", inc.IncidentID)
		}
	}
	return nil
}
