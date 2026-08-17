package agentintegrations

import "github.com/hellolib/agent-notify/internal/config"

// Descriptor is the single registration point for an agent integration.
// Keep agent-specific hook parsing and installation in their existing packages;
// this only centralizes metadata shared by setup and management surfaces.
type Descriptor struct {
	ID          string
	Name        string
	Integration Integration
	Target      func(*config.Config) *config.AgentTargetConfig
	Notify      func(*config.Config) *config.AgentNotifyConfig
}

// All returns descriptors in the user-facing configuration order.
func All() []Descriptor {
	return []Descriptor{
		{
			ID: "claude_code", Name: "Claude Code", Integration: NewClaudeIntegration(),
			Target: func(cfg *config.Config) *config.AgentTargetConfig { return &cfg.Agent.ClaudeCode },
			Notify: func(cfg *config.Config) *config.AgentNotifyConfig { return &cfg.Notify.ClaudeCode },
		},
		{
			ID: "codex", Name: "Codex", Integration: NewCodexIntegration(),
			Target: func(cfg *config.Config) *config.AgentTargetConfig { return &cfg.Agent.Codex },
			Notify: func(cfg *config.Config) *config.AgentNotifyConfig { return &cfg.Notify.Codex },
		},
		{
			ID: "zcode", Name: "ZCode", Integration: NewZcodeIntegration(),
			Target: func(cfg *config.Config) *config.AgentTargetConfig { return &cfg.Agent.ZCode },
			Notify: func(cfg *config.Config) *config.AgentNotifyConfig { return &cfg.Notify.ZCode },
		},
		{
			ID: "grok", Name: "Grok", Integration: NewGrokIntegration(),
			Target: func(cfg *config.Config) *config.AgentTargetConfig { return &cfg.Agent.Grok },
			Notify: func(cfg *config.Config) *config.AgentNotifyConfig { return &cfg.Notify.Grok },
		},
		{
			ID: "droid", Name: "Droid", Integration: NewDroidIntegration(),
			Target: func(cfg *config.Config) *config.AgentTargetConfig { return &cfg.Agent.Droid },
			Notify: func(cfg *config.Config) *config.AgentNotifyConfig { return &cfg.Notify.Droid },
		},
		{
			ID: "opencode", Name: "OpenCode", Integration: NewOpenCodeIntegration(),
			Target: func(cfg *config.Config) *config.AgentTargetConfig { return &cfg.Agent.OpenCode },
			Notify: func(cfg *config.Config) *config.AgentNotifyConfig { return &cfg.Notify.OpenCode },
		},
		{
			ID: "workbuddy", Name: "WorkBuddy / CodeBuddy", Integration: NewWorkBuddyIntegration(),
			Target: func(cfg *config.Config) *config.AgentTargetConfig { return &cfg.Agent.WorkBuddy },
			Notify: func(cfg *config.Config) *config.AgentNotifyConfig { return &cfg.Notify.WorkBuddy },
		},
		{
			ID: "hermes", Name: "Hermes Agent", Integration: NewHermesIntegration(),
			Target: func(cfg *config.Config) *config.AgentTargetConfig { return &cfg.Agent.Hermes },
			Notify: func(cfg *config.Config) *config.AgentNotifyConfig { return &cfg.Notify.Hermes },
		},
		{
			ID: "openclaw", Name: "OpenClaw", Integration: NewOpenClawIntegration(),
			Target: func(cfg *config.Config) *config.AgentTargetConfig { return &cfg.Agent.OpenClaw },
			Notify: func(cfg *config.Config) *config.AgentNotifyConfig { return &cfg.Notify.OpenClaw },
		},
		{
			ID: "pi", Name: "Pi", Integration: NewPiIntegration(),
			Target: func(cfg *config.Config) *config.AgentTargetConfig { return &cfg.Agent.Pi },
			Notify: func(cfg *config.Config) *config.AgentNotifyConfig { return &cfg.Notify.Pi },
		},
	}
}
