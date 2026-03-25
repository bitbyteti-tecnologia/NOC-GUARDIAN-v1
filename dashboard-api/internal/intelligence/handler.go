package intelligence

import (
	"net/http"
)

type WriteJSON func(w http.ResponseWriter, v any, err error)

type Router interface {
	Get(pattern string, handlerFn http.HandlerFunc)
}

func RegisterRoutes(r Router, svc *Service, writeJSON WriteJSON) {
	r.Get("/dashboard/intelligence", IntelligenceHandler(svc, writeJSON))
}

func IntelligenceHandler(svc *Service, writeJSON WriteJSON) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		tenantID := r.URL.Query().Get("tenant_id")
		if svc.EnforceTenant {
			if tid := r.Context().Value("tenant_id"); tid != nil {
				if v, ok := tid.(string); ok && v != "" {
					tenantID = v
				}
			} else {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
		}
		resp, err := svc.Build(r.Context(), tenantID)
		writeJSON(w, resp, err)
	}
}
