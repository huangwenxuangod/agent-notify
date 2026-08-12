package main

import (
	"context"
	"reflect"
	"testing"

	"github.com/hellolib/agent-notify/internal/bridge"
	"github.com/hellolib/agent-notify/internal/config"
)

func TestAutoSetupTargetsOnlyDetectedUninstalledAgents(t *testing.T) {
	got := autoSetupTargets([]bridge.AgentStatus{
		{ID: "codex", Installed: true, HookInstalled: true},
		{ID: "workbuddy", Installed: true},
		{ID: "hermes"},
	})
	if want := []string{"workbuddy"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("autoSetupTargets() = %v, want %v", got, want)
	}
}

func TestDesktopStartsHidden(t *testing.T) {
	if !desktopOptions(nil, nil).StartHidden {
		t.Fatal("desktop app must start hidden in the menu bar")
	}
}

func TestDesktopShowArgumentMakesWindowVisible(t *testing.T) {
	if desktopStartsHidden([]string{"agent-notify-desktop", "--show"}) {
		t.Fatal("--show must make the desktop window visible")
	}
	if !desktopStartsHidden([]string{"agent-notify-desktop", "tray"}) {
		t.Fatal("background tray startup must remain hidden")
	}
}

func TestDesktopUsesSeparateHookAndAutostartBinaries(t *testing.T) {
	options := desktopServiceOptions("/opt/agent-notify", "/Applications/Agent Notify.app/Contents/MacOS/Agent Notify")
	if options.BinaryPath != "/opt/agent-notify" {
		t.Fatalf("hook binary = %q", options.BinaryPath)
	}
	if options.AutostartPath != "/Applications/Agent Notify.app/Contents/MacOS/Agent Notify" {
		t.Fatalf("autostart binary = %q", options.AutostartPath)
	}
}

func TestSyncBridgeConfigUsesHostConfiguration(t *testing.T) {
	original := saveBridgeConfig
	t.Cleanup(func() { saveBridgeConfig = original })
	var got config.Config
	saveBridgeConfig = func(_ context.Context, _, _ string, value config.Config) error {
		got = value
		return nil
	}
	cfg := config.Default()
	cfg.Behavior.DedupeSeconds = 42
	(&App{}).syncBridgeConfig(cfg)
	if got.Behavior.DedupeSeconds != 42 {
		t.Fatalf("synced dedupe=%d, want 42", got.Behavior.DedupeSeconds)
	}
}

func TestHookReviewTargetsOnlyIncludeNewlyInstalledAgentsWithNativeReview(t *testing.T) {
	got := hookReviewTargets([]string{"codex", "workbuddy", "hermes", "codex"})
	if want := []string{"codex", "workbuddy"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("hookReviewTargets() = %v, want %v", got, want)
	}
}

func TestEnableSystemNotificationsOnlyChangesSelectedAgents(t *testing.T) {
	cfg := config.Default()
	enableSystemNotifications(&cfg, []string{"codex", "workbuddy"})
	if !cfg.Notify.Codex.Channels.System.Enabled || !cfg.Notify.WorkBuddy.Channels.System.Enabled {
		t.Fatal("selected agents must have system notifications enabled")
	}
	if cfg.Notify.ClaudeCode.Channels.System.Enabled {
		t.Fatal("unselected agents must remain unchanged")
	}
}

func TestSetClickToFocusOnlyChangesSelectedAgents(t *testing.T) {
	cfg := config.Default()
	cfg.Notify.Codex.Channels.System.ClickToFocus = true
	cfg.Notify.WorkBuddy.Channels.System.ClickToFocus = true

	if !setClickToFocus(&cfg, []string{"codex", "workbuddy"}, false) {
		t.Fatal("setClickToFocus() = false, want changed")
	}
	if cfg.Notify.Codex.Channels.System.ClickToFocus || cfg.Notify.WorkBuddy.Channels.System.ClickToFocus {
		t.Fatal("selected agents must have click-to-focus disabled")
	}
	if !cfg.Notify.ClaudeCode.Channels.System.ClickToFocus {
		t.Fatal("unselected agent must remain unchanged")
	}
}

func TestShouldImportRemoteConfigOnlyWhenHostHasNoConfiguredChannel(t *testing.T) {
	host := config.Default()
	bridge := config.Default()
	bridge.Remote.Feishu.Enabled = true
	bridge.Remote.Feishu.WebhookURL = "https://example.test/feishu"
	if !shouldImportRemoteConfig(host, bridge) {
		t.Fatal("empty host config should import configured bridge remote channels")
	}
	host.Remote.Ntfy.Enabled = true
	host.Remote.Ntfy.TopicURL = "https://ntfy.sh/already-local"
	if shouldImportRemoteConfig(host, bridge) {
		t.Fatal("existing host remote channels must remain authoritative")
	}
}
