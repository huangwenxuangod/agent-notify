package common

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hellolib/agent-notify/internal/testutil"
)

func TestHookBinaryPathPrefersManagedHostBinary(t *testing.T) {
	home := testutil.IsolateHome(t)
	candidate := filepath.Join(home, ".agent-notify", hookBinaryName())
	if err := os.MkdirAll(filepath.Dir(candidate), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(candidate, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := HookBinaryPath(); got != filepath.ToSlash(candidate) {
		t.Fatalf("HookBinaryPath() = %q, want %q", got, filepath.ToSlash(candidate))
	}
}
