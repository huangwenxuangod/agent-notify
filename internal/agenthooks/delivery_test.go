package agenthooks

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/hellolib/agent-notify/internal/config"
	"github.com/hellolib/agent-notify/internal/notify"
	"github.com/hellolib/agent-notify/internal/state"
)

func TestOverallDeliveryResultCountsSystemSuccessWithoutRemoteSenders(t *testing.T) {
	if got := overallDeliveryResult(true, nil, false); got != "sent" {
		t.Fatalf("overallDeliveryResult() = %q, want sent", got)
	}
}

func TestOverallDeliveryResultMarksSystemSuccessAndRemoteFailurePartial(t *testing.T) {
	err := &notify.DeliveryError{Details: []string{"ntfy: down"}}
	if got := overallDeliveryResult(true, err, true); got != "partial" {
		t.Fatalf("overallDeliveryResult() = %q, want partial", got)
	}
}

func TestOverallDeliveryResultMarksRemoteFailureAsError(t *testing.T) {
	if got := overallDeliveryResult(false, errors.New("ntfy: down"), true); got != "error" {
		t.Fatalf("overallDeliveryResult() = %q, want error", got)
	}
}

func TestRetryRemoteOutboxRemovesDeliveredRecord(t *testing.T) {
	root := t.TempDir()
	statePath := filepath.Join(root, "state.json")
	cfg := config.Default()
	cfg.Notify.Codex.Events = []string{"run_completed"}
	cfg.Remote.Ntfy.Enabled = true
	cfg.Remote.Ntfy.TopicURL = "http://127.0.0.1:1/unreachable"
	outbox := state.NewRemoteOutbox(state.RemoteOutboxPath(statePath))
	if err := outbox.Enqueue(state.RemoteOutboxItem{
		Agent: "codex", Event: "run_completed", Channels: []string{"ntfy"}, NextTry: time.Now().Add(-time.Second),
	}); err != nil {
		t.Fatal(err)
	}
	if got, err := RetryRemoteOutbox(context.Background(), cfg, statePath, func(context.Context, notify.Message, []notify.Sender) error { return nil }); err != nil || got != 1 {
		t.Fatalf("RetryRemoteOutbox() = %d, %v; want 1, nil", got, err)
	}
	due, err := outbox.Due(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(due) != 0 {
		t.Fatalf("due retry records = %d, want 0", len(due))
	}
}

func TestRetryRemoteOutboxProcessesOnlyOneDueItemPerRun(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	cfg := config.Default()
	cfg.Notify.Codex.Events = []string{"run_completed"}
	cfg.Remote.Ntfy.Enabled = true
	cfg.Remote.Ntfy.TopicURL = "https://ntfy.sh/test"
	outbox := state.NewRemoteOutbox(state.RemoteOutboxPath(statePath))
	for i := 0; i < 2; i++ {
		if err := outbox.Enqueue(state.RemoteOutboxItem{Agent: "codex", Event: "run_completed", Channels: []string{"ntfy"}, NextTry: time.Now().Add(-time.Second)}); err != nil {
			t.Fatal(err)
		}
	}
	calls := 0
	if _, err := RetryRemoteOutbox(context.Background(), cfg, statePath, func(context.Context, notify.Message, []notify.Sender) error {
		calls++
		return errors.New("temporary failure")
	}); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("send calls = %d, want one per retry tick", calls)
	}
	items, err := outbox.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("outbox items = %d, want both retained for backoff", len(items))
	}
}

func TestRetryRemoteOutboxDropsPreconditionFailures(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	cfg := config.Default()
	cfg.Notify.Codex.Events = []string{"run_completed"}
	cfg.Remote.WechatIlink.Enabled = true
	outbox := state.NewRemoteOutbox(state.RemoteOutboxPath(statePath))
	if err := outbox.Enqueue(state.RemoteOutboxItem{Agent: "codex", Event: "run_completed", Channels: []string{"wechat-ilink"}, NextTry: time.Now().Add(-time.Second)}); err != nil {
		t.Fatal(err)
	}
	if _, err := RetryRemoteOutbox(context.Background(), cfg, statePath, func(context.Context, notify.Message, []notify.Sender) error {
		return errors.New(`wechat-ilink: bridge returned 409: Send one message to the bot to bind this WeChat account`)
	}); err != nil {
		t.Fatal(err)
	}
	items, err := outbox.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("outbox items = %d, want precondition failure discarded", len(items))
	}
}
