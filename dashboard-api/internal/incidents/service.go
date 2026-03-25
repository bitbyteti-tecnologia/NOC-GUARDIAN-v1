package incidents

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"sort"
	"time"
)

type TenantOpener func(ctx context.Context, tenantID string) (*sql.DB, string, error)

type Service struct {
	OpenTenant    TenantOpener
	LogPrefix     string
	EnforceTenant bool
}

func (s *Service) Details(ctx context.Context, tenantID, incidentID string) (DetailsResponse, error) {
	if tenantID == "" || incidentID == "" {
		return DetailsResponse{}, errors.New("tenant_id e incident_id são obrigatórios")
	}

	start := time.Now()
	db, _, err := s.OpenTenant(ctx, tenantID)
	if err != nil {
		return DetailsResponse{}, err
	}
	defer db.Close()

	repo := NewRepository(db)
	if err := repo.EnsureIncidentsTables(ctx); err != nil {
		return DetailsResponse{}, err
	}

	inc, err := repo.GetIncident(ctx, incidentID)
	if err != nil {
		return DetailsResponse{}, err
	}

	events, err := repo.ListEvents(ctx, incidentID)
	if err != nil {
		return DetailsResponse{}, err
	}

	timeline := buildTimeline(events)
	devices := buildDevices(events)

	schema, _ := repo.DetectMetricsSchema(ctx)
	metrics := []string{"cpu_percent", "mem_used_pct", "disk_used_pct", "wan_latency_ms", "net_rx_bps", "net_tx_bps"}
	from := time.Now().UTC().Add(-30 * time.Minute)
	to := time.Now().UTC()
	series, _ := repo.RecentMetrics(ctx, schema, inc.RootDevice, from, to, metrics)

	log.Printf("%sincident details tenant=%s incident=%s events=%d took=%s",
		s.LogPrefix, tenantID, incidentID, len(events), time.Since(start))

	return DetailsResponse{
		Incident: inc,
		Events:   events,
		Timeline: timeline,
		Devices:  devices,
		Metrics:  series,
	}, nil
}

func buildTimeline(events []Event) []TimelineItem {
	items := make([]TimelineItem, 0)
	for _, ev := range events {
		items = append(items, TimelineItem{
			Ts:        ev.FirstSeen,
			Type:      "created",
			EventID:   ev.EventID,
			EventType: ev.EventType,
			Severity:  ev.Severity,
			DeviceID:  ev.DeviceID,
		})
		if ev.LastSeen.After(ev.FirstSeen) {
			items = append(items, TimelineItem{
				Ts:        ev.LastSeen,
				Type:      "updated",
				EventID:   ev.EventID,
				EventType: ev.EventType,
				Severity:  ev.Severity,
				DeviceID:  ev.DeviceID,
			})
		}
		if ev.Status == "resolved" {
			items = append(items, TimelineItem{
				Ts:        ev.UpdatedAt,
				Type:      "resolved",
				EventID:   ev.EventID,
				EventType: ev.EventType,
				Severity:  ev.Severity,
				DeviceID:  ev.DeviceID,
			})
		}
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Ts.Before(items[j].Ts)
	})
	return items
}

func buildDevices(events []Event) []DeviceItem {
	status := map[string]string{}
	for _, ev := range events {
		if ev.DeviceID == "" {
			continue
		}
		cur := status[ev.DeviceID]
		if ev.Status == "active" {
			status[ev.DeviceID] = "active"
		} else if cur == "" {
			status[ev.DeviceID] = "resolved"
		}
	}
	out := make([]DeviceItem, 0, len(status))
	for id, st := range status {
		out = append(out, DeviceItem{DeviceID: id, Status: st})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].DeviceID < out[j].DeviceID
	})
	return out
}
