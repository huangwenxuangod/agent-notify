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
