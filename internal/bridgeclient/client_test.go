package bridgeclient

import (
	"context"
	"github.com/hellolib/agent-notify/internal/notify"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTryDispatchPostsAuthenticatedMessage(t *testing.T) {
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()
	t.Setenv("AGENT_NOTIFY_BRIDGE_URL", server.URL)
	t.Setenv("AGENT_NOTIFY_BRIDGE_TOKEN", "secret")
	ok, err := TryDispatch(context.Background(), notify.Message{Agent: "codex", Event: "run_completed"})
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if gotAuth != "Bearer secret" {
		t.Fatalf("authorization=%q", gotAuth)
	}
}

func TestTryDispatchIsDisabledWithoutURL(t *testing.T) {
	t.Setenv("AGENT_NOTIFY_BRIDGE_URL", "")
	ok, err := TryDispatch(context.Background(), notify.Message{})
	if err != nil || ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
}
