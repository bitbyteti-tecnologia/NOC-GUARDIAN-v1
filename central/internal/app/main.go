package app

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func Run() {
	_ = godotenv.Load()
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	if err := InitMasterDB(); err != nil {
		log.Fatalf("Erro init master DB: %v", err)
	}
	if err := RunMasterMigrations(); err != nil {
		log.Fatalf("Erro migrações master: %v", err)
	}
	if err := SeedSuperAdmin(); err != nil {
		log.Fatalf("Erro seed superadmin: %v", err)
	}

	r := mux.NewRouter()

	// Health
	r.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}).Methods("GET")

	// Auth
	r.HandleFunc("/api/v1/auth/login", LoginHandler).Methods("POST")
	r.Handle("/api/v1/auth/me", RequireAuth(http.HandlerFunc(MeHandler))).Methods("GET")
	r.HandleFunc("/api/v1/auth/refresh", RefreshHandler).Methods("POST")
	r.HandleFunc("/api/v1/auth/logout", LogoutHandler).Methods("POST")
	r.Handle("/api/v1/auth/change-password", RequireAuth(http.HandlerFunc(ChangePasswordHandler))).Methods("POST")
	r.HandleFunc("/api/v1/auth/forgot-password", ForgotPasswordHandler).Methods("POST")
	r.HandleFunc("/api/v1/auth/reset-password", ResetPasswordHandler).Methods("POST")

	// Sessões (refresh tokens)
	r.Handle("/api/v1/auth/sessions", RequireAuth(http.HandlerFunc(ListSessionsHandler))).Methods("GET")
	r.Handle("/api/v1/auth/sessions/revoke-all", RequireAuth(http.HandlerFunc(RevokeAllSessionsHandler))).Methods("POST")

	// Tenants
	r.Handle("/api/v1/tenants", RequireAuth(RequireRole("superadmin", "support")(http.HandlerFunc(CreateTenantHandler)))).Methods("POST")
	r.Handle("/api/v1/tenants", RequireAuth(RequireRole("superadmin", "support")(http.HandlerFunc(ListTenantsHandler)))).Methods("GET")
	r.Handle("/api/v1/tenants/{tenantID}", RequireAuth(RequireRole("superadmin", "support")(http.HandlerFunc(GetTenantInfoHandler)))).Methods("GET")
	r.Handle("/api/v1/tenants/{tenantID}/activation-token", RequireAuth(RequireRole("superadmin", "support", "admin")(http.HandlerFunc(ActivationTokenGetHandler)))).Methods("GET")
	r.Handle("/api/v1/tenants/{tenantID}/activation-token", RequireAuth(RequireRole("superadmin", "support", "admin")(http.HandlerFunc(ActivationTokenPostHandler)))).Methods("POST")
	r.Handle("/api/v1/tenants/{tenantID}/discovery", RequireAuth(RequireRole("superadmin", "support", "admin")(http.HandlerFunc(TenantDiscoveryHandler)))).Methods("POST")
	r.Handle("/api/v1/discovery/run", RequireAuth(RequireRole("superadmin", "support", "admin")(http.HandlerFunc(DiscoveryRunHandler)))).Methods("POST")

	// Users (globais e por tenant)
	r.Handle("/api/v1/users", RequireAuth(RequireRole("superadmin", "support")(http.HandlerFunc(CreateGlobalUserHandler)))).Methods("POST")
	r.Handle("/api/v1/users", RequireAuth(RequireRole("superadmin", "support")(http.HandlerFunc(ListGlobalUsersHandler)))).Methods("GET")
	r.Handle("/api/v1/{tenantID}/users", RequireAuth(RequireRole("superadmin", "support", "admin")(http.HandlerFunc(CreateTenantUserHandler)))).Methods("POST")
	r.Handle("/api/v1/{tenantID}/users", RequireAuth(RequireRole("superadmin", "support", "admin")(http.HandlerFunc(ListTenantUsersHandler)))).Methods("GET")

	// Devices
	r.Handle("/api/v1/{tenantID}/devices", RequireAuth(RequireRole("superadmin", "support", "admin")(http.HandlerFunc(ListDevicesHandler)))).Methods("GET")
	r.Handle("/api/v1/{tenantID}/devices", RequireAuth(RequireRole("superadmin", "support", "admin")(http.HandlerFunc(RegisterDeviceHandler)))).Methods("POST")

	// Métricas
	r.Handle("/api/v1/{tenantID}/metrics/ingest", http.HandlerFunc(MetricsIngestHandler)).Methods("POST")
	r.Handle("/api/v1/{tenantID}/agents", http.HandlerFunc(AgentsListHandler)).Methods("GET")
	r.Handle("/api/v1/{tenantID}/metrics/latest", http.HandlerFunc(MetricsLatestHandler)).Methods("GET")
	r.Handle("/api/v1/{tenantID}/metrics/range", http.HandlerFunc(MetricsRangeHandler)).Methods("GET")

	// Alertas / RCA / Diagnósticos
	r.Handle("/api/v1/{tenantID}/alerts", RequireAuth(RequireRole("superadmin", "support", "admin")(http.HandlerFunc(ListAlertsHandler)))).Methods("GET")
	r.Handle("/api/v1/{tenantID}/rca/run", RequireAuth(RequireRole("superadmin", "support", "admin")(http.HandlerFunc(RCAHandler)))).Methods("POST")
	r.Handle("/api/v1/{tenantID}/diagnostics/ping", RequireAuth(RequireRole("superadmin", "support", "admin")(http.HandlerFunc(DiagnosticPingHandler)))).Methods("POST")

	log.Printf("CENTRAL API rodando na porta %s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}
