package main

import (
	"testing"

	"github.com/hellolib/agent-notify/internal/config"
)

func TestShouldHideWindowOnCloseDefaultsTrueAndHonorsSetting(t *testing.T) {
	if !shouldHideWindowOnClose(config.Config{}) {
		t.Fatal("legacy configuration should keep hiding the window")
	}
	disabled := false
	if shouldHideWindowOnClose(config.Config{Behavior: config.BehaviorConfig{HideWindowOnClose: &disabled}}) {
		t.Fatal("explicit false should allow the app to close")
	}
}

func TestShouldPreventWindowCloseAllowsQuitAndSystemShutdown(t *testing.T) {
	if !shouldPreventWindowClose(config.Config{}, false, false) {
		t.Fatal("normal close should hide the window by default")
	}
	if shouldPreventWindowClose(config.Config{}, true, false) {
		t.Fatal("explicit app quit must not be intercepted")
	}
	if shouldPreventWindowClose(config.Config{}, false, true) {
		t.Fatal("system shutdown must not be intercepted")
	}
}
