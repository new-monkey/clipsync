package auth

import (
	"fmt"
	"os"

	"github.com/golang-jwt/jwt/v5"
)

func ValidateToken(tokenString string) bool {
	if tokenString == "" {
		return false
	}

	secret := os.Getenv("CLIPSYNC_JWT_SECRET")
	if secret == "" {
		expected := os.Getenv("CLIPSYNC_AUTH_TOKEN")
		if expected != "" {
			return tokenString == expected
		}
		return false
	}

	_, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
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