package workbuddyhooks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallPreservesExistingHooks(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".codebuddy", "settings.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"model":"keep","hooks":{"Stop":[{"hooks":[{"type":"command","command":"user-command"}]}]}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Install(path, "/opt/agent-notify"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got["model"] != "keep" {
		t.Fatalf("model was not preserved: %s", data)
	}
	if !strings.Contains(string(data), "user-command") || !strings.Contains(string(data), "handle-workbuddy-hook") {
		t.Fatalf("hooks were not merged: %s", data)
	}
}

func TestBuildHookSettingsUsesCodeBuddyEvents(t *testing.T) {
	settings := BuildHookSettings("/opt/agent-notify")
	hooks, ok := settings["hooks"].(map[string]any)
	if !ok {
		t.Fatalf("hooks = %#v", settings["hooks"])
	}
	for _, event := range managedEvents {
		if _, ok := hooks[event]; !ok {
			t.Fatalf("missing managed event %q", event)
		}
	}
}

