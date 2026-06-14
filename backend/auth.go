package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// ---- JWT (simple HMAC-based, no external deps) ----

var jwtSecret []byte

func init() {
	if s := os.Getenv("JWT_SECRET"); s != "" {
		jwtSecret = []byte(s)
	} else {
		jwtSecret = make([]byte, 32)
		rand.Read(jwtSecret)
		log.Printf("JWT_SECRET not set — generated random secret (sessions won't survive restart)")
	}
}

// GenerateJWT creates a signed JWT-like token for a user (15 min expiry).
func GenerateJWT(store *Store, userID int64, username string) (string, error) {
	// Fetch current token version from DB.
	var version int
	store.DB.QueryRow(`SELECT token_version FROM users WHERE id = ?`, userID).Scan(&version)

	header := b64Encode([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload := b64Encode([]byte(fmt.Sprintf(
		`{"sub":%d,"name":"%s","ver":%d,"iat":%d,"exp":%d}`,
		userID, username, version, time.Now().Unix(), time.Now().Add(15*time.Minute).Unix(),
	)))
	signingInput := header + "." + payload
	sig := hmacSHA256(jwtSecret, []byte(signingInput))
	return signingInput + "." + b64Encode(sig), nil
}

// GenerateRefreshToken creates a random refresh token, stores its hash, and returns the raw token.
func GenerateRefreshToken(store *Store, userID int64) (string, error) {
	raw, hash, _ := GenerateAPIToken() // reuse the same token generation
	err := store.CreateRefreshToken(userID, hash, time.Now().Add(72*time.Hour))
	if err != nil {
		return "", err
	}
	return raw, nil
}

// ValidateAndConsumeRefreshToken validates a refresh token and deletes it atomically.
func ValidateAndConsumeRefreshToken(store *Store, rawToken string) (int64, error) {
	sum := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(sum[:])
	return store.ConsumeRefreshToken(tokenHash)
}

// VerifyJWT validates a token and returns the user ID.
func VerifyJWT(token string) (int64, string, error) {
	parts := strings.SplitN(token, ".", 3)
	if len(parts) != 3 {
		return 0, "", fmt.Errorf("malformed token")
	}
	signingInput := parts[0] + "." + parts[1]
	expectedSig := hmacSHA256(jwtSecret, []byte(signingInput))
	actualSig, err := b64Decode(parts[2])
	if err != nil || subtle.ConstantTimeCompare(expectedSig, actualSig) != 1 {
		return 0, "", fmt.Errorf("invalid signature")
	}

	payload, err := b64Decode(parts[1])
	if err != nil {
		return 0, "", fmt.Errorf("invalid payload")
	}

	var claims struct {
		Sub  int64  `json:"sub"`
		Name string `json:"name"`
		Exp  int64  `json:"exp"`
		Ver  int    `json:"ver"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return 0, "", fmt.Errorf("invalid payload: %w", err)
	}

	if time.Now().Unix() > claims.Exp {
		return 0, "", fmt.Errorf("token expired")
	}
	// Token version is checked by JWTAuth middleware against the DB.
	return claims.Sub, claims.Name, nil
}

// ---- Password hashing ----

func HashPassword(pw string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(bytes), err
}

func CheckPassword(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

// ---- Token generation (for API tokens) ----

func GenerateAPIToken() (token string, hash string, prefix string) {
	b := make([]byte, 32)
	rand.Read(b)
	token = "lp_" + hex.EncodeToString(b)
	sum := sha256.Sum256([]byte(token))
	hash = hex.EncodeToString(sum[:])
	prefix = token[:11] // "lp_" + first 8 hex chars
	return
}

// ---- Middleware ----

type contextKey string

const (
	ctxUserID   contextKey = "user_id"
	ctxUsername contextKey = "username"
	ctxTokenID  contextKey = "token_id"
)

// JWTAuth middleware protects dashboard API routes.
func JWTAuth(store *Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing authorization header"})
				return
			}
			token := strings.TrimPrefix(auth, "Bearer ")
			userID, username, err := VerifyJWT(token)
			if err != nil {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid or expired token"})
				return
			}

			// Verify token version hasn't been revoked (password change).
			var dbVersion int
			store.DB.QueryRow(`SELECT token_version FROM users WHERE id = ?`, userID).Scan(&dbVersion)
			// Version 0 means the JWT predates versioning — allow through.
			if dbVersion > 0 {
				// Extract version from the JWT claims again (we already parsed above, but
				// VerifyJWT doesn't return it — parse just the version).
				parts := strings.SplitN(token, ".", 3)
				if len(parts) == 3 {
					payload, _ := b64Decode(parts[1])
					var claims struct {
						Ver int `json:"ver"`
					}
					json.Unmarshal(payload, &claims)
					if claims.Ver != dbVersion {
						writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "session revoked — please log in again"})
						return
					}
				}
			}

			ctx := context.WithValue(r.Context(), ctxUserID, userID)
			ctx = context.WithValue(ctx, ctxUsername, username)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ProxyAuth middleware protects the /v1/chat/completions proxy endpoint.
// Accepts JWT (dashboard users) OR API tokens.
func ProxyAuth(store *Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing authorization header"})
				return
			}
			token := strings.TrimPrefix(auth, "Bearer ")

			// Try JWT first (dashboard users)
			userID, username, err := VerifyJWT(token)
			if err == nil {
				ctx := context.WithValue(r.Context(), ctxUserID, userID)
				ctx = context.WithValue(ctx, ctxUsername, username)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Try API token
			sum := sha256.Sum256([]byte(token))
			tokenHash := hex.EncodeToString(sum[:])
			tokenID, tokenUserID, err := store.ValidateAPITokenFull(tokenHash)
			if err != nil {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
				return
			}

			// Check cost limit.
			var usageCents, limitCents int64
			store.DB.QueryRow(`SELECT COALESCE(SUM(cost_cents), 0) FROM requests WHERE api_token_id = ?`, tokenID).Scan(&usageCents)
			store.DB.QueryRow(`SELECT COALESCE(cost_limit_cents, 0) FROM api_tokens WHERE id = ?`, tokenID).Scan(&limitCents)
			if limitCents > 0 && usageCents >= limitCents {
				writeJSON(w, http.StatusTooManyRequests, map[string]string{
					"error":       "token cost limit exceeded",
					"usage_cents": fmt.Sprintf("%d", usageCents),
					"limit_cents": fmt.Sprintf("%d", limitCents),
				})
				return
			}

			ctx := context.WithValue(r.Context(), ctxUserID, tokenUserID)
			ctx = context.WithValue(ctx, ctxTokenID, tokenID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ---- Crypto helpers ----

func b64Encode(data []byte) string {
	enc := make([]byte, ((len(data)+2)/3)*4)
	encodeBase64(enc, data)
	return string(enc)
}

func b64Decode(s string) ([]byte, error) {
	dec := make([]byte, (len(s)*3+3)/4)
	n, err := decodeBase64(dec, []byte(s))
	if err != nil {
		return nil, err
	}
	return dec[:n], nil
}

func hmacSHA256(key, data []byte) []byte {
	// Simple HMAC-SHA256 implementation
	blockSize := 64
	if len(key) > blockSize {
		h := sha256.Sum256(key)
		key = h[:]
	}
	if len(key) < blockSize {
		padded := make([]byte, blockSize)
		copy(padded, key)
		key = padded
	}

	oKeyPad := make([]byte, blockSize)
	iKeyPad := make([]byte, blockSize)
	for i := 0; i < blockSize; i++ {
		oKeyPad[i] = key[i] ^ 0x5c
		iKeyPad[i] = key[i] ^ 0x36
	}

	inner := sha256.Sum256(append(iKeyPad, data...))
	outer := sha256.Sum256(append(oKeyPad, inner[:]...))
	return outer[:]
}

// ---- Base64 (URL-safe, no padding) ----

const b64Table = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"

func encodeBase64(dst, src []byte) {
	di := 0
	for i := 0; i < len(src); i += 3 {
		val := uint(src[i]) << 16
		if i+1 < len(src) {
			val |= uint(src[i+1]) << 8
		}
		if i+2 < len(src) {
			val |= uint(src[i+2])
		}
		dst[di] = b64Table[(val>>18)&0x3f]
		di++
		dst[di] = b64Table[(val>>12)&0x3f]
		di++
		if i+1 < len(src) {
			dst[di] = b64Table[(val>>6)&0x3f]
		} else {
			dst[di] = '='
		}
		di++
		if i+2 < len(src) {
			dst[di] = b64Table[val&0x3f]
		} else {
			dst[di] = '='
		}
		di++
	}
}

func decodeBase64(dst, src []byte) (int, error) {
	lookup := make(map[byte]byte, 64)
	for i, c := range []byte(b64Table) {
		lookup[c] = byte(i)
	}

	di := 0
	for i := 0; i < len(src) && src[i] != '='; i += 4 {
		var buf [4]byte
		for j := 0; j < 4 && i+j < len(src); j++ {
			v, ok := lookup[src[i+j]]
			if !ok && src[i+j] != '=' {
				return 0, fmt.Errorf("invalid base64 char: %c", src[i+j])
			}
			buf[j] = v
		}
		dst[di] = (buf[0] << 2) | (buf[1] >> 4)
		di++
		if src[i+2] != '=' {
			dst[di] = (buf[1] << 4) | (buf[2] >> 2)
			di++
		}
		if src[i+3] != '=' {
			dst[di] = (buf[2] << 6) | buf[3]
			di++
		}
	}
	return di, nil
}

// requireAuth extracts the user from context or returns 401.
func requireAuth(w http.ResponseWriter, r *http.Request) (userID int64, ok bool) {
	v := r.Context().Value(ctxUserID)
	if v == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return 0, false
	}
	return v.(int64), true
}
