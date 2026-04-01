// tenant_info.go
// - Implementa GET /api/v1/tenants/{tenantID}
// - Retorna os dados do tenant no MASTER DB: id, name, db_name
// - Usado pela UI para mostrar o nome do cliente (tenant.name) em vez do UUID.

package app

import (
    "context"
    "encoding/json"
    "net/http"

    "github.com/gorilla/mux"
)

func GetTenantInfoHandler(w http.ResponseWriter, r *http.Request) {
    tenantID := mux.Vars(r)["tenantID"]

    var out struct {
        ID     string `json:"id"`
        Name   string `json:"name"`
        DBName string `json:"db_name"`
    }

    err := MasterConn.QueryRow(context.Background(),
        "SELECT id::text, name, db_name FROM tenants WHERE id=$1",
        tenantID,
    ).Scan(&out.ID, &out.Name, &out.DBName)

    if err != nil {
        http.Error(w, "tenant not found", http.StatusNotFound)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(out)
}
