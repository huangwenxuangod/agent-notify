package workbuddymonitor

import "testing"

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
	if event.Body != "Research plan" {
		t.Fatalf("body = %q, want conversation title", event.Body)
	}
}

func TestParserIgnoresDuplicateAndIncompleteSessionEvents(t *testing.T) {
	p := NewParser()
	if _, ok := p.Consume(`[INFO] [INFLIGHT_CLEAR] sid=session-1 reason=session_cancelled`); ok {
		t.Fatal("non-complete session emitted an event")
	}
	if _, ok := p.Consume(`[INFO] [INFLIGHT_CLEAR] sid=session-1 reason=session_end_turn`); !ok {
		t.Fatal("first completion must emit")
	}
	if _, ok := p.Consume(`[INFO] [INFLIGHT_CLEAR] sid=session-1 reason=session_end_turn`); ok {
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
	if event.Body != "UI conversation" || event.Workspace != "/Users/ai1/project" {
		t.Fatalf("event = %#v", event)
	}
}
