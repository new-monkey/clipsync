package auth

import (
	"os"
)

// authValidate is a very small placeholder token validation used by the
// skeleton. It checks the environment CLIPSYNC_AUTH_TOKEN if present; otherwise
// it accepts any non-empty token. This will be replaced with JWT validation
// in a later PR.
func authValidate(token string) bool {
	if token == "" {
		return false
	}
	if expected := os.Getenv("CLIPSYNC_AUTH_TOKEN"); expected != "" {
		return token == expected
	}
	// no env set: accept non-empty token but note this is insecure
	return true
}
