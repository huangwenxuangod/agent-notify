package main

import (
	"path/filepath"
	"time"

	"github.com/hellolib/agent-notify/internal/common"
)

// acquireDesktopInstance prevents a second tray/UI process from subscribing
// to the same agent journals. The kernel releases the lock if the app exits.
func acquireDesktopInstance(path string) *common.FileLock {
	return common.AcquireFileLock(path, 100*time.Millisecond)
}

func desktopInstanceLockPath(statePath string) string {
	return filepath.Join(filepath.Dir(statePath), "desktop.lock")
}
