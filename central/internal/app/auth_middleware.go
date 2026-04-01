// auth_middleware.go
// - Middleware para validar JWT e injetar usuário no contexto.
// - Helpers RequireAuth e RequireRole.

package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
)

func ValidateApiKey(ctx context.Context, tenantID, key string) error {
	parts := strings.SplitN(key, ".", 2)
	if len(parts) != 2 {
		return errors.New("formato de chave inválido")
	}
	prefix := parts[0]

	var keyHash string
	err := MasterConn.QueryRow(ctx,
		"SELECT key_hash FROM api_keys WHERE tenant_id=$1 AND key_prefix=$2",
		tenantID, prefix).Scan(&keyHash)
	if err != nil {
		return errors.New("chave não encontrada")
	}

	h := sha256.New()
	h.Write([]byte(key))
	if hex.EncodeToString(h.Sum(nil)) != keyHash {
		return errors.New("chave inválida")
	}

	// Atualiza último uso
	_, _ = MasterConn.Exec(ctx, "UPDATE api_keys SET last_used_at=now() WHERE key_prefix=$1", prefix)
	return nil
}

type ctxKey string

const ctxUserKey ctxKey = "authUser"

func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(strings.ToLower(auth), "bearer ") {
			http.Error(w, "missing bearer token", http.StatusUnauthorized)
			return
		}
		token := strings.TrimSpace(auth[len("Bearer "):])
		claims, err := ParseJWT(token)
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
		u := &AuthUser{
			ID:    claims.Sub,
			Email: claims.Email,
			Role:  claims.Role,
		}
		u.TenantID = claims.TenantID
		ctx := context.WithValue(r.Context(), ctxUserKey, u)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func CurrentUser(r *http.Request) *AuthUser {
	v := r.Context().Value(ctxUserKey)
	if v == nil {
		return nil
	}
	u, _ := v.(*AuthUser)
	return u
}

// RequireRole verifica se o usuário tem o papel necessário.
// - Para rotas "globais", só superadmin/support.
// - Para rotas de tenant, admin/viewer podem acessar conforme necessidade.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool)
	for _, r := range roles {
		allowed[r] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u := CurrentUser(r)
			if u == nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			if allowed[u.Role] {
				next.ServeHTTP(w, r)
				return
			}
			http.Error(w, "forbidden", http.StatusForbidden)
		})
	}
}
