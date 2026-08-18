//go:build windows

package autostart

import "testing"

func TestStartupCommandQuotesWindowsPath(t *testing.T) {
	got := startupCommand(`C:/Users/test/Agent Notify.exe`)
	want := `"C:/Users/test/Agent Notify.exe" tray`
	if got != want {
		t.Fatalf("startupCommand() = %q, want %q", got, want)
	}
}
