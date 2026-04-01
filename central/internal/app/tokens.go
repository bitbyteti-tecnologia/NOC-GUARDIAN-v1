// tokens.go
// - Geração e gerenciamento de refresh tokens (persistidos no MASTER DB).
// - Guardamos APENAS o hash (sha256) do token no banco.
// - Rotação na /auth/refresh: invalida anterior e cria novo.

package app

import (
    "context"
    "crypto/rand"
    "crypto/sha256"
    "encoding/hex"
    "errors"
    "net/http"
    "os"
    "strconv"
    "strings"
    "time"
)

func genRandomToken(n int) (string, error) {
    b := make([]byte, n)
    _, err := rand.Read(b)
    if err != nil {
        return "", err
    }
    return hex.EncodeToString(b), nil // 2*n hex chars
}

func hashToken(raw string) string {
    h := sha256.Sum256([]byte(raw))
    return hex.EncodeToString(h[:])
}

func accessTTL() time.Duration {
    v := os.Getenv("ACCESS_TOKEN_TTL_HOURS")
    if v == "" {
        return 12 * time.Hour
    }
    i, err := strconv.Atoi(v)
    if err != nil {
        return 12 * time.Hour
    }
    return time.Duration(i) * time.Hour
}

func refreshTTL() time.Duration {
    v := os.Getenv("REFRESH_TOKEN_TTL_DAYS")
    if v == "" {
        return 30 * 24 * time.Hour
    }
    i, err := strconv.Atoi(v)
    if err != nil {
        return 30 * 24 * time.Hour
    }
    return time.Duration(i) * 24 * time.Hour
}

func cookieSecure() bool {
    return strings.ToLower(os.Getenv("COOKIE_SECURE")) == "true"
}
func cookieSameSite() http.SameSite {
    switch strings.ToLower(os.Getenv("COOKIE_SAME_SITE")) {
    case "strict":
        return http.SameSiteStrictMode
    case "none":
        return http.SameSiteNoneMode
    default:
        return http.SameSiteLaxMode
    }
}

func setRefreshCookie(w http.ResponseWriter, token string) {
    domain := os.Getenv("COOKIE_DOMAIN")
    c := &http.Cookie{
        Name:     "refresh_token",
        Value:    token,
        Path:     "/",
        Domain:   domain,
        HttpOnly: true,
        Secure:   cookieSecure(),
        SameSite: cookieSameSite(),
        MaxAge:   int(refreshTTL().Seconds()),
    }
    http.SetCookie(w, c)
}

func clearRefreshCookie(w http.ResponseWriter) {
    domain := os.Getenv("COOKIE_DOMAIN")
    c := &http.Cookie{
        Name:     "refresh_token",
        Value:    "",
        Path:     "/",
        Domain:   domain,
        HttpOnly: true,
        Secure:   cookieSecure(),
        SameSite: cookieSameSite(),
        MaxAge:   -1,
    }
    http.SetCookie(w, c)
}

func createRefreshToken(ctx context.Context, userID, ua, ip string) (raw string, err error) {
    raw, err = genRandomToken(32) // 64 hex chars
    if err != nil {
        return "", err
    }
    hash := hashToken(raw)
    _, err = MasterConn.Exec(ctx, `
        INSERT INTO refresh_tokens (user_id, token_hash, user_agent, ip, expires_at)
        VALUES ($1, $2, $3, $4, now() + $5::interval)
    `, userID, hash, ua, ip, (refreshTTL()).String())
    return raw, err
}

func revokeRefreshByHash(ctx context.Context, hash string) error {
    _, err := MasterConn.Exec(ctx, `UPDATE refresh_tokens SET revoked_at=now() WHERE token_hash=$1 AND revoked_at IS NULL`, hash)
    return err
}

func validateRefresh(ctx context.Context, raw string) (userID string, err error) {
    if raw == "" {
        return "", errors.New("missing token")
    }
    h := hashToken(raw)
    err = MasterConn.QueryRow(ctx, `
        SELECT user_id::text
        FROM refresh_tokens
        WHERE token_hash=$1 AND revoked_at IS NULL AND expires_at > now()
    `, h).Scan(&userID)
    return
}

func rotateRefresh(ctx context.Context, rawOld, userID, ua, ip string) (string, error) {
    oldHash := hashToken(rawOld)
    _, _ = MasterConn.Exec(ctx, `UPDATE refresh_tokens SET revoked_at=now() WHERE token_hash=$1 AND revoked_at IS NULL`, oldHash)
    return createRefreshToken(ctx, userID, ua, ip)
}
