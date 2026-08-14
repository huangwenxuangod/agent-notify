package cli

import (
	"testing"

	"github.com/hellolib/agent-notify/internal/config"
)

func TestApplyChannelToAgentsCoversEveryRegisteredAgent(t *testing.T) {
	cfg := config.Default()
	cfg.Agent.Hermes.Enabled = true
	cfg.Agent.OpenClaw.Enabled = true

	applyChannelToAgents(&cfg, func(enabled bool, notify *config.AgentNotifyConfig) {
		if enabled {
			notify.Channels.Bark.Enabled = true
		}
	})

	if !cfg.Notify.Hermes.Channels.Bark.Enabled {
		t.Fatal("Hermes channel configuration was skipped")
	}
	if !cfg.Notify.OpenClaw.Channels.Bark.Enabled {
		t.Fatal("OpenClaw channel configuration was skipped")
	}
}
