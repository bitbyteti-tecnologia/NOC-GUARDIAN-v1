package app

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type ActivationTokenResponse struct {
	Token     string    `json:"token"`
	Prefix    string    `json:"prefix"`
	CreatedAt time.Time `json:"created_at"`
}

type activationTokenRequest struct {
	Rotate bool `json:"rotate"`
}

func ActivationTokenGetHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := muxVar(r, "tenantID")
	if tenantID == "" {
		http.Error(w, "tenant inválido", http.StatusBadRequest)
		return
	}

	token, prefix, createdAt, err := getActiveActivationToken(r.Context(), tenantID)
	if err != nil {
		if errors.Is(err, errTokenNotFound) {
			http.Error(w, "token não encontrado", http.StatusNotFound)
			return
		}
		http.Error(w, "erro ao buscar token", http.StatusInternalServerError)
		return
	}

	resp := ActivationTokenResponse{
		Token:     token,
		Prefix:    prefix,
		CreatedAt: createdAt,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func ActivationTokenPostHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := muxVar(r, "tenantID")
	if tenantID == "" {
		http.Error(w, "tenant inválido", http.StatusBadRequest)
		return
	}

	rotate := r.URL.Query().Get("rotate") == "1"
	if !rotate && r.Body != nil {
		var req activationTokenRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Rotate {
			rotate = true
		}
	}

	if !rotate {
		token, prefix, createdAt, err := getActiveActivationToken(r.Context(), tenantID)
		if err == nil {
			resp := ActivationTokenResponse{
				Token:     token,
				Prefix:    prefix,
				CreatedAt: createdAt,
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		if err != nil && !errors.Is(err, errTokenNotFound) {
			http.Error(w, "erro ao buscar token", http.StatusInternalServerError)
			return
		}
	}

	token, prefix, hash, enc, err := generateAPIKey()
	if err != nil {
		http.Error(w, "erro ao gerar token", http.StatusInternalServerError)
		return
	}

	_, _ = MasterConn.Exec(r.Context(),
		`UPDATE api_keys SET revoked_at=now()
         WHERE tenant_id=$1 AND name='activation-token' AND revoked_at IS NULL`,
		tenantID,
	)

	_, err = MasterConn.Exec(r.Context(),
		`INSERT INTO api_keys (tenant_id, key_prefix, key_hash, key_enc, name, created_at)
         VALUES ($1, $2, $3, $4, $5, now())`,
		tenantID, prefix, hash, enc, "activation-token",
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

var errTokenNotFound = errors.New("token não encontrado")

func getActiveActivationToken(ctx context.Context, tenantID string) (string, string, time.Time, error) {
	var prefix string
	var enc string
	var createdAt time.Time
	err := MasterConn.QueryRow(ctx,
		`SELECT key_prefix, key_enc, created_at
         FROM api_keys
         WHERE tenant_id=$1 AND name='activation-token' AND revoked_at IS NULL
         ORDER BY created_at DESC
         LIMIT 1`,
		tenantID,
	).Scan(&prefix, &enc, &createdAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", time.Time{}, errTokenNotFound
		}
		return "", "", time.Time{}, err
	}
	if strings.TrimSpace(enc) == "" {
		return "", "", time.Time{}, errTokenNotFound
	}

	token, err := decryptToken(enc)
	if err != nil {
		return "", "", time.Time{}, err
	}
	return token, prefix, createdAt, nil
}

func generateAPIKey() (token string, prefix string, hash string, enc string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", "", "", err
	}
	hexStr := hex.EncodeToString(b)
	prefix = hexStr[:8]
	token = strings.ToLower(prefix + "." + hexStr)

	h := sha256.Sum256([]byte(token))
	hash = hex.EncodeToString(h[:])

	enc, err = encryptToken(token)
	if err != nil {
		return "", "", "", "", err
	}

	return token, prefix, hash, enc, nil
}

func tokenKey() ([]byte, error) {
	secret := getenv("ACTIVATION_TOKEN_SECRET", "")
	if secret == "" {
		secret = getenv("JWT_SECRET", "")
	}
	if secret == "" {
		return nil, errors.New("secret ausente")
	}
	sum := sha256.Sum256([]byte(secret))
	return sum[:], nil
}

func encryptToken(token string) (string, error) {
	key, err := tokenKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(token), nil)
	payload := append(nonce, ciphertext...)
	return base64.StdEncoding.EncodeToString(payload), nil
}

func decryptToken(enc string) (string, error) {
	key, err := tokenKey()
	if err != nil {
		return "", err
	}
	raw, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("token inválido")
	}
	nonce := raw[:gcm.NonceSize()]
	ciphertext := raw[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
