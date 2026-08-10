package bridge_test

import (
	"path/filepath"
	"testing"

	"github.com/hellolib/agent-notify/internal/bridge"
)

func TestServiceCanBeCreatedWithIsolatedState(t *testing.T) {
	svc, err := bridge.NewService(bridge.Options{
		ConfigPath: filepath.Join(t.TempDir(), "config.yaml"),
		StatePath:  filepath.Join(t.TempDir(), "state.json"),
		LogPath:    filepath.Join(t.TempDir(), "agent-notify.log"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if svc == nil {
		t.Fatal("NewService returned nil")
	}
}
