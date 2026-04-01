// users.go
// - Endpoints de gestão de usuários:
//   * POST /api/v1/users                 -> cria usuário global (tenant_id NULL) [superadmin/support]
//   * GET  /api/v1/users                 -> lista usuários globais [superadmin/support]
//   * POST /api/v1/{tenantID}/users      -> cria usuário de tenant [superadmin/support/admin (no próprio tenant)]
//   * GET  /api/v1/{tenantID}/users      -> lista usuários de tenant [superadmin/support/admin (no próprio tenant)]
//
// Obs.:
// - Todos os usuários residem no MASTER DB; tenant_id NULL => global.

package app

import (
    "context"
    "encoding/json"
    "net/http"
    "strings"

    "github.com/gorilla/mux"
)

func CreateGlobalUserHandler(w http.ResponseWriter, r *http.Request) {
    // Requer superadmin/support
    u := CurrentUser(r)
    if u == nil || (u.Role != "superadmin" && u.Role != "support") {
        http.Error(w, "forbidden", http.StatusForbidden)
        return
    }
    var req struct {
        Email    string `json:"email"`
        Password string `json:"password"`
        Role     string `json:"role"` // superadmin|support
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), 400)
        return
    }
    role := strings.ToLower(req.Role)
    if role != "superadmin" && role != "support" {
        http.Error(w, "invalid role", 400)
        return
    }
    hash, err := HashPassword(req.Password)
    if err != nil {
        http.Error(w, "hash error", 500)
        return
    }
    _, err = MasterConn.Exec(context.Background(),
        `INSERT INTO users (email, password_hash, role, tenant_id) VALUES ($1,$2,$3,NULL)
         ON CONFLICT (email) DO UPDATE SET password_hash=EXCLUDED.password_hash, role=EXCLUDED.role, tenant_id=NULL`,
        req.Email, hash, role)
    if err != nil {
        http.Error(w, err.Error(), 500)
        return
    }
    w.WriteHeader(http.StatusCreated)
}

func ListGlobalUsersHandler(w http.ResponseWriter, r *http.Request) {
    // superadmin/support
    u := CurrentUser(r)
    if u == nil || (u.Role != "superadmin" && u.Role != "support") {
        http.Error(w, "forbidden", http.StatusForbidden)
        return
    }
    rows, err := MasterConn.Query(context.Background(),
        `SELECT id::text, email, role, tenant_id::text FROM users WHERE tenant_id IS NULL ORDER BY created_at DESC`)
    if err != nil { http.Error(w, err.Error(), 500); return }
    defer rows.Close()

    type Row struct { ID, Email, Role string; TenantID *string }
    var out []Row
    for rows.Next() {
        var r Row
        var tid *string
        if err := rows.Scan(&r.ID, &r.Email, &r.Role, &tid); err != nil { http.Error(w, err.Error(), 500); return }
        r.TenantID = tid
        out = append(out, r)
    }
    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(out)
}

func CreateTenantUserHandler(w http.ResponseWriter, r *http.Request) {
    // Pode ser feito por superadmin/support, ou admin do próprio tenant.
    cu := CurrentUser(r)
    if cu == nil {
        http.Error(w, "unauthorized", 401); return
    }
    tenantID := mux.Vars(r)["tenantID"]
    // admin só no próprio tenant
    if cu.Role == "admin" {
        if cu.TenantID == nil || *cu.TenantID != tenantID {
            http.Error(w, "forbidden", 403)
            return
        }
    } else if cu.Role != "superadmin" && cu.Role != "support" {
        http.Error(w, "forbidden", 403)
        return
    }

    var req struct {
        Email    string `json:"email"`
        Password string `json:"password"`
        Role     string `json:"role"` // admin|viewer
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), 400)
        return
    }
    role := strings.ToLower(req.Role)
    if role != "admin" && role != "viewer" {
        http.Error(w, "invalid role", 400)
        return
    }
    hash, err := HashPassword(req.Password)
    if err != nil {
        http.Error(w, "hash error", 500); return
    }
    _, err = MasterConn.Exec(context.Background(),
        `INSERT INTO users (email, password_hash, role, tenant_id)
         VALUES ($1,$2,$3,$4::uuid)
         ON CONFLICT (email) DO UPDATE SET password_hash=EXCLUDED.password_hash, role=EXCLUDED.role, tenant_id=$4::uuid`,
        req.Email, hash, role, tenantID)
    if err != nil {
        http.Error(w, err.Error(), 500)
        return
    }
    w.WriteHeader(http.StatusCreated)
}

func ListTenantUsersHandler(w http.ResponseWriter, r *http.Request) {
    cu := CurrentUser(r)
    if cu == nil { http.Error(w, "unauthorized", 401); return }
    tenantID := mux.Vars(r)["tenantID"]
    // admin só vê o seu próprio tenant; superadmin/support vê qualquer um
    if cu.Role == "admin" && (cu.TenantID == nil || *cu.TenantID != tenantID) {
        http.Error(w, "forbidden", 403); return
    }
    rows, err := MasterConn.Query(context.Background(),
        `SELECT id::text, email, role, tenant_id::text FROM users WHERE tenant_id=$1::uuid ORDER BY created_at DESC`, tenantID)
    if err != nil { http.Error(w, err.Error(), 500); return }
    defer rows.Close()
    type Row struct { ID, Email, Role string; TenantID *string }
    var out []Row
    for rows.Next() {
        var r Row
        var tid *string
        if err := rows.Scan(&r.ID, &r.Email, &r.Role, &tid); err != nil { http.Error(w, err.Error(), 500); return }
        r.TenantID = tid
        out = append(out, r)
    }
    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(out)
}
