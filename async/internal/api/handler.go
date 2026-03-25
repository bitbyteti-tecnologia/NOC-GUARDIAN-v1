package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/bitbyteti/noc-guardian/async/internal/alerts"
	"github.com/bitbyteti/noc-guardian/async/internal/db"
	"github.com/bitbyteti/noc-guardian/async/internal/events"
	"github.com/bitbyteti/noc-guardian/async/internal/incidents"
)

type Server struct {
	store      *db.Store
	log        *slog.Logger
	eventsRepo *events.Repository
	alertsRepo *alerts.Repository
	incRepo    *incidents.Repository
}

func NewServer(store *db.Store, log *slog.Logger) *Server {
	return &Server{
		store:      store,
		log:        log,
		eventsRepo: events.NewRepository(store.Pool()),
		alertsRepo: alerts.NewRepository(store.Pool()),
		incRepo:    incidents.NewRepository(store.Pool()),
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", s.handleMetrics)
	mux.HandleFunc("/events", s.handleEvents)
	mux.HandleFunc("/alerts", s.handleAlerts)
	mux.HandleFunc("/alerts/", s.handleAlertAck)
	mux.HandleFunc("/incidents", s.handleIncidents)
	mux.HandleFunc("/incidents/", s.handleIncidentDetail)
	return mux
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		http.Error(w, "tenant_id is required", http.StatusBadRequest)
		return
	}

	var deviceID *string
	if v := r.URL.Query().Get("device_id"); v != "" {
		deviceID = &v
	}

	metrics, err := s.store.QueryMetrics(r.Context(), tenantID, deviceID)
	if err != nil {
		s.log.Error("query metrics failed", "error", err)
		http.Error(w, "failed to query metrics", http.StatusInternalServerError)
		return
	}

	writeJSON(w, metrics, s.log)
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		http.Error(w, "tenant_id is required", http.StatusBadRequest)
		return
	}

	filter := events.ListFilter{
		TenantID: tenantID,
		Status:   r.URL.Query().Get("status"),
		Severity: r.URL.Query().Get("severity"),
	}

	list, err := s.eventsRepo.List(r.Context(), filter)
	if err != nil {
		s.log.Error("query events failed", "error", err)
		http.Error(w, "failed to query events", http.StatusInternalServerError)
		return
	}

	writeJSON(w, list, s.log)
}

func (s *Server) handleAlerts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		http.Error(w, "tenant_id is required", http.StatusBadRequest)
		return
	}

	var acknowledged *bool
	if v := r.URL.Query().Get("acknowledged"); v != "" {
		parsed := v == "true"
		acknowledged = &parsed
	}

	list, err := s.alertsRepo.List(r.Context(), tenantID, acknowledged)
	if err != nil {
		s.log.Error("query alerts failed", "error", err)
		http.Error(w, "failed to query alerts", http.StatusInternalServerError)
		return
	}

	writeJSON(w, list, s.log)
}

func (s *Server) handleAlertAck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/alerts/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[1] != "ack" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "invalid alert id", http.StatusBadRequest)
		return
	}

	if err := s.alertsRepo.Ack(r.Context(), id); err != nil {
		s.log.Error("ack alert failed", "error", err)
		http.Error(w, "failed to ack alert", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleIncidents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		http.Error(w, "tenant_id is required", http.StatusBadRequest)
		return
	}

	status := r.URL.Query().Get("status")
	list, err := s.incRepo.List(r.Context(), tenantID, status)
	if err != nil {
		s.log.Error("query incidents failed", "error", err)
		http.Error(w, "failed to query incidents", http.StatusInternalServerError)
		return
	}
	writeJSON(w, list, s.log)
}

func (s *Server) handleIncidentDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/incidents/")
	if path == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	inc, eventsList, err := s.incRepo.GetDetail(r.Context(), path)
	if err != nil {
		s.log.Error("incident detail failed", "error", err)
		http.Error(w, "failed to get incident", http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]any{
		"incident": inc,
		"events":   eventsList,
	}, s.log)
}

func writeJSON(w http.ResponseWriter, payload any, log *slog.Logger) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Error("encode response failed", "error", err)
	}
}
