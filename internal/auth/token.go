package auth

import (
	"fmt"
	"os"

	"github.com/golang-jwt/jwt/v5"
)

// ValidateToken validates a JWT using HS256 with the secret from
// CLIPSYNC_JWT_SECRET. For short-term development convenience, if the
// env CLIPSYNC_JWT_SECRET is not set but CLIPSYNC_AUTH_TOKEN is set, this
// function will accept that exact token string (legacy fallback). If neither
// env var is set, any non-empty token is accepted (INSECURE - only for local dev).
func ValidateToken(tokenString string) bool {
	if tokenString == "" {
		return false
	}

	secret := os.Getenv("CLIPSYNC_JWT_SECRET")
	if secret == "" {
		// legacy fallback
		expected := os.Getenv("CLIPSYNC_AUTH_TOKEN")
		if expected != "" {
			return tokenString == expected
		}
		// no secret configured: accept non-empty token (developer mode)
		return true
	}

	// parse JWT
	_, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		// require HMAC signing method
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return false
	}
	return true
}
