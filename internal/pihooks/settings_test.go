package pihooks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallWritesUserExtensionIdempotently(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent-notify.ts")
	if err := Install(path, "/tmp/agent-notify"); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(first), "agent-notify-pi-extension") || !strings.Contains(string(first), "handle-pi-hook") {
		t.Fatalf("extension marker/handler missing: %s", first)
	}
	if err := Install(path, "/tmp/agent-notify"); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatal("reinstall should be idempotent")
	}
}

func TestUninstallRemovesOnlyManagedExtension(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent-notify.ts")
	if err := Install(path, "/tmp/agent-notify"); err != nil {
		t.Fatal(err)
	}
	if err := Uninstall(path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("managed extension still exists: %v", err)
	}
}

func TestExtensionForwardsAgentEndImmediatelyAndOnlyReportsActiveShutdown(t *testing.T) {
	source := BuildExtension("/tmp/agent-notify")
	for _, want := range []string{"pi.on(\"agent_start\"", "pi.on(\"agent_end\"", "activeRun"} {
		if !strings.Contains(source, want) {
			t.Fatalf("extension missing %q", want)
		}
	}
	for _, forbidden := range []string{"completionQuietMs", "setTimeout", "clearTimeout", "pendingEnd", "flushPendingEnd"} {
		if strings.Contains(source, forbidden) {
			t.Fatalf("extension must not delay agent_end with %q", forbidden)
		}
	}
	if strings.Contains(source, "agent_settled") || strings.Contains(source, "will_retry") {
		t.Fatal("extension must use the stable Pi event surface without unsupported retry fields")
	}
}
