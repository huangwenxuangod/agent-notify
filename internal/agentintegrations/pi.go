package agentintegrations

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/hellolib/agent-notify/internal/common"
	"github.com/hellolib/agent-notify/internal/pihooks"
)

// PiIntegration installs the official TypeScript Extension discovered by Pi.
type PiIntegration struct{}

func NewPiIntegration() *PiIntegration { return &PiIntegration{} }
func (p *PiIntegration) Name() string  { return "Pi" }

func (p *PiIntegration) DetectInstalled() bool {
	if _, err := exec.LookPath("pi"); err == nil {
		return true
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(home, ".pi", "agent"))
	return err == nil
}

func (p *PiIntegration) SettingsPath(scope string) (string, error) {
	switch scope {
	case "user":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".pi", "agent", "extensions", "agent-notify.ts"), nil
	case "project":
		return filepath.Join(".pi", "extensions", "agent-notify.ts"), nil
	default:
		return "", fmt.Errorf("unsupported scope: %s", scope)
	}
}

func (p *PiIntegration) Install(path, binary string) error {
	return pihooks.Install(path, common.ResolveBinaryPath(binary))
}

func (p *PiIntegration) Uninstall(path string) error { return pihooks.Uninstall(path) }

func (p *PiIntegration) IsHookInstalled(path string) (bool, error) {
	return pihooks.IsInstalled(path)
}
