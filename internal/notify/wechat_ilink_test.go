package notify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWechatIlinkSenderPostsNotificationToLocalBridge(t *testing.T) {
	var got struct {
		Title   string `json:"title"`
		Content string `json:"content"`
		Agent   string `json:"agent"`
		Event   string `json:"event"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/send" {
			t.Fatalf("request = %s %s, want POST /send", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	err := NewWechatIlinkSender(server.URL).Send(context.Background(), Message{
		Agent: "codex", Event: "run_completed", Title: "Codex 完成", Body: "真实测试",
	})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if got.Agent != "codex" || got.Event != "run_completed" || got.Title != "Codex 完成" || !strings.Contains(got.Content, "真实测试") {
		t.Fatalf("payload = %#v", got)
	}
}

func TestWechatIlinkSenderRejectsNonSuccessBridgeResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not bound", http.StatusConflict)
	}))
	defer server.Close()

	if err := NewWechatIlinkSender(server.URL).Send(context.Background(), Message{}); err == nil {
		t.Fatal("Send() error = nil, want bridge error")
	}
}
