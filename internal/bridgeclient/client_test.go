package bridgeclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/hellolib/agent-notify/internal/notify"
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
	t.Setenv("AGENT_NOTIFY_BRIDGE_TOKEN_FILE", t.TempDir()+"/missing-token")
	t.Setenv("AGENT_NOTIFY_BRIDGE_TOKEN", "")
	ok, err := TryDispatch(context.Background(), notify.Message{})
	if err != nil || ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
}

func TestTryDispatchUsesPersistedToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusAccepted) }))
	defer server.Close()
	t.Setenv("AGENT_NOTIFY_BRIDGE_URL", server.URL)
	path := t.TempDir() + "/bridge.token"
	if err := os.WriteFile(path, []byte("file-secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AGENT_NOTIFY_BRIDGE_TOKEN", "")
	t.Setenv("AGENT_NOTIFY_BRIDGE_TOKEN_FILE", path)
	ok, err := TryDispatch(context.Background(), notify.Message{Agent: "codex", Event: "run_completed"})
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
}
