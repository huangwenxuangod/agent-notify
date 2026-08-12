package agentintegrations

import (
	"path/filepath"
	"testing"

	"github.com/hellolib/agent-notify/internal/common"
	"github.com/hellolib/agent-notify/internal/testutil"
)

func TestWorkBuddyIntegrationSettingsPath(t *testing.T) {
	restore := testutil.WithHome(t)
	defer restore()

	i := NewWorkBuddyIntegration()
	path, err := i.SettingsPath("user")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(testutil.Home(t), ".codebuddy", "settings.json"); path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}

	project, err := i.SettingsPath("project")
	if err != nil {
		t.Fatal(err)
	}
	if project != filepath.Join(".codebuddy", "settings.json") {
		t.Fatalf("project = %q", project)
	}
}

func TestWorkBuddyIntegrationInstallsManagedHooks(t *testing.T) {
	i := NewWorkBuddyIntegration()
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := i.Install(path, common.HookBinaryPath()); err != nil {
		t.Fatal(err)
	}
	installed, err := i.IsHookInstalled(path)
	if err != nil {
		t.Fatal(err)
	}
	if !installed {
		t.Fatal("WorkBuddy hooks were not installed")
	}
}
