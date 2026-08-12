package bridge_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/hellolib/agent-notify/internal/bridge"
	"github.com/hellolib/agent-notify/internal/config"
)

func TestEnsureTokenCreatesOwnerOnlyToken(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bridge.token")
	token, err := bridge.EnsureToken(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(token) < 32 {
		t.Fatalf("token length = %d, want at least 32", len(token))
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o, want 600", info.Mode().Perm())
	}
	second, err := bridge.EnsureToken(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(token) != string(second) {
		t.Fatal("EnsureToken changed an existing token")
	}
}

func TestHTTPHandlerSavesConfigWithValidToken(t *testing.T) {
	root := t.TempDir()
	svc, err := bridge.NewService(bridge.Options{ConfigPath: filepath.Join(root, "config.yaml"), StatePath: filepath.Join(root, "state.json"), LogPath: filepath.Join(root, "log")})
	if err != nil {
		t.Fatal(err)
	}
	h := bridge.NewHTTPHandler(svc, []byte("test-token"))
	cfg := config.Default()
	cfg.Agent.Codex.Enabled = true
	body, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPut, "/api/config", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer test-token")
	put := httptest.NewRecorder()
	h.ServeHTTP(put, req)
	if put.Code != http.StatusOK {
		t.Fatalf("put status = %d body=%s", put.Code, put.Body.String())
	}
	getReq := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	getReq.Header.Set("Authorization", "Bearer test-token")
	get := httptest.NewRecorder()
	h.ServeHTTP(get, getReq)
	var got config.Config
	if err := json.Unmarshal(get.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if !got.Agent.Codex.Enabled {
		t.Fatal("saved config did not round-trip")
	}
}

func TestHTTPHandlerHealthIsPublicAndWritesRequireToken(t *testing.T) {
	h := bridge.NewHTTPHandler(nil, []byte("test-token"))

	health := httptest.NewRecorder()
	h.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	if health.Code != http.StatusOK {
		t.Fatalf("health status = %d, want 200", health.Code)
	}
	var payload map[string]any
	if err := json.Unmarshal(health.Body.Bytes(), &payload); err != nil || payload["status"] != "ok" {
		t.Fatalf("health body = %s", health.Body.String())
	}

	unauthorized := httptest.NewRecorder()
	h.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodPut, "/api/config", nil))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d, want 401", unauthorized.Code)
	}

	unknown := httptest.NewRecorder()
	h.ServeHTTP(unknown, httptest.NewRequest(http.MethodGet, "/nope", nil))
	if unknown.Code != http.StatusNotFound {
		t.Fatalf("unknown status = %d, want 404", unknown.Code)
	}
}

func TestHTTPHandlerAcceptsEventIngest(t *testing.T) {
	root := t.TempDir()
	svc, err := bridge.NewService(bridge.Options{ConfigPath: filepath.Join(root, "config.yaml"), StatePath: filepath.Join(root, "state.json"), LogPath: filepath.Join(root, "log")})
	if err != nil {
		t.Fatal(err)
	}
	h := bridge.NewHTTPHandler(svc, []byte("token"))
	body := bytes.NewBufferString(`{"agent":"codex","event":"run_completed","sessionid":"s1"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/events", body)
	req.Header.Set("Authorization", "Bearer token")
	r := httptest.NewRecorder()
	h.ServeHTTP(r, req)
	if r.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", r.Code, r.Body.String())
	}
}

func TestHTTPHandlerRecordsHostEventWithoutDispatchingItAgain(t *testing.T) {
	root := t.TempDir()
	svc, err := bridge.NewService(bridge.Options{ConfigPath: filepath.Join(root, "config.yaml"), StatePath: filepath.Join(root, "state.json"), LogPath: filepath.Join(root, "log")})
	if err != nil {
		t.Fatal(err)
	}
	h := bridge.NewHTTPHandler(svc, []byte("token"))
	req := httptest.NewRequest(http.MethodPost, "/api/events", bytes.NewBufferString(`{"agent":"codex","event":"run_completed","sessionid":"host-s1"}`))
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("X-Agent-Notify-Journal-Only", "true")
	recorder := httptest.NewRecorder()
	h.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	events, err := svc.ListEvents(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Result != "sent" || events[0].SessionID != "host-s1" {
		t.Fatalf("events=%#v, want recorded host event", events)
	}
}

func TestHTTPHandlerFreezesConfiguredRemoteChannels(t *testing.T) {
	root := t.TempDir()
	svc, err := bridge.NewService(bridge.Options{ConfigPath: filepath.Join(root, "config.yaml"), StatePath: filepath.Join(root, "state.json"), LogPath: filepath.Join(root, "log")})
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.Remote.Ntfy.Enabled = true
	cfg.Remote.Ntfy.TopicURL = "https://ntfy.sh/test"
	if err := svc.SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	h := bridge.NewHTTPHandler(svc, []byte("token"))
	req := httptest.NewRequest(http.MethodPut, "/api/freeze", bytes.NewBufferString(`{"duration_seconds":1800}`))
	req.Header.Set("Authorization", "Bearer token")
	recorder := httptest.NewRecorder()
	h.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var value struct {
		Channels []string `json:"channels"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &value); err != nil {
		t.Fatal(err)
	}
	if len(value.Channels) != 1 || value.Channels[0] != "ntfy" {
		t.Fatalf("channels=%v", value.Channels)
	}
}

func TestHTTPHandlerDoesNotFreezeEmptyDefaultRemoteChannels(t *testing.T) {
	root := t.TempDir()
	svc, err := bridge.NewService(bridge.Options{ConfigPath: filepath.Join(root, "config.yaml"), StatePath: filepath.Join(root, "state.json"), LogPath: filepath.Join(root, "log")})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.SaveConfig(config.Default()); err != nil {
		t.Fatal(err)
	}
	h := bridge.NewHTTPHandler(svc, []byte("token"))
	req := httptest.NewRequest(http.MethodPut, "/api/freeze", bytes.NewBufferString(`{"duration_seconds":1800}`))
	req.Header.Set("Authorization", "Bearer token")
	recorder := httptest.NewRecorder()
	h.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestHTTPHandlerTestsOnlyRequestedRemoteChannel(t *testing.T) {
	var ntfyCalls, slackCalls int
	ntfy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ntfyCalls++
		w.WriteHeader(http.StatusOK)
	}))
	defer ntfy.Close()
	slack := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slackCalls++
		w.WriteHeader(http.StatusOK)
	}))
	defer slack.Close()

	root := t.TempDir()
	svc, err := bridge.NewService(bridge.Options{ConfigPath: filepath.Join(root, "config.yaml"), StatePath: filepath.Join(root, "state.json"), LogPath: filepath.Join(root, "log")})
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.Remote.Ntfy.Enabled = true
	cfg.Remote.Ntfy.TopicURL = ntfy.URL + "/topic"
	cfg.Remote.Slack.Enabled = true
	cfg.Remote.Slack.WebhookURL = slack.URL
	if err := svc.SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	h := bridge.NewHTTPHandler(svc, []byte("token"))
	req := httptest.NewRequest(http.MethodPost, "/api/test-channel", bytes.NewBufferString(`{"channel":"ntfy"}`))
	req.Header.Set("Authorization", "Bearer token")
	recorder := httptest.NewRecorder()
	h.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if ntfyCalls != 1 || slackCalls != 0 {
		t.Fatalf("ntfyCalls=%d slackCalls=%d", ntfyCalls, slackCalls)
	}
}
