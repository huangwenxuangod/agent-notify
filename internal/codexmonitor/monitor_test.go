package codexmonitor

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseFinalAnswerIgnoresIntermediateAgentMessage(t *testing.T) {
	line := `{"timestamp":"2026-08-12T02:07:42.142Z","type":"event_msg","payload":{"type":"agent_message","message":"Task complete","phase":"final_answer"}}`
	if _, ok := ParseFinalAnswer(line); ok {
		t.Fatal("intermediate agent_message must not produce a completion")
	}
}

func TestParseFinalAnswerIgnoresCommentary(t *testing.T) {
	line := `{"type":"event_msg","payload":{"type":"agent_message","message":"Working","phase":"commentary"}}`
	if _, ok := ParseFinalAnswer(line); ok {
		t.Fatal("commentary must not produce a completion")
	}
}

func TestParseJournalEventDetectsCodexTaskError(t *testing.T) {
	line := `{"timestamp":"2026-08-13T09:58:08.797Z","type":"event_msg","payload":{"type":"task_complete"},"error":{"message":"exceeded retry limit, last status: 429 Too Many Requests","codex_error_info":{"response_too_many_failed_attempts":{"http_status_code":429}}}}`
	event, ok := ParseJournalEvent(line)
	if !ok {
		t.Fatal("Codex task error was not detected")
	}
	if event.Event != "run_failed" {
		t.Fatalf("event = %q, want run_failed", event.Event)
	}
	if event.Body != "exceeded retry limit, last status: 429 Too Many Requests" {
		t.Fatalf("body = %q", event.Body)
	}
}

func TestParseJournalEventDetectsCodexTaskComplete(t *testing.T) {
	line := `{"type":"event_msg","payload":{"type":"task_complete","last_agent_message":"Task complete"}}`
	event, ok := ParseJournalEvent(line)
	if !ok {
		t.Fatal("Codex task completion was not detected")
	}
	if event.Event != "run_completed" || event.Body != "Task complete" {
		t.Fatalf("event = %#v, want completed task", event)
	}
}

func TestReadNewContinuesPastLargeJournalRecord(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rollout-large.jsonl")
	largeRecord := `{"type":"response_item","payload":{"output":"` + strings.Repeat("x", 128*1024) + `"}}` + "\n"
	finalAnswer := `{"type":"event_msg","payload":{"type":"task_complete","last_agent_message":"Completed after large output"}}` + "\n"
	if err := os.WriteFile(path, []byte(largeRecord+finalAnswer), 0o600); err != nil {
		t.Fatal(err)
	}

	var messages []string
	next, err := readNew(path, 0, func(event Event) { messages = append(messages, event.Body) })
	if err != nil {
		t.Fatalf("readNew() error = %v", err)
	}
	if next != int64(len(largeRecord)+len(finalAnswer)) {
		t.Fatalf("next = %d, want %d", next, len(largeRecord)+len(finalAnswer))
	}
	if len(messages) != 1 || messages[0] != "Completed after large output" {
		t.Fatalf("messages = %#v", messages)
	}
}

func TestReadNewSkipsExistingJournalContentAfterTruncation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rollout-truncated.jsonl")
	line := `{"type":"event_msg","payload":{"type":"task_complete","turn_id":"turn-1","last_agent_message":"historical task"}}` + "\n"
	if err := os.WriteFile(path, []byte(line), 0o600); err != nil {
		t.Fatal(err)
	}

	var events []Event
	next, err := readNew(path, int64(len(line)+1), func(event Event) { events = append(events, event) })
	if err != nil {
		t.Fatalf("readNew() error = %v", err)
	}
	if next != int64(len(line)) {
		t.Fatalf("next = %d, want %d", next, len(line))
	}
	if len(events) != 0 {
		t.Fatalf("events = %#v, want no replay after truncation", events)
	}
}

func TestWatchEmitsEachCodexTurnOnlyOnce(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "rollout-turns.jsonl")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events := make(chan Event, 2)
	go func() { _ = Watch(ctx, root, func(event Event) { events <- event }) }()
	time.Sleep(100 * time.Millisecond)
	line := `{"type":"event_msg","payload":{"type":"task_complete","turn_id":"turn-1","last_agent_message":"done"}}` + "\n"
	if err := os.WriteFile(path, []byte(line+line), 0o600); err != nil {
		t.Fatal(err)
	}
	select {
	case <-events:
	case <-time.After(2 * time.Second):
		t.Fatal("watch did not emit the first terminal event")
	}
	select {
	case duplicate := <-events:
		t.Fatalf("duplicate terminal event = %#v", duplicate)
	case <-time.After(time.Second):
	}
}

func TestSessionIDNormalizesRolloutFilenameToCodexSessionID(t *testing.T) {
	path := "/tmp/rollout-2026-08-06T13-58-53-019fd5a7-3881-7d71-9c98-5b3b165e97ca.jsonl"
	if got, want := sessionID(path), "019fd5a7-3881-7d71-9c98-5b3b165e97ca"; got != want {
		t.Fatalf("sessionID() = %q, want %q", got, want)
	}
}

func TestWatchContinuesAfterCodexSessionFileReplacement(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "rollout-019fe961-e541-7252-a5ef-07fb9c2f8f63.jsonl")
	if err := os.WriteFile(path, []byte("old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events := make(chan Event, 1)
	go func() { _ = Watch(ctx, root, func(event Event) { events <- event }) }()
	time.Sleep(100 * time.Millisecond)
	if err := os.Rename(path, path+".old"); err != nil {
		t.Fatal(err)
	}
	line := `{"type":"event_msg","payload":{"type":"task_complete","turn_id":"turn-replaced","last_agent_message":"after replacement"}}` + "\n"
	if err := os.WriteFile(path, []byte(line), 0o600); err != nil {
		t.Fatal(err)
	}
	select {
	case event := <-events:
		if event.Body != "after replacement" {
			t.Fatalf("body = %q", event.Body)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("watch did not read replacement session file")
	}
}
