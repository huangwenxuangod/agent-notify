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
