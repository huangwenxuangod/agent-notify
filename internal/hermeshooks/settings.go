package hermeshooks

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/hellolib/agent-notify/internal/common"
)

const hookMarker = "handle-hermes-hook"

// Install writes Hermes' documented Gateway Hook pair. Gateway hooks are the
// only official file-based event surface; they do not pretend to cover CLI
// turns, which require a Hermes plugin or configured shell hook.
func Install(dir, binary string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	binary = common.ResolveBinaryPath(binary)
	manifest := "name: agent-notify\n" +
		"description: Forward Hermes Gateway lifecycle events to Agent Notify\n" +
		"events:\n" +
		"  - agent:end\n" +
		"  - agent:start\n" +
		"  - agent:error\n" +
		"  - tool:error\n"
	if err := common.WriteFileAtomic(filepath.Join(dir, "HOOK.yaml"), []byte(manifest), 0o600); err != nil {
		return err
	}
	handler := "# agent-notify Hermes Gateway hook\n" +
		"# " + hookMarker + "\n" +
		"import json\n" +
		"import subprocess\n\n" +
		"async def handle(event_type, context):\n" +
		"    payload = {\"event\": event_type, **context}\n" +
		"    subprocess.run([" + pythonString(binary) + ", \"handle-hermes-hook\"], input=json.dumps(payload), text=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=10, check=False)\n"
	return common.WriteFileAtomic(filepath.Join(dir, "handler.py"), []byte(handler), 0o700)
}

func pythonString(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "\\'") + "'"
}

func IsInstalled(dir string) (bool, error) {
	data, err := os.ReadFile(filepath.Join(dir, "handler.py"))
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !strings.Contains(string(data), hookMarker) {
		return false, nil
	}
	manifest, err := os.ReadFile(filepath.Join(dir, "HOOK.yaml"))
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	for _, event := range []string{"agent:end", "agent:error", "tool:error"} {
		if !strings.Contains(string(manifest), event) {
			return false, nil
		}
	}
	return true, nil
}

func Uninstall(dir string) error {
	installed, err := IsInstalled(dir)
	if err != nil || !installed {
		return err
	}
	if err := os.Remove(filepath.Join(dir, "HOOK.yaml")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Remove(filepath.Join(dir, "handler.py"))
}
