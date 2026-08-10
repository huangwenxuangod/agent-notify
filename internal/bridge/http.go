package bridge

import (
	"encoding/json"
	"net/http"

	"github.com/hellolib/agent-notify/internal/config"
)

func NewHTTPHandler(service *Service, token []byte) http.Handler {
	mux := http.NewServeMux()
	jsonWrite := func(w http.ResponseWriter, status int, value any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(value)
	}
	withAuth := func(w http.ResponseWriter, r *http.Request) bool {
		got, err := bearerToken(r.Header.Get("Authorization"))
		if err != nil || !validToken(got, token) {
			jsonWrite(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return false
		}
		return true
	}
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		jsonWrite(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/api/agents", func(w http.ResponseWriter, r *http.Request) {
		if !withAuth(w, r) {
			return
		}
		if service == nil {
			jsonWrite(w, 500, map[string]string{"error": "bridge unavailable"})
			return
		}
		value, err := service.ScanAgents()
		if err != nil {
			jsonWrite(w, 500, map[string]string{"error": err.Error()})
			return
		}
		jsonWrite(w, 200, value)
	})
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		if !withAuth(w, r) {
			return
		}
		if service == nil {
			jsonWrite(w, 500, map[string]string{"error": "bridge unavailable"})
			return
		}
		if r.Method == http.MethodGet {
			value, err := service.GetConfig()
			if err != nil {
				jsonWrite(w, 500, map[string]string{"error": err.Error()})
				return
			}
			jsonWrite(w, 200, value)
			return
		}
		if r.Method != http.MethodPut {
			jsonWrite(w, 405, map[string]string{"error": "method not allowed"})
			return
		}
		var value config.Config
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&value) != nil {
			jsonWrite(w, 400, map[string]string{"error": "invalid json"})
			return
		}
		if err := service.SaveConfig(value); err != nil {
			jsonWrite(w, 500, map[string]string{"error": err.Error()})
			return
		}
		jsonWrite(w, 200, value)
	})
	mux.HandleFunc("/api/events", func(w http.ResponseWriter, r *http.Request) {
		if !withAuth(w, r) {
			return
		}
		if service == nil {
			jsonWrite(w, 500, map[string]string{"error": "bridge unavailable"})
			return
		}
		value, err := service.ListEvents(100)
		if err != nil {
			jsonWrite(w, 500, map[string]string{"error": err.Error()})
			return
		}
		jsonWrite(w, 200, value)
	})
	mux.HandleFunc("/api/logs", func(w http.ResponseWriter, r *http.Request) {
		if !withAuth(w, r) {
			return
		}
		if service == nil {
			jsonWrite(w, 500, map[string]string{"error": "bridge unavailable"})
			return
		}
		value, err := service.ListLogs(100)
		if err != nil {
			jsonWrite(w, 500, map[string]string{"error": err.Error()})
			return
		}
		jsonWrite(w, 200, value)
	})
	mux.HandleFunc("/api/setup/scan", func(w http.ResponseWriter, r *http.Request) {
		if !withAuth(w, r) {
			return
		}
		if service == nil {
			jsonWrite(w, 500, map[string]string{"error": "bridge unavailable"})
			return
		}
		value, err := service.ScanAgents()
		if err != nil {
			jsonWrite(w, 500, map[string]string{"error": err.Error()})
			return
		}
		jsonWrite(w, 200, value)
	})
	mux.HandleFunc("/api/setup/install", func(w http.ResponseWriter, r *http.Request) {
		if !withAuth(w, r) {
			return
		}
		if service == nil {
			jsonWrite(w, 500, map[string]string{"error": "bridge unavailable"})
			return
		}
		var req SetupRequest
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req) != nil {
			jsonWrite(w, 400, map[string]string{"error": "invalid json"})
			return
		}
		value, err := service.InstallAgents(req)
		if err != nil {
			jsonWrite(w, 500, map[string]string{"error": err.Error()})
			return
		}
		jsonWrite(w, 200, value)
	})
	mux.HandleFunc("/api/setup/uninstall", func(w http.ResponseWriter, r *http.Request) {
		if !withAuth(w, r) {
			return
		}
		if service == nil {
			jsonWrite(w, 500, map[string]string{"error": "bridge unavailable"})
			return
		}
		var req SetupRequest
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req) != nil {
			jsonWrite(w, 400, map[string]string{"error": "invalid json"})
			return
		}
		value, err := service.UninstallAgents(req)
		if err != nil {
			jsonWrite(w, 500, map[string]string{"error": err.Error()})
			return
		}
		jsonWrite(w, 200, value)
	})
	mux.HandleFunc("/api/autostart", func(w http.ResponseWriter, r *http.Request) {
		if !withAuth(w, r) {
			return
		}
		if service == nil {
			jsonWrite(w, 500, map[string]string{"error": "bridge unavailable"})
			return
		}
		if r.Method == http.MethodGet {
			jsonWrite(w, 200, service.AutostartStatus())
			return
		}
		if r.Method != http.MethodPut {
			jsonWrite(w, 405, map[string]string{"error": "method not allowed"})
			return
		}
		var req struct {
			Enabled bool `json:"enabled"`
		}
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req) != nil {
			jsonWrite(w, 400, map[string]string{"error": "invalid json"})
			return
		}
		if err := service.SetAutostart(req.Enabled); err != nil {
			jsonWrite(w, 500, map[string]string{"error": err.Error()})
			return
		}
		jsonWrite(w, 200, service.AutostartStatus())
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { mux.ServeHTTP(w, r) })
}
