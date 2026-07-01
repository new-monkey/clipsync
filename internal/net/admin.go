package netw

import (
	"encoding/json"
	"log"
	"net/http"

	"clipsync/internal/hub"
)

// NewAdminHandler returns an http.Handler that exposes simple admin endpoints
// for channels and clients. This is intentionally minimal and protected by
// environment-level network controls or additional auth added later.
func NewAdminHandler(h *hub.Hub) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/channels", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			chs := h.ListChannels()
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"channels": chs})
		case "POST":
			var body struct{ Channel string `json:"channel"` }
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
	})

	mux.HandleFunc("/api/clients", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(405)
			return
		}
		cls := h.ListClients()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"clients": cls})
	})

	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})

	return mux
}
