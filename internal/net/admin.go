package netw

import (
	"encoding/json"
	"net/http"
	"strings"

	"clipsync/internal/auth"
	"clipsync/internal/hub"
)

func requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Authorization header required"))
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Invalid authorization format"))
			return
		}

		if !auth.ValidateToken(parts[1]) {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("Invalid or expired token"))
			return
		}

		next(w, r)
	}
}

func NewAdminHandler(h *hub.Hub) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/channels", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			chs := h.ListChannels()
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"channels": chs})
		case "POST":
			var body struct {
				Channel string `json:"channel"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Channel == "" {
				w.WriteHeader(400)
				w.Write([]byte("invalid channel body"))
				return
			}
			h.CreateChannel(body.Channel)
			w.WriteHeader(201)
			w.Write([]byte("created"))
		default:
			w.WriteHeader(405)
		}
	}))

	mux.HandleFunc("/api/clients", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(405)
			return
		}
		cls := h.ListClients()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"clients": cls})
	}))

	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})

	return mux
}
