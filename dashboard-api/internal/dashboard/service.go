package dashboard

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"
)

type TenantOpener func(ctx context.Context, tenantID string) (*sql.DB, string, error)

type Service struct {
	OpenTenant    TenantOpener
	OnlineWindow  time.Duration
	LogPrefix     string
	Cache         Cache
	CacheTTL      time.Duration
	MaxPoints     int
	SchemaCache   *SchemaCache
	EnforceTenant bool
}

func (s *Service) AggregateSeries(ctx context.Context, req AggregateRequest) (AggregateResponse, error) {
	if req.TenantID == "" || req.MetricName == "" {
		return AggregateResponse{}, errors.New("tenant_id e metric_name são obrigatórios")
	}
	if req.Interval <= 0 {
		return AggregateResponse{}, errors.New("interval inválido")
	}
	if req.To.IsZero() {
		req.To = time.Now().UTC()
	}
	if req.From.IsZero() {
		req.From = req.To.Add(-1 * time.Hour)
	}
	if req.From.After(req.To) {
		return AggregateResponse{}, errors.New("from não pode ser maior que to")
	}
	if req.Mode == "" {
		req.Mode = "all"
	}
	if req.Mode != "all" && req.Mode != "online" {
		return AggregateResponse{}, errors.New("mode inválido (use all | online)")
	}
	if req.Fill != "" && req.Fill != "zero" && req.Fill != "carry" {
		return AggregateResponse{}, errors.New("fill inválido (use zero | carry)")
	}
	if req.OnlineWindow <= 0 {
		req.OnlineWindow = s.OnlineWindow
	}
	if req.OnlineWindow <= 0 {
		req.OnlineWindow = 2 * time.Minute
	}
	if req.MaxPoints <= 0 {
		req.MaxPoints = s.MaxPoints
	}
	if req.MaxPoints <= 0 {
		req.MaxPoints = 1000
	}

	expected := int(req.To.Sub(req.From).Seconds()/req.Interval.Seconds()) + 1
	if expected > req.MaxPoints {
		return AggregateResponse{}, fmt.Errorf("interval muito pequeno para o range solicitado (max %d pontos)", req.MaxPoints)
	}

	cacheKey := ""
	if s.Cache != nil {
		cacheKey = fmt.Sprintf("series:%s:%s:%s:%d:%d:%d:%s",
			req.TenantID,
			req.MetricName,
			req.Mode,
			req.From.Unix(),
			req.To.Unix(),
			int(req.Interval.Seconds()),
			req.Fill,
		)
		if cached, err := s.Cache.Get(ctx, cacheKey); err == nil && cached != "" {
			if v, err := DecodeCache(cached); err == nil {
				return v, nil
			}
		}
	}

	db, _, err := s.OpenTenant(ctx, req.TenantID)
	if err != nil {
		return AggregateResponse{}, err
	}
	defer db.Close()

	repo := NewRepository(db)
	schema := SchemaV1
	if s.SchemaCache != nil {
		if cached, ok := s.SchemaCache.Get(req.TenantID); ok {
			schema = cached
		} else {
			detected, err := repo.DetectSchema(ctx)
			if err != nil || detected == SchemaUnknown {
				log.Printf("%sschema detect failed tenant=%s err=%v (fallback v1)", s.LogPrefix, req.TenantID, err)
				schema = SchemaV1
			} else {
				schema = detected
			}
			s.SchemaCache.Set(req.TenantID, schema)
		}
	} else {
		detected, err := repo.DetectSchema(ctx)
		if err != nil || detected == SchemaUnknown {
			log.Printf("%sschema detect failed tenant=%s err=%v (fallback v1)", s.LogPrefix, req.TenantID, err)
			schema = SchemaV1
		} else {
			schema = detected
		}
	}

	start := time.Now()
	points, err := repo.QueryAggregate(ctx, req, schema)
	elapsed := time.Since(start)
	if err != nil {
		return AggregateResponse{}, err
	}

	log.Printf("%saggregate series tenant=%s metric=%s mode=%s points=%d took=%s",
		s.LogPrefix, req.TenantID, req.MetricName, req.Mode, len(points), elapsed)

	filled := points
	if req.Fill != "" {
		filled = fillBuckets(req.From, req.To, req.Interval, points, req.Fill)
	}

	resp := AggregateResponse{
		TenantID:   req.TenantID,
		MetricName: req.MetricName,
		Mode:       req.Mode,
		From:       req.From,
		To:         req.To,
		Interval:   fmt.Sprintf("%ds", int(req.Interval.Seconds())),
		Fill:       req.Fill,
		Points:     filled,
	}

	if s.Cache != nil && cacheKey != "" && s.CacheTTL > 0 {
		if raw, err := EncodeCache(resp); err == nil {
			_ = s.Cache.Set(ctx, cacheKey, raw, s.CacheTTL)
		}
	}

	return resp, nil
}

func fillBuckets(from, to time.Time, interval time.Duration, points []AggregatePoint, mode string) []AggregatePoint {
	if interval <= 0 {
		return points
	}
	lookup := make(map[int64]float64, len(points))
	for _, p := range points {
		lookup[p.Timestamp.Unix()] = p.Value
	}

	out := make([]AggregatePoint, 0)
	var last float64
	hasLast := false

	for t := from.UTC(); !t.After(to.UTC()); t = t.Add(interval) {
		ts := t.Unix()
		if v, ok := lookup[ts]; ok {
			out = append(out, AggregatePoint{Timestamp: t, Value: v})
			last = v
			hasLast = true
			continue
		}
		switch mode {
		case "zero":
			out = append(out, AggregatePoint{Timestamp: t, Value: 0})
		case "carry":
			if hasLast {
				out = append(out, AggregatePoint{Timestamp: t, Value: last})
			} else {
				out = append(out, AggregatePoint{Timestamp: t, Value: 0})
			}
		}
	}
	return out
}
