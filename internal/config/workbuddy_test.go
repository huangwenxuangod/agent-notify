package config

import "testing"

func TestDefaultIncludesWorkBuddyWithSupportedEvents(t *testing.T) {
	cfg := Default()
	if cfg.Agent.WorkBuddy.Enabled {
		t.Fatal("WorkBuddy should start disabled")
	}
	for _, event := range []string{"permission_required", "input_required", "run_completed", "run_failed"} {
		if !containsEvent(cfg.Notify.WorkBuddy.Events, event) {
			t.Fatalf("missing WorkBuddy event %q", event)
		}
	}
}

func containsEvent(events []string, want string) bool {
	for _, event := range events {
		if event == want {
			return true
		}
	}
	return false
}
