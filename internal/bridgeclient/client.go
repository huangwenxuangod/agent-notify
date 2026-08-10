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
)

func TryDispatch(ctx context.Context, msg notify.Message) (bool, error) {
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
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
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
