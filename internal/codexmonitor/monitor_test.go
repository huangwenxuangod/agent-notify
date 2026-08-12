package codexmonitor

import "testing"

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
