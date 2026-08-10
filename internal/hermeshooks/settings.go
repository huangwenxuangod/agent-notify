package hermeshooks

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"
)

const hookMarker = "handle-hermes-hook"

var events = []string{"agent:end", "agent:error", "approval:required", "session:start"}

func Install(path, binary string) error {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		data = []byte("{}\n")
	} else if err != nil {
		return err
	}
	var root map[string]any
	if err := yaml.Unmarshal(data, &root); err != nil {
		return err
	}
	if root == nil {
		root = map[string]any{}
	}
	hooks, _ := root["hooks"].(map[string]any)
	if hooks == nil {
		hooks = map[string]any{}
	}
	command := strings.TrimSpace(binary) + " " + hookMarker
	for _, event := range events {
		hooks[event] = map[string]any{"command": command, "agent_notify": true}
	}
	root["hooks"] = hooks
	out, err := yaml.Marshal(root)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o600)
}

func IsInstalled(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return strings.Contains(string(data), hookMarker), nil
}

func Uninstall(path string) error {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var root map[string]any
	if err := yaml.Unmarshal(data, &root); err != nil {
		return err
	}
	if hooks, ok := root["hooks"].(map[string]any); ok {
		for k, v := range hooks {
			if strings.Contains(fmt.Sprint(v), hookMarker) {
				delete(hooks, k)
			}
		}
		if len(hooks) == 0 {
			delete(root, "hooks")
		}
	}
	out, err := yaml.Marshal(root)
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o600)
}
