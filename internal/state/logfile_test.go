package state

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAppendLog(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent-notify.log")

	if err := AppendLog(path, "first line"); err != nil {
		t.Fatalf("AppendLog() error = %v", err)
	}
	if err := AppendLog(path, "second line"); err != nil {
		t.Fatalf("AppendLog() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	got := string(data)
	if !strings.Contains(got, "first line") || !strings.Contains(got, "second line") {
		t.Fatalf("log content = %q, want both lines", got)
	}
}

func TestAppendLogRotatesAfterThreeDays(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent-notify.log")
	then := time.Date(2026, time.August, 1, 12, 0, 0, 0, time.UTC)
	if err := appendLogAt(path, "old line", then); err != nil {
		t.Fatalf("append old log: %v", err)
	}
	if err := appendLogAt(path, "new line", then.Add(72*time.Hour)); err != nil {
		t.Fatalf("append rotated log: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(data); got != "new line\n" {
		t.Fatalf("log after rotation = %q, want only new line", got)
	}
}
