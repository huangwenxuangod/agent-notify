package bridge

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/hellolib/agent-notify/internal/agentintegrations"
	"github.com/hellolib/agent-notify/internal/autostart"
	"github.com/hellolib/agent-notify/internal/config"
	"github.com/hellolib/agent-notify/internal/state"
)

type Options struct {
	ConfigPath string
	StatePath  string
	LogPath    string
	BinaryPath string
}

type Service struct {
	options Options
}

type AgentStatus struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Installed       bool   `json:"installed"`
	HookInstalled   bool   `json:"hook_installed"`
	UserSettings    string `json:"user_settings,omitempty"`
	ProjectSettings string `json:"project_settings,omitempty"`
	Error           string `json:"error,omitempty"`
}

type SetupRequest struct {
	Agents     []string `json:"agents"`
	Scope      string   `json:"scope"`
	BinaryPath string   `json:"binary_path,omitempty"`
}

type AgentResult struct {
	ID      string `json:"id"`
	Success bool   `json:"success"`
	Path    string `json:"path,omitempty"`
	Error   string `json:"error,omitempty"`
}

type SetupResult struct {
	Results []AgentResult `json:"results"`
}

type AutostartStatus struct {
	Supported bool   `json:"supported"`
	Enabled   bool   `json:"enabled"`
	Platform  string `json:"platform"`
	Path      string `json:"path,omitempty"`
	Error     string `json:"error,omitempty"`
}

func NewService(options Options) (*Service, error) {
	if options.ConfigPath == "" {
		var err error
		options.ConfigPath, err = config.DefaultPath()
		if err != nil {
			return nil, err
		}
	}
	if options.StatePath == "" {
		var err error
		options.StatePath, err = config.StatePath()
		if err != nil {
			return nil, err
		}
	}
	if options.LogPath == "" {
		var err error
		options.LogPath, err = config.LogPath()
		if err != nil {
			return nil, err
		}
	}
	if options.BinaryPath == "" {
		options.BinaryPath = os.Args[0]
	}
	return &Service{options: options}, nil
}

type agentSpec struct {
	id          string
	integration agentintegrations.Integration
	target      func(*config.Config) *config.AgentTargetConfig
}

func (s *Service) specs(cfg *config.Config) []agentSpec {
	return []agentSpec{
		{"claude_code", agentintegrations.NewClaudeIntegration(), func(c *config.Config) *config.AgentTargetConfig { return &c.Agent.ClaudeCode }},
		{"codex", agentintegrations.NewCodexIntegration(), func(c *config.Config) *config.AgentTargetConfig { return &c.Agent.Codex }},
		{"zcode", agentintegrations.NewZcodeIntegration(), func(c *config.Config) *config.AgentTargetConfig { return &c.Agent.ZCode }},
		{"grok", agentintegrations.NewGrokIntegration(), func(c *config.Config) *config.AgentTargetConfig { return &c.Agent.Grok }},
		{"droid", agentintegrations.NewDroidIntegration(), func(c *config.Config) *config.AgentTargetConfig { return &c.Agent.Droid }},
		{"opencode", agentintegrations.NewOpenCodeIntegration(), func(c *config.Config) *config.AgentTargetConfig { return &c.Agent.OpenCode }},
		{"workbuddy", agentintegrations.NewWorkBuddyIntegration(), func(c *config.Config) *config.AgentTargetConfig { return &c.Agent.WorkBuddy }},
		{"hermes", agentintegrations.NewHermesIntegration(), func(c *config.Config) *config.AgentTargetConfig { return &c.Agent.Hermes }},
		{"openclaw", agentintegrations.NewOpenClawIntegration(), func(c *config.Config) *config.AgentTargetConfig { return &c.Agent.OpenClaw }},
	}
}

func (s *Service) GetConfig() (config.Config, error)  { return config.Load(s.options.ConfigPath) }
func (s *Service) SaveConfig(cfg config.Config) error { return config.Save(s.options.ConfigPath, cfg) }

func (s *Service) ScanAgents() ([]AgentStatus, error) {
	cfg, err := s.GetConfig()
	if err != nil {
		return nil, err
	}
	statuses := make([]AgentStatus, 0)
	for _, spec := range s.specs(&cfg) {
		status := AgentStatus{ID: spec.id, Name: spec.integration.Name(), Installed: spec.integration.DetectInstalled()}
		userPath, userErr := spec.integration.SettingsPath("user")
		projectPath, projectErr := spec.integration.SettingsPath("project")
		if userErr == nil {
			status.UserSettings = userPath
		}
		if projectErr == nil {
			status.ProjectSettings = projectPath
		}
		path := userPath
		if cfgTarget := spec.target(&cfg); cfgTarget.InstallScope == "project" && projectErr == nil {
			path = projectPath
		}
		if path != "" {
			status.HookInstalled, err = spec.integration.IsHookInstalled(path)
			if err != nil {
				status.Error = err.Error()
			}
		}
		statuses = append(statuses, status)
	}
	return statuses, nil
}

func (s *Service) InstallAgents(req SetupRequest) (SetupResult, error) {
	cfg, err := s.GetConfig()
	if err != nil {
		return SetupResult{}, err
	}
	scope := strings.TrimSpace(req.Scope)
	if scope == "" {
		scope = "user"
	}
	binary := req.BinaryPath
	if binary == "" {
		binary = s.options.BinaryPath
	}
	selected := map[string]bool{}
	for _, id := range req.Agents {
		selected[id] = true
	}
	result := SetupResult{}
	for _, spec := range s.specs(&cfg) {
		if len(selected) > 0 && !selected[spec.id] {
			continue
		}
		path, pathErr := spec.integration.SettingsPath(scope)
		r := AgentResult{ID: spec.id, Path: path}
		if pathErr == nil {
			pathErr = spec.integration.Install(path, binary)
		}
		if pathErr != nil {
			r.Error = pathErr.Error()
		} else {
			r.Success = true
			target := spec.target(&cfg)
			target.Enabled = true
			target.InstallScope = scope
			target.InstalledPaths = config.RecordInstalledPath(target.InstalledPaths, path)
		}
		result.Results = append(result.Results, r)
	}
	if err := s.SaveConfig(cfg); err != nil {
		return result, err
	}
	return result, nil
}

func (s *Service) UninstallAgents(req SetupRequest) (SetupResult, error) {
	cfg, err := s.GetConfig()
	if err != nil {
		return SetupResult{}, err
	}
	selected := map[string]bool{}
	for _, id := range req.Agents {
		selected[id] = true
	}
	result := SetupResult{}
	for _, spec := range s.specs(&cfg) {
		if len(selected) > 0 && !selected[spec.id] {
			continue
		}
		target := spec.target(&cfg)
		paths := append([]string(nil), target.InstalledPaths...)
		for _, scope := range []string{"user", "project"} {
			if path, e := spec.integration.SettingsPath(scope); e == nil && !containsPath(paths, path) {
				paths = append(paths, path)
			}
		}
		r := AgentResult{ID: spec.id, Success: true}
		for _, path := range paths {
			if e := spec.integration.Uninstall(path); e != nil {
				r.Success = false
				r.Error = e.Error()
				break
			}
		}
		if r.Success {
			target.Enabled = false
			target.InstalledPaths = nil
		}
		result.Results = append(result.Results, r)
	}
	if err := s.SaveConfig(cfg); err != nil {
		return result, err
	}
	return result, nil
}

func containsPath(paths []string, path string) bool {
	for _, p := range paths {
		if p == path {
			return true
		}
	}
	return false
}

func (s *Service) ListEvents(limit int) ([]state.EventRecord, error) {
	return state.NewEventJournal(state.EventJournalPath(s.options.StatePath), 5<<20).List(limit)
}
func (s *Service) ListLogs(limit int) ([]string, error) {
	data, err := os.ReadFile(s.options.LogPath)
	if errors.Is(err, os.ErrNotExist) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return []string{}, nil
	}
	if limit > 0 && len(lines) > limit {
		lines = lines[len(lines)-limit:]
	}
	return lines, nil
}

func (s *Service) TestNotification() error {
	return fmt.Errorf("notification test requires an enabled system channel")
}
func (s *Service) AutostartStatus() AutostartStatus {
	st, err := autostart.New(s.options.BinaryPath).Status()
	if err != nil {
		return AutostartStatus{Error: err.Error()}
	}
	return AutostartStatus{Supported: st.Supported, Enabled: st.Enabled, Platform: st.Platform, Path: st.Path}
}
func (s *Service) SetAutostart(enabled bool) error {
	m := autostart.New(s.options.BinaryPath)
	if enabled {
		return m.Enable()
	}
	return m.Disable()
}
