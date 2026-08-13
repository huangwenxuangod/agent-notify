package state_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/hellolib/agent-notify/internal/state"
)

func TestRemoteOutboxRemovesItemAfterSuccessfulRetry(t *testing.T) {
	store := state.NewRemoteOutbox(filepath.Join(t.TempDir(), "remote-outbox.json"))
	item := state.RemoteOutboxItem{
		Agent:     "codex",
		Event:     "run_completed",
		SessionID: "s1",
		Channels:  []string{"ntfy"},
		NextTry:   time.Now().Add(-time.Second),
	}
	if err := store.Enqueue(item); err != nil {
		t.Fatal(err)
	}
	items, err := store.Due(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("due items = %d, want 1", len(items))
	}
	if err := store.Remove(items[0].ID); err != nil {
		t.Fatal(err)
	}
	items, err = store.Due(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("due items after remove = %d, want 0", len(items))
	}
}
