package agenthooks

import (
	"testing"

	"github.com/hellolib/agent-notify/internal/config"
	"github.com/hellolib/agent-notify/internal/notify"
)

func TestBuildRemoteSendersDoesNotIncludeSystemSender(t *testing.T) {
	cfg := config.Default()
	cfg.Notify.Codex.Channels.System.Enabled = true
	cfg.Notify.Codex.Events = []string{"run_completed"}
	cfg.Remote.Ntfy.Enabled = true
	cfg.Remote.Ntfy.TopicURL = "https://ntfy.sh/topic"

	senders := buildRemoteSenders(cfg, notify.Message{Agent: "codex", Event: "run_completed"})
	if len(senders) != 1 || senders[0].Name() != "ntfy" {
		t.Fatalf("senders = %#v, want only ntfy", senders)
	}
}

func TestBuildSystemSenderUsesAgentLocalConfiguration(t *testing.T) {
	cfg := config.Default()
	cfg.Notify.Codex.Channels.System.Enabled = true
	cfg.Notify.Codex.Events = []string{"run_completed"}

	sender := buildSystemSender(cfg, notify.Message{Agent: "codex", Event: "run_completed"})
	if sender == nil || sender.Name() != "system" {
		t.Fatalf("system sender = %#v, want system sender", sender)
	}
}
