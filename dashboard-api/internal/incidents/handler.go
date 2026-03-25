package incidents

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type WriteJSON func(w http.ResponseWriter, v any, err error)

type Router interface {
	Get(pattern string, handlerFn http.HandlerFunc)
}

func RegisterRoutes(r Router, svc *Service, writeJSON WriteJSON) {
	r.Get("/dashboard/incidents/{id}/details", DetailsHandler(svc, writeJSON))
}

func DetailsHandler(svc *Service, writeJSON WriteJSON) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		id := chi.URLParam(r, "id")
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
		resp, err := svc.Details(r.Context(), tenantID, id)
		writeJSON(w, resp, err)
	}
}
