package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type Register struct {
	Email    string `json:"email"`
	DeviceID string `json:"deviceId"`
	Referral string `json:"referral,omitempty"`
	Version  string `json:"version,omitempty"`
}
type Beat struct {
	Email    string `json:"email"`
	DeviceID string `json:"deviceId"`
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/agent/register", func(w http.ResponseWriter, r *http.Request) {
		var req Register
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		log.Printf("REGISTER %s %s referral=%q version=%q ua=%q",
			req.Email, req.DeviceID, req.Referral, req.Version, r.UserAgent())
		json.NewEncoder(w).Encode(map[string]any{"ok": true, "ts": time.Now().UTC()})
	})

	mux.HandleFunc("/api/agent/beat", func(w http.ResponseWriter, r *http.Request) {
		var req Beat
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		log.Printf("BEAT %s %s ua=%q", req.Email, req.DeviceID, r.UserAgent())
		json.NewEncoder(w).Encode(map[string]any{"ok": true, "ts": time.Now().UTC()})
	})

	addr := "127.0.0.1:8787"
	log.Printf("mock API listening on http://%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
