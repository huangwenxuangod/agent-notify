package agentintegrations

import (
	"fmt"
	"github.com/hellolib/agent-notify/internal/openclawhooks"
	"os"
	"os/exec"
	"path/filepath"
)

type OpenClawIntegration struct{}

func NewOpenClawIntegration() *OpenClawIntegration { return &OpenClawIntegration{} }
func (o *OpenClawIntegration) Name() string        { return "OpenClaw" }
func (o *OpenClawIntegration) DetectInstalled() bool {
	if _, err := exec.LookPath("openclaw"); err == nil {
		return true
	}
	h, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(h, ".openclaw"))
	return err == nil
}
func (o *OpenClawIntegration) SettingsPath(scope string) (string, error) {
	if scope != "user" && scope != "project" {
		return "", fmt.Errorf("unsupported scope: %s", scope)
	}
	if scope == "project" {
		return filepath.Join(".openclaw", "extensions", "agent-notify"), nil
	}
	h, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(h, ".openclaw", "extensions", "agent-notify"), nil
}
func (o *OpenClawIntegration) Install(path, binary string) error {
	return openclawhooks.Install(path, binary)
}
func (o *OpenClawIntegration) Uninstall(path string) error { return openclawhooks.Uninstall(path) }
func (o *OpenClawIntegration) IsHookInstalled(path string) (bool, error) {
	return openclawhooks.IsInstalled(path)
}
