package openclawhooks

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallCreatesOfficialPluginPackage(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "agent-notify")
	if err := Install(dir, "/tmp/agent-notify"); err != nil {
		t.Fatalf("Install() error = %v", err)
	}

	for _, name := range []string{"package.json", "openclaw.plugin.json", "index.js"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("%s missing: %v", name, err)
		}
	}
	entry, err := os.ReadFile(filepath.Join(dir, "index.js"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"definePluginEntry", "agent_end", "handle-openclaw-hook"} {
		if !strings.Contains(string(entry), want) {
			t.Fatalf("plugin entry missing %q:\n%s", want, entry)
		}
	}
	manifest, err := os.ReadFile(filepath.Join(dir, "openclaw.plugin.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(manifest), `"id": "agent-notify"`) {
		t.Fatalf("plugin manifest missing plugin id:\n%s", manifest)
	}
	packageJSON, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"runtimeExtensions", "pluginApi", "minGatewayVersion"} {
		if !strings.Contains(string(packageJSON), want) {
			t.Fatalf("package metadata missing %q:\n%s", want, packageJSON)
		}
	}
	for _, want := range []string{"event.success", "event.messages", "context.runId"} {
		if !strings.Contains(string(entry), want) {
			t.Fatalf("plugin entry missing official agent_end field %q:\n%s", want, entry)
		}
	}
	for _, raw := range [][]byte{packageJSON, manifest} {
		var value map[string]any
		if err := json.Unmarshal(raw, &value); err != nil {
			t.Fatalf("generated JSON is invalid: %v\n%s", err, raw)
		}
	}
	if node, err := exec.LookPath("node"); err == nil {
		if out, err := exec.Command(node, "--check", filepath.Join(dir, "index.js")).CombinedOutput(); err != nil {
			t.Fatalf("generated plugin has invalid JavaScript: %v\n%s", err, out)
		}
	}
}
