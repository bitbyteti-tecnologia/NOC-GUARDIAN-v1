// auth_sessions.go
// - Endpoints para gerenciamento de sessões (refresh tokens) do usuário logado.
// - Usa a tabela refresh_tokens no MASTER DB.
// - GET  /api/v1/auth/sessions -> lista sessões recentes
// - POST /api/v1/auth/sessions/revoke-all -> revoga todas as sessões do usuário e limpa cookie
//
// Interação:
// - Login cria um refresh_token (cookie HttpOnly) e grava hash em refresh_tokens.
// - Refresh rotaciona o refresh_token (revoga antigo e cria novo).
// - Aqui damos visão e controle operacional do NOC Guardian.

package main

import (
    "context"
    "encoding/json"
    "net/http"
    "time"
)

type SessionRow struct {
    ID        string     `json:"id"`
    CreatedAt time.Time  `json:"created_at"`
    ExpiresAt time.Time  `json:"expires_at"`
    RevokedAt *time.Time `json:"revoked_at,omitempty"`
    IP        *string    `json:"ip,omitempty"`
    UserAgent *string    `json:"user_agent,omitempty"`
}

func ListSessionsHandler(w http.ResponseWriter, r *http.Request) {
    u := CurrentUser(r)
    if u == nil {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusUnauthorized)
        _ = json.NewEncoder(w).Encode(map[string]any{"error": "unauthorized"})
        return
    }

    rows, err := MasterConn.Query(context.Background(), `
        SELECT id::text, created_at, expires_at, revoked_at, ip, user_agent
        FROM refresh_tokens
        WHERE user_id = $1::uuid
        ORDER BY created_at DESC
        LIMIT 50
    `, u.ID)
    if err != nil {
        http.Error(w, err.Error(), 500)
        return
    }
    defer rows.Close()

    var out []SessionRow
    for rows.Next() {
        var s SessionRow
        if err := rows.Scan(&s.ID, &s.CreatedAt, &s.ExpiresAt, &s.RevokedAt, &s.IP, &s.UserAgent); err != nil {
            http.Error(w, err.Error(), 500)
            return
        }
        out = append(out, s)
    }

    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(out)
}

func RevokeAllSessionsHandler(w http.ResponseWriter, r *http.Request) {
    u := CurrentUser(r)
    if u == nil {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusUnauthorized)
        _ = json.NewEncoder(w).Encode(map[string]any{"error": "unauthorized"})
        return
    }

    _, err := MasterConn.Exec(context.Background(), `
        UPDATE refresh_tokens
        SET revoked_at = now()
        WHERE user_id = $1::uuid AND revoked_at IS NULL
    `, u.ID)
    if err != nil {
        http.Error(w, err.Error(), 500)
        return
    }

    // UX: limpa cookie do refresh imediatamente
    clearRefreshCookie(w)
    w.WriteHeader(http.StatusNoContent)
}
