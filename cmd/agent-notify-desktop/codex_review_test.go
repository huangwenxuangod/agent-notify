package main

import (
	"strings"
	"testing"
)

func TestOpenCodexHookReviewLaunchesTerminalWithCodexInstructions(t *testing.T) {
	original := runCodexHookReview
	t.Cleanup(func() { runCodexHookReview = original })
	var gotName string
	var gotArgs []string
	runCodexHookReview = func(name string, args ...string) error {
		gotName = name
		gotArgs = args
		return nil
	}

	if err := openCodexHookReview(); err != nil {
		t.Fatalf("openCodexHookReview() error = %v", err)
	}
	if gotName != "osascript" {
		t.Fatalf("command = %q, want osascript", gotName)
	}
	if len(gotArgs) != 2 || gotArgs[0] != "-e" {
		t.Fatalf("args = %#v, want osascript script", gotArgs)
	}
	if !strings.Contains(gotArgs[1], "codex") || !strings.Contains(gotArgs[1], "/hooks") {
		t.Fatalf("script = %q, want Codex and /hooks guidance", gotArgs[1])
	}
}

func TestOpenWorkBuddyHookReviewLaunchesTerminalWithCodeBuddyInstructions(t *testing.T) {
	original := runWorkBuddyHookReview
	t.Cleanup(func() { runWorkBuddyHookReview = original })
	var gotName string
	var gotArgs []string
	runWorkBuddyHookReview = func(name string, args ...string) error {
		gotName = name
		gotArgs = args
		return nil
	}

	if err := openWorkBuddyHookReview(); err != nil {
		t.Fatalf("openWorkBuddyHookReview() error = %v", err)
	}
	if gotName != "osascript" {
		t.Fatalf("command = %q, want osascript", gotName)
	}
	if len(gotArgs) != 2 || gotArgs[0] != "-e" {
		t.Fatalf("args = %#v, want osascript script", gotArgs)
	}
	if !strings.Contains(gotArgs[1], "codebuddy") || !strings.Contains(gotArgs[1], "/hooks") {
		t.Fatalf("script = %q, want CodeBuddy and /hooks guidance", gotArgs[1])
	}
}
