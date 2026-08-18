package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultWechatIlinkBridgeURL = "http://127.0.0.1:45176"

// WechatIlinkSender hands text notifications to the user-local Bun bridge.
// The bridge owns WeChat QR login, credentials, long polling and recipient binding.
type WechatIlinkSender struct {
	bridgeURL  string
	httpClient *http.Client
}

func NewWechatIlinkSender(bridgeURL string) *WechatIlinkSender {
	if strings.TrimSpace(bridgeURL) == "" {
		bridgeURL = defaultWechatIlinkBridgeURL
	}
	return &WechatIlinkSender{bridgeURL: strings.TrimRight(bridgeURL, "/"), httpClient: &http.Client{Timeout: 10 * time.Second}}
}

func (s *WechatIlinkSender) Name() string { return "wechat-ilink" }

func (s *WechatIlinkSender) Send(ctx context.Context, msg Message) error {
	payload, err := json.Marshal(struct {
		Title   string `json:"title"`
		Content string `json:"content"`
		Agent   string `json:"agent"`
		Event   string `json:"event"`
	}{Title: wechatTitle(msg), Content: wechatContent(msg), Agent: msg.Agent, Event: msg.Event})
	if err != nil {
		return fmt.Errorf("wechat-ilink: marshal payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.bridgeURL+"/send", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("wechat-ilink: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("wechat-ilink: bridge unavailable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("wechat-ilink: bridge returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}
