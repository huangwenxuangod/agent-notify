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
	if !strings.Contains(string(manifest), "agent:end") {
		t.Fatalf("HOOK.yaml does not subscribe to agent:end:\n%s", manifest)
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
