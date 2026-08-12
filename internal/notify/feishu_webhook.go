package notify

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type FeishuWebhookSender struct {
	webhookURL string
	secret     string
	client     *http.Client
}

func NewFeishuWebhookSender(webhookURL string) *FeishuWebhookSender {
	return NewFeishuWebhookSenderWithSecret(webhookURL, "")
}

// NewFeishuWebhookSenderWithSecret creates a Feishu custom-bot sender. When a
// signing secret is configured, Feishu requires timestamp/sign in the JSON body.
func NewFeishuWebhookSenderWithSecret(webhookURL, secret string) *FeishuWebhookSender {
	return &FeishuWebhookSender{webhookURL: strings.TrimSpace(webhookURL), secret: strings.TrimSpace(secret), client: &http.Client{Timeout: 10 * time.Second}}
}
func (s *FeishuWebhookSender) Name() string { return "feishu" }
func (s *FeishuWebhookSender) Send(ctx context.Context, msg Message) error {
	if s.webhookURL == "" {
		return fmt.Errorf("feishu: webhook_url is empty")
	}
	payloadValue := map[string]any{"msg_type": "text", "content": map[string]string{"text": strings.TrimSpace(msg.Title + "\n" + msg.Body)}}
	if s.secret != "" {
		timestamp := fmt.Sprintf("%d", time.Now().Unix())
		mac := hmac.New(sha256.New, []byte(s.secret))
		_, _ = mac.Write([]byte(timestamp + "\n" + s.secret))
		payloadValue["timestamp"] = timestamp
		payloadValue["sign"] = base64.StdEncoding.EncodeToString(mac.Sum(nil))
	}
	payload, err := json.Marshal(payloadValue)
	if err != nil {
		return fmt.Errorf("feishu: marshal payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.webhookURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("feishu: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("feishu: send request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("feishu: unexpected status %d", resp.StatusCode)
	}
	return nil
}
