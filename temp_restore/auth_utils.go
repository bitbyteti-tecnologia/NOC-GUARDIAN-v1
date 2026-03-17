// auth_utils.go
// - Funções para hash de senha (bcrypt) e geração/validação de JWT.

package main

import (
    "errors"
    "os"
    "time"

    "github.com/golang-jwt/jwt/v5"
    "golang.org/x/crypto/bcrypt"
)

func HashPassword(pw string) (string, error) {
    b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
    return string(b), err
}

func CheckPassword(hash, pw string) bool {
    return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

func jwtSecret() []byte {
    s := os.Getenv("JWT_SECRET")
    if s == "" {
        // fallback em dev
        s = "dev-jwt-secret-change-me"
    }
    return []byte(s)
}

func GenerateJWT(u AuthUser, ttlHours int) (string, error) {
    now := time.Now().UTC()
    claims := jwt.MapClaims{
        "sub":       u.ID,
        "email":     u.Email,
        "role":      u.Role,
        "tenant_id": u.TenantID,
        "iat":       now.Unix(),
        "exp":       now.Add(time.Duration(ttlHours) * time.Hour).Unix(),
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(jwtSecret())
}

func ParseJWT(tokenStr string) (*TokenClaims, error) {
    token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
        if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, errors.New("invalid signing method")
        }
        return jwtSecret(), nil
    })
    if err != nil || !token.Valid {
        return nil, errors.New("invalid token")
    }
    mc, ok := token.Claims.(jwt.MapClaims)
    if !ok {
        return nil, errors.New("invalid claims")
    }
    var tc TokenClaims
    tc.Sub, _ = mc["sub"].(string)
    tc.Email, _ = mc["email"].(string)
    tc.Role, _ = mc["role"].(string)
    if v, ok := mc["tenant_id"]; ok && v != nil {
        if s, ok2 := v.(string); ok2 {
            tc.TenantID = &s
        }
    }
    if v, ok := mc["exp"].(float64); ok { tc.Exp = int64(v) }
    if v, ok := mc["iat"].(float64); ok { tc.Iat = int64(v) }
    return &tc, nil
}
