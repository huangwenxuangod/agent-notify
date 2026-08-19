package bridge

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"context"
	"github.com/hellolib/agent-notify/internal/agenthooks"
	"github.com/hellolib/agent-notify/internal/agentintegrations"
	"github.com/hellolib/agent-notify/internal/autostart"
	"github.com/hellolib/agent-notify/internal/config"
	"github.com/hellolib/agent-notify/internal/notify"
	"github.com/hellolib/agent-notify/internal/state"
)

type Options struct {
	ConfigPath    string
	StatePath     string
	LogPath       string
	BinaryPath    string
	AutostartPath string
	RemoteOnly    bool
}

type Service struct {
	options    Options
	remoteOnly bool
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
	if options.AutostartPath == "" {
		options.AutostartPath = options.BinaryPath
	}
	return &Service{options: options, remoteOnly: options.RemoteOnly}, nil
}

func (s *Service) GetConfig() (config.Config, error)  { return config.Load(s.options.ConfigPath) }
func (s *Service) SaveConfig(cfg config.Config) error { return config.Save(s.options.ConfigPath, cfg) }

func (s *Service) IngestMessage(ctx context.Context, msg notify.Message) error {
	cfg, err := s.GetConfig()
	if err != nil {
		return err
	}
	if s.remoteOnly {
		return agenthooks.DispatchRemote(ctx, cfg, s.options.StatePath, s.options.LogPath, msg)
	}
	return agenthooks.Dispatch(ctx, cfg, s.options.StatePath, s.options.LogPath, msg)
}

// RecordEvent persists a host-side dispatch outcome without sending it again.
func (s *Service) RecordEvent(msg notify.Message, result string) error {
	if result == "" {
		result = "sent"
	}
	sourceApp := msg.SourceApp.BundleID
	if sourceApp == "" {
		sourceApp = msg.SourceApp.TermProgram
	}
	return state.NewEventJournal(state.EventJournalPath(s.options.StatePath), 5<<20).Append(state.EventRecord{
		Timestamp: time.Now().UTC(), EventID: msg.EventID, Agent: msg.Agent, Event: msg.Event,
		SessionID: msg.SessionID, TurnID: msg.TurnID, RunID: msg.RunID, SourceEvent: msg.SourceEvent, Workspace: msg.Workspace, Title: msg.Title,
		Body: msg.Body, Origin: msg.Origin, SourceApp: sourceApp, Result: result,
	})
}

func (s *Service) ScanAgents() ([]AgentStatus, error) {
	cfg, err := s.GetConfig()
	if err != nil {
		return nil, err
	}
	statuses := make([]AgentStatus, 0)
	for _, descriptor := range agentintegrations.All() {
		status := AgentStatus{ID: descriptor.ID, Name: descriptor.Name, Installed: descriptor.Integration.DetectInstalled()}
		userPath, userErr := descriptor.Integration.SettingsPath("user")
		projectPath, projectErr := descriptor.Integration.SettingsPath("project")
		if userErr == nil {
			status.UserSettings = userPath
		}
		if projectErr == nil {
			status.ProjectSettings = projectPath
		}
		path := userPath
		if cfgTarget := descriptor.Target(&cfg); cfgTarget.InstallScope == "project" && projectErr == nil {
			path = projectPath
		}
		if path != "" {
			status.HookInstalled, err = descriptor.Integration.IsHookInstalled(path)
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
	for _, descriptor := range agentintegrations.All() {
		if len(selected) > 0 && !selected[descriptor.ID] {
			continue
		}
		path, pathErr := descriptor.Integration.SettingsPath(scope)
		r := AgentResult{ID: descriptor.ID, Path: path}
		if pathErr == nil {
			pathErr = descriptor.Integration.Install(path, binary)
		}
		if pathErr != nil {
			r.Error = pathErr.Error()
		} else {
			r.Success = true
			target := descriptor.Target(&cfg)
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
	for _, descriptor := range agentintegrations.All() {
		if len(selected) > 0 && !selected[descriptor.ID] {
			continue
		}
		target := descriptor.Target(&cfg)
		paths := append([]string(nil), target.InstalledPaths...)
		for _, scope := range []string{"user", "project"} {
			if path, e := descriptor.Integration.SettingsPath(scope); e == nil && !containsPath(paths, path) {
				paths = append(paths, path)
			}
		}
		r := AgentResult{ID: descriptor.ID, Success: true}
		for _, path := range paths {
			if e := descriptor.Integration.Uninstall(path); e != nil {
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

func (s *Service) FreezeStatus() state.FreezeState {
	return state.NewFreezeStore(state.FreezePath(s.options.StatePath)).Load()
}

func (s *Service) FreezeRemoteChannels(duration time.Duration) (state.FreezeState, error) {
	if duration <= 0 {
		return state.FreezeState{}, fmt.Errorf("freeze duration must be positive")
	}
	cfg, err := s.GetConfig()
	if err != nil {
		return state.FreezeState{}, err
	}
	channels := configuredRemoteChannels(cfg)
	if len(channels) == 0 {
		return state.FreezeState{}, fmt.Errorf("no configured remote notification channels")
	}
	now := time.Now()
	store := state.NewFreezeStore(state.FreezePath(s.options.StatePath))
	if err := store.Set(now.Add(duration), channels, now); err != nil {
		return state.FreezeState{}, err
	}
	return store.Load(), nil
}

func (s *Service) ClearFreeze() error {
	return state.NewFreezeStore(state.FreezePath(s.options.StatePath)).Clear()
}

func configuredRemoteChannels(cfg config.Config) []string {
	remote := cfg.Remote
	seen := map[string]bool{
		"feishu":       configuredWebhook(remote.Feishu.Enabled, remote.Feishu.WebhookURL),
		"wechat":       configuredWebhook(remote.Wechat.Enabled, remote.Wechat.WebhookURL),
		"wechat-ilink": remote.WechatIlink.Enabled,
		"wechat-work":  configuredWebhook(remote.WechatWork.Enabled, remote.WechatWork.WebhookURL),
		"dingtalk":     configuredWebhook(remote.DingTalk.Enabled, remote.DingTalk.WebhookURL),
		"bark":         configuredWebhook(remote.Bark.Enabled, remote.Bark.WebhookURL),
		"ntfy":         configuredWebhook(remote.Ntfy.Enabled, remote.Ntfy.TopicURL),
		"slack":        configuredWebhook(remote.Slack.Enabled, remote.Slack.WebhookURL),
	}
	channels := make([]string, 0, len(seen))
	for _, name := range state.RemoteFreezeChannels {
		if seen[name] {
			channels = append(channels, name)
		}
	}
	return channels
}

func (s *Service) TestNotification() error {
	return fmt.Errorf("notification test requires an enabled system channel")
}

// TestRemoteChannel sends one test event through exactly one configured remote
// channel. It never persists the temporary single-channel config.
func (s *Service) TestRemoteChannel(ctx context.Context, channel string) error {
	cfg, err := s.GetConfig()
	if err != nil {
		return err
	}
	channel = strings.ToLower(strings.TrimSpace(channel))
	testCfg := cfg
	testCfg.Remote = config.RemoteDeliveryConfig{}
	switch channel {
	case "feishu":
		testCfg.Remote.Feishu = cfg.Remote.Feishu
		if !configuredFeishu(testCfg.Remote.Feishu) {
			return fmt.Errorf("feishu channel is not configured")
		}
	case "wechat":
		testCfg.Remote.Wechat = cfg.Remote.Wechat
		if !configuredWebhook(testCfg.Remote.Wechat.Enabled, testCfg.Remote.Wechat.WebhookURL) {
			return fmt.Errorf("custom webhook channel is not configured")
		}
	case "wechat-ilink":
		testCfg.Remote.WechatIlink = cfg.Remote.WechatIlink
		if !testCfg.Remote.WechatIlink.Enabled {
			return fmt.Errorf("wechat ilink channel is not enabled")
		}
	case "wechat-work":
		testCfg.Remote.WechatWork = cfg.Remote.WechatWork
		if !configuredWebhook(testCfg.Remote.WechatWork.Enabled, testCfg.Remote.WechatWork.WebhookURL) {
			return fmt.Errorf("wechat work channel is not configured")
		}
	case "dingtalk":
		testCfg.Remote.DingTalk = cfg.Remote.DingTalk
		if !configuredWebhook(testCfg.Remote.DingTalk.Enabled, testCfg.Remote.DingTalk.WebhookURL) {
			return fmt.Errorf("dingtalk channel is not configured")
		}
	case "bark":
		testCfg.Remote.Bark = cfg.Remote.Bark
		if !configuredWebhook(testCfg.Remote.Bark.Enabled, testCfg.Remote.Bark.WebhookURL) {
			return fmt.Errorf("bark channel is not configured")
		}
	case "ntfy":
		testCfg.Remote.Ntfy = cfg.Remote.Ntfy
		if !configuredWebhook(testCfg.Remote.Ntfy.Enabled, testCfg.Remote.Ntfy.TopicURL) {
			return fmt.Errorf("ntfy channel is not configured")
		}
	case "slack":
		testCfg.Remote.Slack = cfg.Remote.Slack
		if !configuredWebhook(testCfg.Remote.Slack.Enabled, testCfg.Remote.Slack.WebhookURL) {
			return fmt.Errorf("slack channel is not configured")
		}
	default:
		return fmt.Errorf("unsupported remote channel %q", channel)
	}

	// A channel connection test must not depend on which agents are currently
	// installed or have selected event types.
	testCfg.Notify.Codex.Events = []string{"run_completed"}
	message := notify.Message{
		Agent: "codex", Event: "run_completed", Title: "Agent Notify 测试通知",
		Body: "机器人通知渠道已连接", SessionID: fmt.Sprintf("channel-test-%d", time.Now().UnixNano()),
	}
	return agenthooks.DispatchRemote(ctx, testCfg, s.options.StatePath, s.options.LogPath, message)
}

func configuredFeishu(c config.FeishuWebhookConfig) bool {
	return configuredWebhook(c.Enabled, c.WebhookURL)
}

func configuredWebhook(enabled bool, endpoint string) bool {
	return enabled && strings.TrimSpace(endpoint) != ""
}
func (s *Service) AutostartStatus() AutostartStatus {
	st, err := autostart.New(s.options.AutostartPath).Status()
	if err != nil {
		return AutostartStatus{Error: err.Error()}
	}
	return AutostartStatus{Supported: st.Supported, Enabled: st.Enabled, Platform: st.Platform, Path: st.Path}
}
func (s *Service) SetAutostart(enabled bool) error {
	m := autostart.New(s.options.AutostartPath)
	if enabled {
		return m.Enable()
	}
	return m.Disable()
}
