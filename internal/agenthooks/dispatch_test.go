package agenthooks

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hellolib/agent-notify/internal/config"
	"github.com/hellolib/agent-notify/internal/notify"
	"github.com/hellolib/agent-notify/internal/state"
)

func TestApplyFocusCacheRoutesByPlatform(t *testing.T) {
	cases := []struct {
		goos        string
		wantWindow  string
		wantCapture string
	}{
		{"linux", "123", ""},
		{"darwin", "", "123"},
		{"windows", "", "123"},
		{"freebsd", "", ""},
	}
	for _, c := range cases {
		msg := notify.Message{}
		applyFocusCache(&msg, c.goos, "123")
		if msg.FocusWindowID != c.wantWindow || msg.FocusCapture != c.wantCapture {
			t.Fatalf("%s: got FocusWindowID=%q FocusCapture=%q, want %q/%q",
				c.goos, msg.FocusWindowID, msg.FocusCapture, c.wantWindow, c.wantCapture)
		}
	}
}

func TestDispatchAppendsNonSessionEventToJournal(t *testing.T) {
	t.Setenv("AGENT_NOTIFY_BRIDGE_URL", "")
	t.Setenv("AGENT_NOTIFY_BRIDGE_TOKEN", "")
	t.Setenv("AGENT_NOTIFY_BRIDGE_TOKEN_FILE", filepath.Join(t.TempDir(), "missing-token"))
	root := t.TempDir()
	statePath := filepath.Join(root, "state.json")
	logPath := filepath.Join(root, "agent-notify.log")
	cfg := config.Default()
	if err := Dispatch(context.Background(), cfg, statePath, logPath, notify.Message{Agent: "codex", Event: "run_completed", SessionID: "s1", Body: "done", Origin: "desktop_monitor"}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(state.EventJournalPath(statePath))
	if err != nil {
		t.Fatal(err)
	}
	var record state.EventRecord
	if err := json.Unmarshal(data[:len(data)-1], &record); err != nil {
		t.Fatal(err)
	}
	if record.Agent != "codex" || record.Event != "run_completed" || record.Origin != "desktop_monitor" || record.Result != "no_sender" {
		t.Fatalf("record = %#v", record)
	}
}

func TestDispatchSuppressesWorkBuddyMonitorAfterNativeHook(t *testing.T) {
	t.Setenv("AGENT_NOTIFY_BRIDGE_URL", "")
	t.Setenv("AGENT_NOTIFY_BRIDGE_TOKEN", "")
	t.Setenv("AGENT_NOTIFY_BRIDGE_TOKEN_FILE", filepath.Join(t.TempDir(), "missing-token"))
	root := t.TempDir()
	statePath := filepath.Join(root, "state.json")
	logPath := filepath.Join(root, "agent-notify.log")
	cfg := config.Default()
	native := notify.Message{Agent: "workbuddy", Event: "run_completed", SessionID: "s1", Body: "native", Origin: "native_hook"}
	monitor := notify.Message{Agent: "workbuddy", Event: "run_completed", SessionID: "s1", Body: "monitor", Origin: "desktop_monitor"}
	if err := Dispatch(context.Background(), cfg, statePath, logPath, native); err != nil {
		t.Fatal(err)
	}
	if err := Dispatch(context.Background(), cfg, statePath, logPath, monitor); err != nil {
		t.Fatal(err)
	}
	records, err := state.NewEventJournal(state.EventJournalPath(statePath), 1<<20).List(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 || records[0].Result != "no_sender" || records[1].Result != "deduplicated" {
		t.Fatalf("records = %#v, want native delivery then deduplicated monitor fallback", records)
	}
}

func TestDispatchRecordsPartialWhenOneRemoteChannelSucceeds(t *testing.T) {
	result := deliveryResult(&notify.DeliveryError{Successful: true, Details: []string{"ntfy: down"}})
	if result != "partial" {
		t.Fatalf("deliveryResult() = %q, want partial", result)
	}
}

func TestReserveTerminalSourceSuppressesWorkBuddyFallbackAfterNativeHook(t *testing.T) {
	store := state.NewStore(filepath.Join(t.TempDir(), "state.json"))
	now := time.Now()
	first, err := reserveTerminalSource(store, notify.Message{Agent: "workbuddy", Event: "run_completed", SessionID: "s1"}, "native_hook", time.Minute, now)
	if err != nil || !first {
		t.Fatalf("native hook reservation = (%v, %v), want true, nil", first, err)
	}
	second, err := reserveTerminalSource(store, notify.Message{Agent: "workbuddy", Event: "run_completed", SessionID: "s1"}, "desktop_monitor", time.Minute, now.Add(time.Second))
	if err != nil || second {
		t.Fatalf("desktop fallback reservation = (%v, %v), want false, nil", second, err)
	}
}

func TestBuildSendersUsesClaudeCodeConfigByDefault(t *testing.T) {
	cfg := config.Default()
	cfg.Notify.ClaudeCode.Channels.System.Enabled = true
	cfg.Notify.ClaudeCode.Channels.Feishu.Enabled = true
	cfg.Notify.ClaudeCode.Events = []string{"run_completed"}
	cfg.Notify.Codex.Channels.System.Enabled = false
	cfg.Notify.Codex.Channels.Feishu.Enabled = false

	senders := buildSenders(cfg, notify.Message{Event: "run_completed"})

	if len(senders) != 2 {
		t.Fatalf("len(senders) = %d, want 2", len(senders))
	}
	if senders[0].Name() != "system" {
		t.Fatalf("senders[0] = %q, want system", senders[0].Name())
	}
	if senders[1].Name() != "feishu" {
		t.Fatalf("senders[1] = %q, want feishu", senders[1].Name())
	}
}

func TestBuildSendersUsesCodexConfigForCodexMessages(t *testing.T) {
	cfg := config.Default()
	cfg.Notify.ClaudeCode.Channels.System.Enabled = true
	cfg.Notify.ClaudeCode.Channels.Feishu.Enabled = true
	cfg.Notify.ClaudeCode.Events = []string{"run_completed"}
	cfg.Notify.Codex.Channels.System.Enabled = true
	cfg.Notify.Codex.Channels.Feishu.Enabled = false
	cfg.Notify.Codex.Events = []string{"run_completed"}

	senders := buildSenders(cfg, notify.Message{Agent: "codex", Event: "run_completed"})

	if len(senders) != 1 {
		t.Fatalf("len(senders) = %d, want 1", len(senders))
	}
	if senders[0].Name() != "system" {
		t.Fatalf("senders[0] = %q, want system", senders[0].Name())
	}
}

func TestBuildSendersFiltersCodexEventsNotSelected(t *testing.T) {
	cfg := config.Default()
	cfg.Notify.Codex.Channels.System.Enabled = true
	cfg.Notify.Codex.Channels.Feishu.Enabled = true
	// 用户只订阅了 permission_required，run_completed 不应触发任何 sender
	cfg.Notify.Codex.Events = []string{"permission_required"}

	senders := buildSenders(cfg, notify.Message{Agent: "codex", Event: "run_completed"})

	if len(senders) != 0 {
		t.Fatalf("len(senders) = %d, want 0 (event not subscribed)", len(senders))
	}
}

func TestBuildSendersSendsSubscribedCodexEvent(t *testing.T) {
	cfg := config.Default()
	cfg.Notify.Codex.Channels.System.Enabled = true
	cfg.Notify.Codex.Channels.Feishu.Enabled = true
	cfg.Notify.Codex.Events = []string{"permission_required", "run_completed"}

	senders := buildSenders(cfg, notify.Message{Agent: "codex", Event: "permission_required"})

	if len(senders) != 2 {
		t.Fatalf("len(senders) = %d, want 2", len(senders))
	}
}

func TestBuildSendersUsesGrokConfigForGrokMessages(t *testing.T) {
	cfg := config.Default()
	cfg.Notify.ClaudeCode.Channels.System.Enabled = true
	cfg.Notify.ClaudeCode.Events = []string{"run_completed"}
	cfg.Notify.Grok.Channels.System.Enabled = true
	cfg.Notify.Grok.Channels.Feishu.Enabled = false
	cfg.Notify.Grok.Events = []string{"run_completed"}

	senders := buildSenders(cfg, notify.Message{Agent: "grok", Event: "run_completed"})

	if len(senders) != 1 {
		t.Fatalf("len(senders) = %d, want 1", len(senders))
	}
	if senders[0].Name() != "system" {
		t.Fatalf("senders[0] = %q, want system", senders[0].Name())
	}
}

func TestBuildSendersUsesDroidConfigForDroidMessages(t *testing.T) {
	cfg := config.Default()
	cfg.Notify.ClaudeCode.Channels.System.Enabled = true
	cfg.Notify.ClaudeCode.Events = []string{"run_completed"}
	cfg.Notify.Droid.Channels.System.Enabled = true
	cfg.Notify.Droid.Channels.Feishu.Enabled = false
	cfg.Notify.Droid.Events = []string{"run_completed"}

	senders := buildSenders(cfg, notify.Message{Agent: "droid", Event: "run_completed"})

	if len(senders) != 1 {
		t.Fatalf("len(senders) = %d, want 1", len(senders))
	}
	if senders[0].Name() != "system" {
		t.Fatalf("senders[0] = %q, want system", senders[0].Name())
	}
}

// TestBuildSendersDroidEventNotEnabled verifies that a Droid message is
// dropped when its event is not in the enabled list, and that another agent's
// enabled event does not leak into Droid dispatch.
func TestBuildSendersDroidEventNotEnabled(t *testing.T) {
	cfg := config.Default()
	cfg.Notify.Droid.Channels.System.Enabled = true
	cfg.Notify.Droid.Events = []string{"permission_required"}

	senders := buildSenders(cfg, notify.Message{Agent: "droid", Event: "run_completed"})

	if len(senders) != 0 {
		t.Fatalf("len(senders) = %d, want 0 for disabled event", len(senders))
	}
}

func TestBuildSendersAddsBarkForCodex(t *testing.T) {
	cfg := config.Default()
	cfg.Notify.Codex.Channels.Bark.Enabled = true
	cfg.Notify.Codex.Channels.Bark.WebhookURL = "https://api.day.app/key"
	cfg.Notify.Codex.Events = []string{"run_completed"}

	senders := buildSenders(cfg, notify.Message{Agent: "codex", Event: "run_completed"})

	if len(senders) != 1 {
		t.Fatalf("len(senders) = %d, want 1", len(senders))
	}
	if senders[0].Name() != "bark" {
		t.Fatalf("senders[0] = %q, want bark", senders[0].Name())
	}
}

func TestFilterFrozenSendersDropsFrozenChannels(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	logPath := filepath.Join(dir, "log.txt")
	now := time.Now()
	store := state.NewFreezeStore(state.FreezePath(statePath))
	if err := store.Set(now.Add(time.Hour), []string{"feishu"}, now); err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()
	cfg.Notify.ClaudeCode.Channels.System.Enabled = true
	cfg.Notify.ClaudeCode.Channels.Feishu.Enabled = true
	cfg.Notify.ClaudeCode.Events = []string{"run_completed"}
	senders := buildSenders(cfg, notify.Message{Event: "run_completed"})
	if len(senders) != 2 {
		t.Fatalf("precondition: len(senders)=%d", len(senders))
	}

	filtered := filterFrozenSenders(statePath, logPath, "run_completed", senders, now)
	if len(filtered) != 1 || filtered[0].Name() != "system" {
		names := make([]string, len(filtered))
		for i, s := range filtered {
			names[i] = s.Name()
		}
		t.Fatalf("filtered=%v, want [system]", names)
	}
}

func TestFilterFrozenSendersInactivePassthrough(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	logPath := filepath.Join(dir, "log.txt")

	cfg := config.Default()
	cfg.Notify.ClaudeCode.Channels.System.Enabled = true
	cfg.Notify.ClaudeCode.Channels.Feishu.Enabled = true
	cfg.Notify.ClaudeCode.Events = []string{"run_completed"}
	senders := buildSenders(cfg, notify.Message{Event: "run_completed"})

	filtered := filterFrozenSenders(statePath, logPath, "run_completed", senders, time.Now())
	if len(filtered) != len(senders) {
		t.Fatalf("len(filtered)=%d, want %d", len(filtered), len(senders))
	}
}
