package notify

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestFeishuWebhookSenderPostsTextMessage(t *testing.T) {
	var payload struct {
		MsgType string `json:"msg_type"`
		Content struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	if err := NewFeishuWebhookSender(srv.URL).Send(context.Background(), Message{Title: "完成", Body: "任务已结束"}); err != nil {
		t.Fatal(err)
	}
	if payload.MsgType != "text" || payload.Content.Text != "完成\n任务已结束" {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestFeishuWebhookSenderSignsWhenSecretConfigured(t *testing.T) {
	var payload struct {
		Timestamp string `json:"timestamp"`
		Sign      string `json:"sign"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	if err := NewFeishuWebhookSenderWithSecret(srv.URL, "secret").Send(context.Background(), Message{Title: "T", Body: "B"}); err != nil {
		t.Fatal(err)
	}
	if _, err := strconv.ParseInt(payload.Timestamp, 10, 64); err != nil {
		t.Fatalf("timestamp = %q is not unix seconds: %v", payload.Timestamp, err)
	}
	if _, err := base64.StdEncoding.DecodeString(payload.Sign); err != nil {
		t.Fatalf("sign = %q is not base64: %v", payload.Sign, err)
	}
}
