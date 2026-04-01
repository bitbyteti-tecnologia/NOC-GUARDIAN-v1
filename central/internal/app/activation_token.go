package app

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type ActivationTokenResponse struct {
	Token     string    `json:"token"`
	Prefix    string    `json:"prefix"`
	CreatedAt time.Time `json:"created_at"`
}

func ActivationTokenHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := muxVar(r, "tenantID")
	if tenantID == "" {
		http.Error(w, "tenant inválido", http.StatusBadRequest)
		return
	}

	token, prefix, hash, err := generateAPIKey()
	if err != nil {
		http.Error(w, "erro ao gerar token", http.StatusInternalServerError)
		return
	}

	_, err = MasterConn.Exec(context.Background(),
		`INSERT INTO api_keys (tenant_id, key_prefix, key_hash, name, created_at)
         VALUES ($1, $2, $3, $4, now())`,
		tenantID, prefix, hash, "activation-token",
	)
	if err != nil {
		http.Error(w, "erro ao salvar token", http.StatusInternalServerError)
		return
	}

	resp := ActivationTokenResponse{
		Token:     token,
		Prefix:    prefix,
		CreatedAt: time.Now().UTC(),
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func generateAPIKey() (token string, prefix string, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", "", err
	}
	hexStr := hex.EncodeToString(b)
	prefix = hexStr[:8]
	token = strings.ToLower(prefix + "." + hexStr)

	h := sha256.Sum256([]byte(token))
	hash = hex.EncodeToString(h[:])
	return token, prefix, hash, nil
}
