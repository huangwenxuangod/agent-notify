package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hellolib/agent-notify/internal/testutil"
)

func TestRunBridgeCreatesTokenAndStopsWithContext(t *testing.T) {
	home := testutil.IsolateHome(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var stdout, stderr bytes.Buffer
	done := make(chan error, 1)
	go func() {
		done <- runBridge(ctx, Streams{Stdout: &stdout, Stderr: &stderr}, 0)
	}()

	deadline := time.Now().Add(time.Second)
	for !strings.Contains(stdout.String(), "Bridge listening") && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if !strings.Contains(stdout.String(), "Bridge listening") {
		t.Fatalf("bridge did not report readiness: stdout=%q stderr=%q", stdout.String(), stderr.String())
	}

	info, err := os.Stat(filepath.Join(home, ".agent-notify", "bridge.token"))
	if err != nil {
		t.Fatalf("bridge token: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("token mode = %o, want 600", info.Mode().Perm())
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("runBridge returned %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("bridge did not stop after context cancellation")
	}
}
