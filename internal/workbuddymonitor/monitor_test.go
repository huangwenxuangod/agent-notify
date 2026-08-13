package workbuddymonitor

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParserEmitsCompletedSessionWithKnownContext(t *testing.T) {
	p := NewParser()
	p.Consume(`[2026-08-12T01:55:27.718Z] [pid=6442] [INFO] [EdgeSync] §3.4 RENAME: cid=session-1 title="Research plan" isUserDefined=false`)
	p.Consume(`[2026-08-12T01:55:27.721Z] [pid=6442] [WRAPPED_SINK_UPSERT] sid=session-1 cwd="/Users/ai1/project" isPlayground=false status=working origin=(local)`)

	event, ok := p.Consume(`[2026-08-12T01:55:27.779Z] [pid=6442] [INFLIGHT_CLEAR] sid=session-1 reason=session_end_turn`)
	if !ok {
		t.Fatal("completed session was not detected")
	}
	if event.SessionID != "session-1" || event.Workspace != "/Users/ai1/project" {
		t.Fatalf("event = %#v", event)
	}
	if event.Event != "run_completed" || event.Body != "Research plan" {
		t.Fatalf("body = %q, want conversation title", event.Body)
	}
}

func TestParserIgnoresDuplicateAndIncompleteSessionEvents(t *testing.T) {
	p := NewParser()
	if event, ok := p.Consume(`[INFO] [INFLIGHT_CLEAR] sid=session-1 reason=session_cancelled`); !ok || event.Event != "run_failed" {
		t.Fatal("cancelled session should emit run_failed")
	}
	if _, ok := p.Consume(`[INFO] [INFLIGHT_CLEAR] sid=session-2 reason=session_end_turn`); !ok {
		t.Fatal("first completion must emit")
	}
	if _, ok := p.Consume(`[INFO] [INFLIGHT_CLEAR] sid=session-2 reason=session_end_turn`); ok {
		t.Fatal("duplicate completion emitted an event")
	}
}

func TestParserEmitsWorkBuddyDesktopSessionEndTurnLog(t *testing.T) {
	p := NewParser()
	p.Consume(`[INFO] [EdgeSync] §3.4 RENAME: cid=session-ui title="UI conversation" isUserDefined=false`)
	p.Consume(`[WRAPPED_SINK_UPSERT] sid=session-ui cwd="/Users/ai1/project" status=working`)

	event, ok := p.Consume(`[EdgeSync] §4.6 合成 session_end_turn: sid=session-ui status=completed stopReason=end_turn`)
	if !ok {
		t.Fatal("WorkBuddy desktop completion was not detected")
	}
	if event.Event != "run_completed" || event.Body != "UI conversation" || event.Workspace != "/Users/ai1/project" {
		t.Fatalf("event = %#v", event)
	}
}

func TestParserEmitsWorkBuddyDesktopFailureAndCancellation(t *testing.T) {
	for _, tc := range []struct {
		name, line, wantEvent string
	}{
		{"api error", `[ERROR] sid=failed-session error="429 Too Many Requests"`, "run_failed"},
		{"cancelled", `[INFLIGHT_CLEAR] sid=cancelled-session reason=session_cancelled`, "run_failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			event, ok := NewParser().Consume(tc.line)
			if !ok || event.Event != tc.wantEvent {
				t.Fatalf("event=%#v ok=%v, want %s", event, ok, tc.wantEvent)
			}
		})
	}
}

func TestWatchContinuesAfterWorkBuddyRotatesMainLog(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.log")
	if err := os.WriteFile(path, []byte("old log\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events := make(chan Event, 1)
	go func() { _ = Watch(ctx, path, func(event Event) { events <- event }) }()
	time.Sleep(100 * time.Millisecond)
	if err := os.Rename(path, filepath.Join(dir, "main.old.log")); err != nil {
		t.Fatal(err)
	}
	line := `[EdgeSync] §4.6 session_end_turn: sid=rotated-session status=completed stopReason=end_turn` + "\n"
	if err := os.WriteFile(path, []byte(line), 0o600); err != nil {
		t.Fatal(err)
	}
	select {
	case event := <-events:
		if event.SessionID != "rotated-session" {
			t.Fatalf("session id = %q, want rotated-session", event.SessionID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("watch did not read completion from the new log after rotation")
	}
}
