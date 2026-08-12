package agentintegrations

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/hellolib/agent-notify/internal/common"
	"github.com/hellolib/agent-notify/internal/workbuddyhooks"
)

// WorkBuddyIntegration uses the CodeBuddy hook schema shared by WorkBuddy.
type WorkBuddyIntegration struct{}

func NewWorkBuddyIntegration() *WorkBuddyIntegration { return &WorkBuddyIntegration{} }
func (w *WorkBuddyIntegration) Name() string         { return "WorkBuddy / CodeBuddy" }
func (w *WorkBuddyIntegration) DetectInstalled() bool {
	if _, err := exec.LookPath("codebuddy"); err == nil {
		return true
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(home, ".codebuddy"))
	return err == nil
}
func (w *WorkBuddyIntegration) SettingsPath(scope string) (string, error) {
	switch scope {
	case "user":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".codebuddy", "settings.json"), nil
	case "project":
		return filepath.Join(".codebuddy", "settings.json"), nil
	default:
		return "", fmt.Errorf("unsupported scope: %s", scope)
	}
}
func (w *WorkBuddyIntegration) Install(path, binaryPath string) error {
	return workbuddyhooks.Install(path, common.ResolveBinaryPath(binaryPath))
}
func (w *WorkBuddyIntegration) Uninstall(path string) error { return workbuddyhooks.Uninstall(path) }
func (w *WorkBuddyIntegration) IsHookInstalled(path string) (bool, error) {
	return workbuddyhooks.IsInstalledWithBinary(path, common.HookBinaryPath())
}
