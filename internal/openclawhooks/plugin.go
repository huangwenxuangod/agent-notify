package openclawhooks

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hellolib/agent-notify/internal/common"
)

const marker = "handle-openclaw-hook"

// Install writes a self-contained OpenClaw ESM plugin package. The OpenClaw
// operator still enables it and explicitly grants conversation access; those
// are host policy decisions and cannot be safely bypassed by this installer.
func Install(dir, binary string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	binary = common.ResolveBinaryPath(binary)
	packageJSON := `{
  "name": "agent-notify-openclaw",
  "version": "1.0.0",
  "private": true,
  "type": "module",
  "peerDependencies": { "openclaw": ">=2026.3.24-beta.2" },
  "openclaw": {
    "extensions": ["./index.js"],
    "runtimeExtensions": ["./index.js"],
    "compat": {
      "pluginApi": ">=2026.3.24-beta.2",
      "minGatewayVersion": "2026.3.24-beta.2"
    },
    "build": {
      "openclawVersion": "2026.3.24-beta.2",
      "pluginSdkVersion": "2026.3.24-beta.2"
    }
  }
}
`
	manifest := `{
  "id": "agent-notify",
  "name": "Agent Notify",
  "description": "Forward OpenClaw lifecycle events to Agent Notify",
  "activation": { "onStartup": true },
  "configSchema": { "type": "object", "additionalProperties": false }
}
`
	entry := "// agent-notify OpenClaw plugin\n" +
		"// " + marker + "\n" +
		"import { spawn } from \"node:child_process\";\n" +
		"import { definePluginEntry } from \"openclaw/plugin-sdk/plugin-entry\";\n\n" +
		"const binary = " + jsString(binary) + ";\n\n" +
		"function forward(event, context) {\n" +
		"  const success = event.success !== false;\n" +
		"  const messages = Array.isArray(event.messages) ? event.messages : [];\n" +
		"  const lastMessage = messages.length === 0 ? \"\" : String(messages[messages.length - 1] ?? \"\");\n" +
		"  const child = spawn(binary, [\"handle-openclaw-hook\"], { detached: true, stdio: [\"pipe\", \"ignore\", \"ignore\"] });\n" +
		"  child.on(\"error\", () => {});\n" +
		"  child.stdin.end(JSON.stringify({ event: success ? \"agent_end\" : \"agent_error\", session_id: context.sessionId ?? context.sessionKey ?? \"\", run_id: event.runId ?? context.runId ?? \"\", workspace: context.workspaceDir ?? \"\", message: lastMessage, error: event.error?.message ?? \"\" }));\n" +
		"  child.unref();\n" +
		"}\n\n" +
		"export default definePluginEntry({\n" +
		"  id: \"agent-notify\", name: \"Agent Notify\", description: \"Forward lifecycle events to Agent Notify\",\n" +
		"  register(api) {\n" +
		"    api.on(\"agent_end\", (event, context) => forward(event, context));\n" +
		"  },\n" +
		"});\n"
	for name, data := range map[string]string{
		"package.json":         packageJSON,
		"openclaw.plugin.json": manifest,
		"index.js":             entry,
	} {
		if err := common.WriteFileAtomic(filepath.Join(dir, name), []byte(data), 0o600); err != nil {
			return err
		}
	}
	return nil
}

func jsString(value string) string { return fmt.Sprintf("%q", value) }

func IsInstalled(dir string) (bool, error) {
	data, err := os.ReadFile(filepath.Join(dir, "index.js"))
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !strings.Contains(string(data), marker) {
		return false, nil
	}
	for _, name := range []string{"package.json", "openclaw.plugin.json"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			return false, err
		}
	}
	return true, nil
}

func Uninstall(dir string) error {
	installed, err := IsInstalled(dir)
	if err != nil || !installed {
		return err
	}
	for _, name := range []string{"package.json", "openclaw.plugin.json", "index.js"} {
		if err := os.Remove(filepath.Join(dir, name)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}
