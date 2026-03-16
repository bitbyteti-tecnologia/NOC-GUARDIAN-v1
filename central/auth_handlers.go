package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// =====================
// AUTH: LOGIN / ME
// =====================

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var id, email, role, pwHash string
	var tenantID *string

	err := MasterConn.QueryRow(context.Background(),
		`SELECT id::text, email, role, password_hash, tenant_id::text
         FROM users WHERE email=$1`, req.Email).
		Scan(&id, &email, &role, &pwHash, &tenantID)

	if err != nil {
		log.Printf("[AUTH] login query error email=%s err=%v", req.Email, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "invalid credentials"})
		return
	}

	// Log seguro (sem senha)
	prefix := pwHash
	if len(prefix) > 12 {
		prefix = prefix[:12]
	}
	log.Printf("[AUTH] user found email=%s role=%s hash_prefix=%s hash_len=%d tenant_nil=%v",
		email, role, prefix, len(pwHash), tenantID == nil)

	if err := bcrypt.CompareHashAndPassword([]byte(pwHash), []byte(req.Password)); err != nil {
		log.Printf("[AUTH] bcrypt FAIL email=%s err=%v", email, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "invalid credentials"})
		return
	}

	log.Printf("[AUTH] bcrypt OK email=%s", email)

	user := AuthUser{ID: id, Email: email, Role: role, TenantID: tenantID}
	access, err := GenerateJWT(user, int(accessTTL().Hours()))
	if err != nil {
		http.Error(w, "token error", http.StatusInternalServerError)
		return
	}

	ua := r.Header.Get("User-Agent")
	ip := clientIP(r)
	rawRefresh, err := createRefreshToken(r.Context(), id, ua, ip)
	if err != nil {
		http.Error(w, "refresh error", http.StatusInternalServerError)
		return
	}
	setRefreshCookie(w, rawRefresh)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"token": access,
		"user":  user,
	})
}

func MeHandler(w http.ResponseWriter, r *http.Request) {
	u := CurrentUser(r)
	w.Header().Set("Content-Type", "application/json")
	if u == nil {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "unauthorized"})
		return
	}
	_ = json.NewEncoder(w).Encode(u)
}

// =====================
// AUTH: REFRESH / LOGOUT
// =====================

func RefreshHandler(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("refresh_token")
	if err != nil {
		http.Error(w, "no refresh cookie", http.StatusUnauthorized)
		return
	}

	userID, err := validateRefresh(r.Context(), c.Value)
	if err != nil {
		http.Error(w, "invalid refresh", http.StatusUnauthorized)
		return
	}

	var email, role string
	var tenantID *string
	err = MasterConn.QueryRow(r.Context(),
		`SELECT email, role, tenant_id::text FROM users WHERE id=$1`,
		userID,
	).Scan(&email, &role, &tenantID)
	if err != nil {
		http.Error(w, "user not found", http.StatusUnauthorized)
		return
	}

	user := AuthUser{ID: userID, Email: email, Role: role, TenantID: tenantID}
	access, err := GenerateJWT(user, int(accessTTL().Hours()))
	if err != nil {
		http.Error(w, "token error", http.StatusInternalServerError)
		return
	}

	ua := r.Header.Get("User-Agent")
	ip := clientIP(r)
	newRefresh, err := rotateRefresh(r.Context(), c.Value, userID, ua, ip)
	if err != nil {
		http.Error(w, "refresh rotate error", http.StatusInternalServerError)
		return
	}
	setRefreshCookie(w, newRefresh)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"token": access})
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie("refresh_token"); err == nil {
		_ = revokeRefreshByHash(r.Context(), hashToken(c.Value))
	}
	clearRefreshCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// =====================
// AUTH: CHANGE / FORGOT / RESET
// =====================

func ChangePasswordHandler(w http.ResponseWriter, r *http.Request) {
	u := CurrentUser(r)
	if u == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Current string `json:"current"`
		New     string `json:"new"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var hash string
	err := MasterConn.QueryRow(r.Context(),
		"SELECT password_hash FROM users WHERE id=$1", u.ID).
		Scan(&hash)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Current)) != nil {
		http.Error(w, "current password invalid", http.StatusUnauthorized)
		return
	}

	newHash, err := HashPassword(req.New)
	if err != nil {
		http.Error(w, "hash error", http.StatusInternalServerError)
		return
	}

	_, err = MasterConn.Exec(r.Context(),
		"UPDATE users SET password_hash=$1 WHERE id=$2", newHash, u.ID)
	if err != nil {
		http.Error(w, "update error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func ForgotPasswordHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var id string
	err := MasterConn.QueryRow(r.Context(),
		"SELECT id::text FROM users WHERE email=$1", req.Email).
		Scan(&id)

	// Não vaza existência do usuário
	if err != nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	raw, err := genRandomToken(32)
	if err != nil {
		http.Error(w, "token gen error", http.StatusInternalServerError)
		return
	}
	hash := hashToken(raw)

	// Invalida tokens anteriores ainda não usados (mantém auditoria; apenas 1 token ativo por usuário)
	if _, err := MasterConn.Exec(r.Context(),
		`UPDATE password_resets
            SET used_at = NOW()
          WHERE user_id = $1::uuid
            AND used_at IS NULL`,
		id,
	); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	// Salva SOMENTE o hash do token (nunca o token raw)
	// Expiração: ajuste conforme sua política (ex.: 30 min, 1h, 2h)
	_, err = MasterConn.Exec(r.Context(),
		`INSERT INTO password_resets (user_id, token_hash, expires_at, created_at)
         VALUES ($1, $2, NOW() + INTERVAL '1 hour', NOW())`,
		id, hash,
	)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	// TODO: Enviar e-mail com o token RAW no link de reset.
	// Exemplo (ajuste conforme seu frontend):
	// resetURL := fmt.Sprintf("%s/reset-password?token=%s", PublicURL, raw)
	// _ = SendResetEmail(req.Email, resetURL)

	// Resposta silenciosa (não revela se o usuário existe)
	w.WriteHeader(http.StatusNoContent)
}

func clientIP(r *http.Request) string {
	// Atenção: confie em X-Forwarded-For apenas se você estiver atrás de proxy confiável
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// pode vir "ip1, ip2, ip3"
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			if ip != "" {
				return ip
			}
		}
	}

	xrip := strings.TrimSpace(r.Header.Get("X-Real-IP"))
	if xrip != "" {
		return xrip
	}

	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}

func ResetPasswordHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token string `json:"token"`
		New   string `json:"new"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Token) == "" || strings.TrimSpace(req.New) == "" {
		http.Error(w, "missing token or new password", http.StatusBadRequest)
		return
	}

	tokenHash := hashToken(req.Token)

	tx, err := MasterConn.Begin(r.Context())
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	// Busca o reset válido (não expirado e não usado) e trava a linha para evitar corrida
	var userID string
	err = tx.QueryRow(r.Context(),
		`SELECT user_id::text
           FROM password_resets
          WHERE token_hash = $1
            AND expires_at > NOW()
            AND used_at IS NULL
          FOR UPDATE`,
		tokenHash,
	).Scan(&userID)
	if err != nil {
		http.Error(w, "invalid or expired token", http.StatusBadRequest)
		return
	}

	newHash, err := HashPassword(req.New)
	if err != nil {
		http.Error(w, "hash error", http.StatusInternalServerError)
		return
	}

	// Atualiza a senha
	_, err = tx.Exec(r.Context(),
		`UPDATE users SET password_hash=$1 WHERE id=$2`,
		newHash, userID,
	)
	if err != nil {
		http.Error(w, "update error", http.StatusInternalServerError)
		return
	}

	// Marca o token como usado (auditoria)
	_, err = tx.Exec(r.Context(),
		`UPDATE password_resets SET used_at=NOW() WHERE token_hash=$1`,
		tokenHash,
	)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	// Opcional (recomendado): revogar refresh tokens ativos do usuário ao resetar senha
	// _, _ = tx.Exec(r.Context(),
	//     `UPDATE refresh_tokens
	//         SET revoked_at=NOW()
	//       WHERE user_id=$1 AND revoked_at IS NULL`,
	//     userID,
	// )

	if err := tx.Commit(r.Context()); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
