package main

import (
	"path/filepath"
	"testing"
)

func TestAcquireDesktopInstanceAllowsOnlyOneOwner(t *testing.T) {
	path := filepath.Join(t.TempDir(), "desktop.lock")
	first := acquireDesktopInstance(path)
	if first == nil {
		t.Fatal("first desktop instance did not acquire lock")
	}
	defer first.Release()
	if second := acquireDesktopInstance(path); second != nil {
		second.Release()
		t.Fatal("second desktop instance acquired lock")
	}
}
