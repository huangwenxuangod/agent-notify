package hermeshooks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallCreatesOfficialGatewayHook(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "agent-notify")
	if err := Install(dir, "/tmp/agent-notify"); err != nil {
		t.Fatalf("Install() error = %v", err)
	}

	manifest, err := os.ReadFile(filepath.Join(dir, "HOOK.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range []string{"agent:end", "agent:error", "tool:error"} {
		if !strings.Contains(string(manifest), event) {
			t.Fatalf("HOOK.yaml does not subscribe to %s:\n%s", event, manifest)
		}
	}
	handler, err := os.ReadFile(filepath.Join(dir, "handler.py"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(handler), "handle-hermes-hook") {
		t.Fatalf("handler.py does not invoke the managed handler:\n%s", handler)
	}
	installed, err := IsInstalled(dir)
	if err != nil || !installed {
		t.Fatalf("IsInstalled() = %v, %v; want true, nil", installed, err)
	}
}

func TestIsInstalledRejectsStaleGatewayHookManifest(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "agent-notify")
	if err := Install(dir, "/tmp/agent-notify"); err != nil {
		t.Fatal(err)
	}
	stale := "name: agent-notify\nevents:\n  - agent:start\n  - agent:end\n"
	if err := os.WriteFile(filepath.Join(dir, "HOOK.yaml"), []byte(stale), 0o600); err != nil {
		t.Fatal(err)
	}
	installed, err := IsInstalled(dir)
	if err != nil {
		t.Fatal(err)
	}
	if installed {
		t.Fatal("stale manifest must be repaired by the next automatic setup")
	}
}
