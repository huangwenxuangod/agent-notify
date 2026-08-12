package bridgeclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/hellolib/agent-notify/internal/config"
	"github.com/hellolib/agent-notify/internal/notify"
	"github.com/hellolib/agent-notify/internal/state"
)

func TestTryDispatchPostsAuthenticatedMessage(t *testing.T) {
	var gotAuth string
	var gotResult string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotResult = r.Header.Get("X-Agent-Notify-Result")
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
	if gotResult != "sent" {
		t.Fatalf("result=%q, want sent", gotResult)
	}
}

func TestRecordEventPostsPartialResult(t *testing.T) {
	var gotResult string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotResult = r.Header.Get("X-Agent-Notify-Result")
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()
	if _, err := RecordEvent(context.Background(), server.URL, "secret", notify.Message{Agent: "workbuddy", Event: "run_completed"}, "partial"); err != nil {
		t.Fatal(err)
	}
	if gotResult != "partial" {
		t.Fatalf("result=%q, want partial", gotResult)
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

func TestListEventsUsesAuthenticatedBridge(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			t.Fatalf("authorization=%q", got)
		}
		if r.URL.Path != "/api/events" {
			t.Fatalf("path=%q", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode([]state.EventRecord{{Agent: "codex", Event: "run_completed"}})
	}))
	defer server.Close()

	events, err := ListEvents(context.Background(), server.URL, "secret")
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Agent != "codex" {
		t.Fatalf("events=%+v", events)
	}
}

func TestConfigRoundTripUsesAuthenticatedBridge(t *testing.T) {
	var saved config.Config
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			t.Fatalf("authorization=%q", got)
		}
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(config.Default())
		case http.MethodPut:
			if err := json.NewDecoder(r.Body).Decode(&saved); err != nil {
				t.Fatal(err)
			}
			_ = json.NewEncoder(w).Encode(saved)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	got, err := GetConfig(context.Background(), server.URL, "secret")
	if err != nil {
		t.Fatal(err)
	}
	got.Behavior.DedupeSeconds = 23
	if err := SaveConfig(context.Background(), server.URL, "secret", got); err != nil {
		t.Fatal(err)
	}
	if saved.Behavior.DedupeSeconds != 23 {
		t.Fatalf("saved dedupe=%d", saved.Behavior.DedupeSeconds)
	}
}

func TestFreezeRemoteUsesAuthenticatedBridge(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			t.Fatalf("authorization=%q", got)
		}
		if r.Method != http.MethodPut || r.URL.Path != "/api/freeze" {
			t.Fatalf("request=%s %s", r.Method, r.URL.Path)
		}
		var request struct {
			DurationSeconds int `json:"duration_seconds"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request.DurationSeconds != 3600 {
			t.Fatalf("duration=%d", request.DurationSeconds)
		}
		_ = json.NewEncoder(w).Encode(state.FreezeState{Channels: []string{"ntfy"}})
	}))
	defer server.Close()

	value, err := FreezeRemote(context.Background(), server.URL, "secret", 3600)
	if err != nil {
		t.Fatal(err)
	}
	if len(value.Channels) != 1 || value.Channels[0] != "ntfy" {
		t.Fatalf("freeze=%+v", value)
	}
}

func TestTestRemoteChannelUsesAuthenticatedBridge(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			t.Fatalf("authorization=%q", got)
		}
		if r.Method != http.MethodPost || r.URL.Path != "/api/test-channel" {
			t.Fatalf("request=%s %s", r.Method, r.URL.Path)
		}
		var request struct {
			Channel string `json:"channel"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request.Channel != "ntfy" {
			t.Fatalf("channel=%q", request.Channel)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	if err := TestRemoteChannel(context.Background(), server.URL, "secret", "ntfy"); err != nil {
		t.Fatal(err)
	}
}
