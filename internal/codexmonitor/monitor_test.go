package codexmonitor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseFinalAnswer(t *testing.T) {
	line := `{"timestamp":"2026-08-12T02:07:42.142Z","type":"event_msg","payload":{"type":"agent_message","message":"Task complete","phase":"final_answer"}}`
	message, ok := ParseFinalAnswer(line)
	if !ok {
		t.Fatal("final answer was not detected")
	}
	if message != "Task complete" {
		t.Fatalf("message = %q", message)
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

func TestReadNewContinuesPastLargeJournalRecord(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rollout-large.jsonl")
	largeRecord := `{"type":"response_item","payload":{"output":"` + strings.Repeat("x", 128*1024) + `"}}` + "\n"
	finalAnswer := `{"type":"event_msg","payload":{"type":"agent_message","message":"Completed after large output","phase":"final_answer"}}` + "\n"
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
