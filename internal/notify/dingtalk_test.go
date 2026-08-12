package notify

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDingTalkSenderSignsWhenSecretConfigured(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("timestamp") == "" {
			t.Error("timestamp query parameter is missing")
		}
		if !strings.Contains(r.URL.RawQuery, "sign=") {
			t.Error("sign query parameter is missing")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
	}))
	defer server.Close()

	sender := NewDingTalkSenderWithSecret(server.URL+"?access_token=test", "secret")
	if err := sender.Send(context.Background(), Message{Title: "Test", Body: "hello"}); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
}
