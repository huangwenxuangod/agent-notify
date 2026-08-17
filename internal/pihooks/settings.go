package pihooks

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/hellolib/agent-notify/internal/common"
)

const marker = "agent-notify-pi-extension"

func Install(path, binary string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return common.WriteFileAtomic(path, []byte(extensionSource(binary)), 0o600)
}

func IsInstalled(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return strings.Contains(string(data), marker) && strings.Contains(string(data), "handle-pi-hook"), nil
}

func Uninstall(path string) error {
	installed, err := IsInstalled(path)
	if err != nil || !installed {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
