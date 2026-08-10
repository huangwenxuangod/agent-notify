package openclawhooks

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const marker = "handle-openclaw-hook"

func Install(dir, binary string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	manifest := map[string]any{"name": "agent-notify", "version": "1.0.0", "agent_notify": true, "hooks": []string{"gateway_start", "gateway_stop", "agent_end", "model_error", "tool_error", "approval_required"}}
	data, _ := json.MarshalIndent(manifest, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), append(data, '\n'), 0o600); err != nil {
		return err
	}
	plugin := "// agent-notify typed OpenClaw plugin\n// " + marker + "\n" + strings.TrimSpace(binary) + "\n"
	return os.WriteFile(filepath.Join(dir, "index.js"), []byte(plugin), 0o600)
}
func IsInstalled(dir string) (bool, error) {
	data, err := os.ReadFile(filepath.Join(dir, "index.js"))
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return strings.Contains(string(data), marker), nil
}
func Uninstall(dir string) error {
	if ok, err := IsInstalled(dir); err != nil {
		return err
	} else if !ok {
		return nil
	}
	_ = os.Remove(filepath.Join(dir, "manifest.json"))
	return os.Remove(filepath.Join(dir, "index.js"))
}
