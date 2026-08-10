package bridgeclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/hellolib/agent-notify/internal/notify"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func TryDispatch(ctx context.Context, msg notify.Message) (bool, error) {
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("AGENT_NOTIFY_BRIDGE_URL")), "/")
	if base == "" {
		return false, nil
	}
	token := strings.TrimSpace(os.Getenv("AGENT_NOTIFY_BRIDGE_TOKEN"))
	if token == "" {
		return false, fmt.Errorf("AGENT_NOTIFY_BRIDGE_TOKEN is required when bridge mode is enabled")
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
