package bridgeclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hellolib/agent-notify/internal/config"
	"github.com/hellolib/agent-notify/internal/notify"
	"github.com/hellolib/agent-notify/internal/state"
)

func TryDispatch(ctx context.Context, msg notify.Message) (bool, error) {
	return RecordEvent(ctx, "", "", msg, "sent")
}

// RecordEvent persists a host-side result without re-dispatching channels.
func RecordEvent(ctx context.Context, base, token string, msg notify.Message, result string) (bool, error) {
	if base == "" || token == "" {
		defaultBase, defaultToken := credentials()
		if base == "" {
			base = defaultBase
		}
		if token == "" {
			token = defaultToken
		}
	}
	if base == "" && token == "" {
		return false, nil
	}
	if base == "" {
		base = "http://127.0.0.1:45173"
	}
	if token == "" {
		return false, fmt.Errorf("bridge token is missing (set AGENT_NOTIFY_BRIDGE_TOKEN or run ./deploy.sh up)")
	}
	body, err := json.Marshal(msg)
	if err != nil {
		return false, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/events", bytes.NewReader(body))
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-Notify-Journal-Only", "true")
	req.Header.Set("X-Agent-Notify-Result", result)
	resp, err := (&http.Client{Timeout: 3 * time.Second}).Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false, fmt.Errorf("bridge returned HTTP %d", resp.StatusCode)
	}
	return true, nil
}

// ListEvents reads the Docker Bridge event journal. Empty credentials use the
// same environment and persisted-token resolution as hook forwarding.
func ListEvents(ctx context.Context, base, token string) ([]state.EventRecord, error) {
	if base == "" || token == "" {
		defaultBase, defaultToken := credentials()
		if base == "" {
			base = defaultBase
		}
		if token == "" {
			token = defaultToken
		}
	}
	if base == "" {
		base = "http://127.0.0.1:45173"
	}
	if token == "" {
		return nil, fmt.Errorf("bridge token is missing (run ./deploy.sh up)")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(base, "/")+"/api/events", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := (&http.Client{Timeout: 3 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("bridge returned HTTP %d", resp.StatusCode)
	}
	var events []state.EventRecord
	if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
		return nil, err
	}
	return events, nil
}

// GetConfig reads the Docker Bridge runtime configuration.
func GetConfig(ctx context.Context, base, token string) (config.Config, error) {
	resp, err := request(ctx, http.MethodGet, "/api/config", base, token, nil)
	if err != nil {
		return config.Config{}, err
	}
	defer resp.Body.Close()
	var cfg config.Config
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		return config.Config{}, err
	}
	return cfg, nil
}

// SaveConfig replaces the Docker Bridge runtime configuration.
func SaveConfig(ctx context.Context, base, token string, cfg config.Config) error {
	body, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	resp, err := request(ctx, http.MethodPut, "/api/config", base, token, bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// FreezeRemote pauses configured remote channels while preserving local system notifications.
func FreezeRemote(ctx context.Context, base, token string, durationSeconds int) (state.FreezeState, error) {
	body, err := json.Marshal(struct {
		DurationSeconds int `json:"duration_seconds"`
	}{DurationSeconds: durationSeconds})
	if err != nil {
		return state.FreezeState{}, err
	}
	resp, err := request(ctx, http.MethodPut, "/api/freeze", base, token, bytes.NewReader(body))
	if err != nil {
		return state.FreezeState{}, err
	}
	defer resp.Body.Close()
	var freeze state.FreezeState
	if err := json.NewDecoder(resp.Body).Decode(&freeze); err != nil {
		return state.FreezeState{}, err
	}
	return freeze, nil
}

// ClearFreeze resumes all remote notification channels.
func ClearFreeze(ctx context.Context, base, token string) error {
	resp, err := request(ctx, http.MethodDelete, "/api/freeze", base, token, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// TestRemoteChannel asks the bridge to send through only the named remote
// channel, rather than fan-out through every enabled destination.
func TestRemoteChannel(ctx context.Context, base, token, channel string) error {
	body, err := json.Marshal(struct {
		Channel string `json:"channel"`
	}{Channel: channel})
	if err != nil {
		return err
	}
	resp, err := request(ctx, http.MethodPost, "/api/test-channel", base, token, bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func request(ctx context.Context, method, path, base, token string, body io.Reader) (*http.Response, error) {
	if base == "" || token == "" {
		defaultBase, defaultToken := credentials()
		if base == "" {
			base = defaultBase
		}
		if token == "" {
			token = defaultToken
		}
	}
	if base == "" {
		base = "http://127.0.0.1:45173"
	}
	if token == "" {
		return nil, fmt.Errorf("bridge token is missing (run ./deploy.sh up)")
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(base, "/")+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := (&http.Client{Timeout: 3 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, fmt.Errorf("bridge returned HTTP %d", resp.StatusCode)
	}
	return resp, nil
}

func credentials() (string, string) {
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("AGENT_NOTIFY_BRIDGE_URL")), "/")
	tokenPath := strings.TrimSpace(os.Getenv("AGENT_NOTIFY_BRIDGE_TOKEN_FILE"))
	if tokenPath == "" {
		if path, err := config.BridgeTokenPath(); err == nil {
			tokenPath = path
		}
	}
	token := strings.TrimSpace(os.Getenv("AGENT_NOTIFY_BRIDGE_TOKEN"))
	if token == "" && tokenPath != "" {
		if data, err := os.ReadFile(filepath.Clean(tokenPath)); err == nil {
			token = strings.TrimSpace(string(data))
		}
	}
	return base, token
}
