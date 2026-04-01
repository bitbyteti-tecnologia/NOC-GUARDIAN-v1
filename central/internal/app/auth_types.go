// auth_types.go
// - Tipos e helpers compartilhados para autenticação e autorização.

package app

type AuthUser struct {
    ID       string  `json:"id"`
    Email    string  `json:"email"`
    Role     string  `json:"role"`      // superadmin|support|admin|viewer
    TenantID *string `json:"tenant_id"` // nil = global
}

type TokenClaims struct {
    Sub      string  `json:"sub"`
    Email    string  `json:"email"`
    Role     string  `json:"role"`
    TenantID *string `json:"tenant_id,omitempty"`
    Exp      int64   `json:"exp"`
    Iat      int64   `json:"iat"`
}
