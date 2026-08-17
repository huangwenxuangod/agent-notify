package pihooks

import (
	"strings"
	"testing"
)

func TestParseMessageMapsFinalAgentEndCompletion(t *testing.T) {
	msg, err := ParseMessage(strings.NewReader(`{"event":"agent_end","session_id":"s1","run_id":"r1","cwd":"/workspace","message":"完成了"}`))
	if err != nil {
		t.Fatal(err)
	}
	if msg.Agent != "pi" || msg.Event != "run_completed" || msg.SessionID != "s1" || msg.RunID != "r1" {
		t.Fatalf("unexpected message: %#v", msg)
	}
	if msg.Workspace != "/workspace" || msg.Body != "完成了" {
		t.Fatalf("unexpected context: %#v", msg)
	}
}

func TestParseMessageMapsOnlyQuitShutdownToFailure(t *testing.T) {
	msg, err := ParseMessage(strings.NewReader(`{"event":"session_shutdown","reason":"quit","session_id":"s2","cwd":"/repo","error":"用户中断"}`))
	if err != nil {
		t.Fatal(err)
	}
	if msg.Event != "run_failed" || msg.Body != "用户中断" {
		t.Fatalf("unexpected shutdown message: %#v", msg)
	}
	if _, err := ParseMessage(strings.NewReader(`{"event":"session_shutdown","reason":"new","session_id":"s2"}`)); err == nil {
		t.Fatal("session switch shutdown should be ignored")
	}
}

func TestParseMessageMapsAgentEndErrorToFailure(t *testing.T) {
	msg, err := ParseMessage(strings.NewReader(`{"event":"agent_end","session_id":"s3","error":"rate limit"}`))
	if err != nil {
		t.Fatal(err)
	}
	if msg.Event != "run_failed" || msg.Body != "rate limit" {
		t.Fatalf("unexpected failure message: %#v", msg)
	}
}

func TestParseMessageRejectsUnsupportedEvents(t *testing.T) {
	if _, err := ParseMessage(strings.NewReader(`{"event":"agent_settled","session_id":"s4"}`)); err == nil {
		t.Fatal("unsupported Pi event must be rejected")
	}
}
