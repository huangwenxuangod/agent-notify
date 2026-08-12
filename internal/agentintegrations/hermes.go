package agentintegrations

import (
	"fmt"
	"github.com/hellolib/agent-notify/internal/hermeshooks"
	"os"
	"os/exec"
	"path/filepath"
)

type HermesIntegration struct{}

func NewHermesIntegration() *HermesIntegration { return &HermesIntegration{} }
func (h *HermesIntegration) Name() string      { return "Hermes Agent" }
func (h *HermesIntegration) DetectInstalled() bool {
	if _, err := exec.LookPath("hermes"); err == nil {
		return true
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(home, ".hermes"))
	return err == nil
}
func (h *HermesIntegration) SettingsPath(scope string) (string, error) {
	switch scope {
	case "user":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".hermes", "hooks", "agent-notify"), nil
	case "project":
		return filepath.Join(".hermes", "hooks", "agent-notify"), nil
	default:
		return "", fmt.Errorf("unsupported scope: %s", scope)
	}
}
func (h *HermesIntegration) Install(path, binary string) error {
	return hermeshooks.Install(path, binary)
}
func (h *HermesIntegration) Uninstall(path string) error { return hermeshooks.Uninstall(path) }
func (h *HermesIntegration) IsHookInstalled(path string) (bool, error) {
	return hermeshooks.IsInstalled(path)
}
