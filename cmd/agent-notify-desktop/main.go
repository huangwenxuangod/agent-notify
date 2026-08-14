package main

import (
	"context"
	"embed"
	"fmt"
	"github.com/hellolib/agent-notify/internal/agenthooks"
	"github.com/hellolib/agent-notify/internal/app/tester"
	"github.com/hellolib/agent-notify/internal/bridge"
	"github.com/hellolib/agent-notify/internal/bridgeclient"
	"github.com/hellolib/agent-notify/internal/codexmonitor"
	"github.com/hellolib/agent-notify/internal/common"
	"github.com/hellolib/agent-notify/internal/config"
	"github.com/hellolib/agent-notify/internal/notify"
	"github.com/hellolib/agent-notify/internal/state"
	"github.com/hellolib/agent-notify/internal/tray"
	"github.com/hellolib/agent-notify/internal/workbuddymonitor"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"io/fs"
	"log"
	"os"
	"os/exec"
	goruntime "runtime"
	"sync"
	"time"
)

//go:embed all:frontend/dist
var assets embed.FS

func assetRoot() (fs.FS, error) {
	return fs.Sub(assets, "frontend/dist")
}

type App struct {
	service     *bridge.Service
	autoSetupMu sync.Mutex
}

var saveBridgeConfig = bridgeclient.SaveConfig

func desktopServiceOptions(hookBinary, autostartBinary string) bridge.Options {
	return bridge.Options{BinaryPath: hookBinary, AutostartPath: autostartBinary}
}

type HookRuntimeStatus struct {
	Installed   bool   `json:"installed"`
	LastEventAt string `json:"last_event_at,omitempty"`
	LastEvent   string `json:"last_event,omitempty"`
}

type DesktopStatus struct {
	Agents       []bridge.AgentStatus `json:"agents"`
	Events       []interface{}        `json:"events"`
	Logs         []string             `json:"logs"`
	PendingRetry int                  `json:"pending_retry"`
}

const codexHookReviewScript = `tell application "Terminal"
	activate
	do script "printf '%s\\n' 'Agent Notify: type /hooks and trust agent-notify.'; exec codex"
end tell`

const workBuddyHookReviewScript = `tell application "Terminal"
	activate
	do script "printf '%s\\n' 'Agent Notify: type /hooks and approve agent-notify.'; exec codebuddy"
end tell`

var runCodexHookReview = func(name string, args ...string) error {
	return exec.Command(name, args...).Run()
}

var runWorkBuddyHookReview = func(name string, args ...string) error {
	return exec.Command(name, args...).Run()
}

func openCodexHookReview() error {
	return runCodexHookReview("osascript", "-e", codexHookReviewScript)
}

func openWorkBuddyHookReview() error {
	return runWorkBuddyHookReview("osascript", "-e", workBuddyHookReviewScript)
}

func autoSetupTargets(agents []bridge.AgentStatus) []string {
	targets := make([]string, 0)
	for _, agent := range agents {
		if agent.Installed && !agent.HookInstalled {
			targets = append(targets, agent.ID)
		}
	}
	return targets
}

// hookReviewTargets contains only agents that expose a native review surface.
// The caller supplies IDs installed successfully in this run.
func hookReviewTargets(installed []string) []string {
	wanted := map[string]bool{"codex": true, "workbuddy": true}
	seen := make(map[string]bool)
	targets := make([]string, 0, 2)
	for _, id := range installed {
		if wanted[id] && !seen[id] {
			targets = append(targets, id)
			seen[id] = true
		}
	}
	return targets
}

func openNativeHookReviews(ids []string) {
	if goruntime.GOOS != "darwin" {
		return
	}
	for _, id := range hookReviewTargets(ids) {
		var err error
		switch id {
		case "codex":
			err = openCodexHookReview()
		case "workbuddy":
			if _, lookupErr := exec.LookPath("codebuddy"); lookupErr != nil {
				log.Printf("skip workbuddy hook review: codebuddy CLI not found")
				continue
			}
			err = openWorkBuddyHookReview()
		}
		if err != nil {
			log.Printf("open %s hook review: %v", id, err)
		}
	}
}

func notifyConfigForAgent(cfg *config.Config, id string) *config.AgentNotifyConfig {
	switch id {
	case "claude_code":
		return &cfg.Notify.ClaudeCode
	case "codex":
		return &cfg.Notify.Codex
	case "zcode":
		return &cfg.Notify.ZCode
	case "grok":
		return &cfg.Notify.Grok
	case "droid":
		return &cfg.Notify.Droid
	case "opencode":
		return &cfg.Notify.OpenCode
	case "workbuddy":
		return &cfg.Notify.WorkBuddy
	case "hermes":
		return &cfg.Notify.Hermes
	case "openclaw":
		return &cfg.Notify.OpenClaw
	default:
		return nil
	}
}

func enableSystemNotifications(cfg *config.Config, agents []string) bool {
	changed := false
	for _, id := range agents {
		notifyCfg := notifyConfigForAgent(cfg, id)
		if notifyCfg != nil && !notifyCfg.Channels.System.Enabled {
			notifyCfg.Channels.System.Enabled = true
			changed = true
		}
	}
	return changed
}

func setSystemNotifications(cfg *config.Config, agents []string, enabled bool) bool {
	changed := false
	for _, id := range agents {
		notifyCfg := notifyConfigForAgent(cfg, id)
		if notifyCfg != nil && notifyCfg.Channels.System.Enabled != enabled {
			notifyCfg.Channels.System.Enabled = enabled
			changed = true
		}
	}
	return changed
}

// Codex Desktop emits structured task errors in its session journal even
// though the official CLI hook API has no failure event. Enable this desktop
// compatibility event only when Codex is actually connected.
func enableCodexDesktopFailureNotifications(cfg *config.Config, agents []string) bool {
	for _, id := range agents {
		if id != "codex" {
			continue
		}
		for _, event := range cfg.Notify.Codex.Events {
			if event == "run_failed" {
				return false
			}
		}
		cfg.Notify.Codex.Events = append(cfg.Notify.Codex.Events, "run_failed")
		return true
	}
	return false
}

func setClickToFocus(cfg *config.Config, agents []string, enabled bool) bool {
	changed := false
	for _, id := range agents {
		notifyCfg := notifyConfigForAgent(cfg, id)
		if notifyCfg != nil && notifyCfg.Channels.System.ClickToFocus != enabled {
			notifyCfg.Channels.System.ClickToFocus = enabled
			changed = true
		}
	}
	return changed
}

func connectedAgentIDs(agents []bridge.AgentStatus) []string {
	connected := make([]string, 0)
	for _, agent := range agents {
		if agent.Installed && agent.HookInstalled {
			connected = append(connected, agent.ID)
		}
	}
	return connected
}

func hasRemoteConfig(remote config.RemoteDeliveryConfig) bool {
	return (remote.Feishu.Enabled && remote.Feishu.WebhookURL != "") ||
		(remote.Wechat.Enabled && remote.Wechat.WebhookURL != "") ||
		(remote.WechatWork.Enabled && remote.WechatWork.WebhookURL != "") ||
		(remote.DingTalk.Enabled && remote.DingTalk.WebhookURL != "") ||
		(remote.Bark.Enabled && remote.Bark.WebhookURL != "") ||
		(remote.Ntfy.Enabled && remote.Ntfy.TopicURL != "") ||
		(remote.Slack.Enabled && remote.Slack.WebhookURL != "")
}

func shouldImportRemoteConfig(host, docker config.Config) bool {
	return !hasRemoteConfig(host.Remote) && hasRemoteConfig(docker.Remote)
}

func (a *App) syncBridgeConfig(cfg config.Config) {
	if err := saveBridgeConfig(context.Background(), "", "", cfg); err != nil {
		log.Printf("sync host config to bridge: %v", err)
	}
}

func (a *App) migrateRemoteConfig() {
	host, err := a.service.GetConfig()
	if err != nil {
		return
	}
	docker, err := bridgeclient.GetConfig(context.Background(), "", "")
	if err != nil || !shouldImportRemoteConfig(host, docker) {
		return
	}
	host.Remote = docker.Remote
	if err := a.service.SaveConfig(host); err != nil {
		log.Printf("migrate remote notification config: %v", err)
		return
	}
	a.syncBridgeConfig(host)
}

func NewApp() (*App, error) {
	cp, e := config.DefaultPath()
	if e != nil {
		return nil, e
	}
	sp, e := config.StatePath()
	if e != nil {
		return nil, e
	}
	lp, e := config.LogPath()
	if e != nil {
		return nil, e
	}
	options := desktopServiceOptions(common.HookBinaryPath(), common.ResolveBinaryPath(""))
	options.ConfigPath, options.StatePath, options.LogPath = cp, sp, lp
	svc, e := bridge.NewService(options)
	if e != nil {
		return nil, e
	}
	return &App{service: svc}, nil
}
func (a *App) Startup(ctx context.Context)         {}
func (a *App) Scan() ([]bridge.AgentStatus, error) { return a.service.ScanAgents() }
func (a *App) AutoSetup() ([]bridge.AgentStatus, error) {
	a.autoSetupMu.Lock()
	defer a.autoSetupMu.Unlock()
	a.migrateRemoteConfig()
	agents, err := a.service.ScanAgents()
	if err != nil {
		return nil, err
	}
	if targets := autoSetupTargets(agents); len(targets) > 0 {
		result, err := a.service.InstallAgents(bridge.SetupRequest{Agents: targets, Scope: "user"})
		if err != nil {
			return nil, err
		}
		installed := make([]string, 0, len(result.Results))
		for _, item := range result.Results {
			if item.Success {
				installed = append(installed, item.ID)
			}
		}
		openNativeHookReviews(installed)
		agents, err = a.service.ScanAgents()
		if err != nil {
			return nil, err
		}
	}
	connected := connectedAgentIDs(agents)
	if len(connected) == 0 {
		return agents, nil
	}
	cfg, err := a.service.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("read host notification config: %w", err)
	}
	if enableSystemNotifications(&cfg, connected) || enableCodexDesktopFailureNotifications(&cfg, connected) {
		if err := a.service.SaveConfig(cfg); err != nil {
			return nil, fmt.Errorf("save host notification config: %w", err)
		}
	}
	a.syncBridgeConfig(cfg)
	return agents, nil
}
func (a *App) Install(agents []string, scope string) (bridge.SetupResult, error) {
	return a.service.InstallAgents(bridge.SetupRequest{Agents: agents, Scope: scope})
}
func (a *App) Uninstall(agents []string, scope string) (bridge.SetupResult, error) {
	return a.service.UninstallAgents(bridge.SetupRequest{Agents: agents, Scope: scope})
}
func (a *App) Events() ([]interface{}, error) {
	events, err := a.service.ListEvents(8)
	if err != nil {
		return nil, err
	}
	out := make([]interface{}, len(events))
	for i, v := range events {
		out[i] = v
	}
	return out, nil
}
func (a *App) Status() (DesktopStatus, error) {
	agents, err := a.service.ScanAgents()
	if err != nil {
		return DesktopStatus{}, err
	}
	events, err := a.service.ListEvents(8)
	if err != nil {
		return DesktopStatus{}, err
	}
	logs, err := a.service.ListLogs(8)
	if err != nil {
		return DesktopStatus{}, err
	}
	statePath, err := config.StatePath()
	if err != nil {
		return DesktopStatus{}, err
	}
	pending, err := state.NewRemoteOutbox(state.RemoteOutboxPath(statePath)).List()
	if err != nil {
		return DesktopStatus{}, err
	}
	out := make([]interface{}, len(events))
	for i, event := range events {
		out[i] = event
	}
	return DesktopStatus{Agents: agents, Events: out, Logs: logs, PendingRetry: len(pending)}, nil
}
func (a *App) Config() (config.Config, error) {
	return a.service.GetConfig()
}
func (a *App) SaveConfig(cfg config.Config) error {
	if err := a.service.SaveConfig(cfg); err != nil {
		return err
	}
	a.syncBridgeConfig(cfg)
	return nil
}
func (a *App) SendTest(agent string) error {
	if agent == "" {
		agent = "codex"
	}
	_, err := bridgeclient.TryDispatch(context.Background(), notify.Message{
		Agent:     agent,
		Event:     "run_completed",
		Title:     "Agent Notify 测试通知",
		Body:      "已通过 Docker Bridge 发送测试事件",
		SessionID: "desktop-test",
	})
	return err
}
func (a *App) SendTestChannel(channel string) error {
	return a.service.TestRemoteChannel(context.Background(), channel)
}
func (a *App) TestSystemNotification() error {
	_, err := tester.NewService().TestSystem(context.Background())
	return err
}
func (a *App) CodexHookStatus() (HookRuntimeStatus, error) {
	status := HookRuntimeStatus{}
	agents, err := a.service.ScanAgents()
	if err != nil {
		return status, err
	}
	for _, agent := range agents {
		if agent.ID == "codex" {
			status.Installed = agent.HookInstalled
			break
		}
	}
	events, err := bridgeclient.ListEvents(context.Background(), "", "")
	if err != nil {
		return status, nil
	}
	for _, event := range events {
		if event.Agent == "codex" && event.Timestamp.After(time.Time{}) {
			if status.LastEventAt == "" || event.Timestamp.Format(time.RFC3339Nano) > status.LastEventAt {
				status.LastEventAt = event.Timestamp.Format(time.RFC3339)
				status.LastEvent = event.Event
			}
		}
	}
	return status, nil
}
func (a *App) OpenCodexHookReview() error { return openCodexHookReview() }
func (a *App) PauseOneHour() error {
	_, err := bridgeclient.FreezeRemote(context.Background(), "", "", 3600)
	return err
}
func (a *App) ResumeNotifications() error {
	return bridgeclient.ClearFreeze(context.Background(), "", "")
}
func (a *App) Autostart() bridge.AutostartStatus { return a.service.AutostartStatus() }
func (a *App) SetAutostart(enabled bool) error   { return a.service.SetAutostart(enabled) }
func (a *App) ClickToFocus() (bool, error) {
	agents, err := a.service.ScanAgents()
	if err != nil {
		return false, err
	}
	connected := connectedAgentIDs(agents)
	if len(connected) == 0 {
		return true, nil
	}
	cfg, err := a.service.GetConfig()
	if err != nil {
		return false, err
	}
	for _, id := range connected {
		notifyCfg := notifyConfigForAgent(&cfg, id)
		if notifyCfg != nil && !notifyCfg.Channels.System.ClickToFocus {
			return false, nil
		}
	}
	return true, nil
}
func (a *App) SetClickToFocus(enabled bool) error {
	agents, err := a.service.ScanAgents()
	if err != nil {
		return err
	}
	cfg, err := a.service.GetConfig()
	if err != nil {
		return err
	}
	if !setClickToFocus(&cfg, connectedAgentIDs(agents), enabled) {
		return nil
	}
	return a.SaveConfig(cfg)
}

func (a *App) SystemNotifications() (bool, error) {
	agents, err := a.service.ScanAgents()
	if err != nil {
		return false, err
	}
	connected := connectedAgentIDs(agents)
	if len(connected) == 0 {
		return true, nil
	}
	cfg, err := a.service.GetConfig()
	if err != nil {
		return false, err
	}
	for _, id := range connected {
		notifyCfg := notifyConfigForAgent(&cfg, id)
		if notifyCfg != nil && !notifyCfg.Channels.System.Enabled {
			return false, nil
		}
	}
	return true, nil
}

func (a *App) SetSystemNotifications(enabled bool) error {
	agents, err := a.service.ScanAgents()
	if err != nil {
		return err
	}
	cfg, err := a.service.GetConfig()
	if err != nil {
		return err
	}
	if !setSystemNotifications(&cfg, connectedAgentIDs(agents), enabled) {
		return nil
	}
	return a.SaveConfig(cfg)
}

func (a *App) startTray(ctx context.Context) {
	if err := notify.EnsureRegisteredTerminalNotifier(); err != nil {
		log.Printf("register macOS notification helper: %v", err)
	}
	go func() { _, _ = a.AutoSetup() }()
	go a.retryRemoteOutbox(ctx)
	go a.watchWorkBuddyDesktop(ctx)
	go a.watchCodexDesktop(ctx)
	tray.Start(tray.Actions{
		Open:   func() { runtime.WindowShow(ctx) },
		Pause:  func() { _ = a.PauseOneHour() },
		Resume: func() { _ = a.ResumeNotifications() },
		Quit: func() {
			tray.Quit()
			runtime.Quit(ctx)
		},
	})
}

func (a *App) retryRemoteOutbox(ctx context.Context) {
	retry := func() {
		cfg, err := a.service.GetConfig()
		if err != nil {
			log.Printf("read config for remote retry: %v", err)
			return
		}
		statePath, err := config.StatePath()
		if err != nil {
			log.Printf("resolve state path for remote retry: %v", err)
			return
		}
		completed, err := agenthooks.RetryRemoteOutbox(ctx, cfg, statePath, func(retryCtx context.Context, msg notify.Message, senders []notify.Sender) error {
			return agenthooks.DispatchRemoteOutboxItem(retryCtx, cfg, statePath, msg, senders)
		})
		if err != nil {
			log.Printf("retry remote notifications: %v", err)
			return
		}
		if completed > 0 {
			log.Printf("retried %d remote notification(s)", completed)
		}
	}
	retry()
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			retry()
		}
	}
}

func (a *App) watchCodexDesktop(ctx context.Context) {
	path, err := codexmonitor.DefaultSessionsPath()
	if err != nil {
		log.Printf("resolve Codex desktop sessions: %v", err)
		return
	}
	if err := codexmonitor.Watch(ctx, path, func(event codexmonitor.Event) {
		msg := notify.Message{
			Agent: "codex", Event: event.Event, SessionID: event.SessionID,
			TurnID: event.TurnID,
			Title:  notify.FormatTitle("codex", event.Event), Body: event.Body, Origin: "desktop_monitor",
		}
		a.dispatchDesktopMessage(msg)
	}); err != nil {
		log.Printf("watch Codex desktop sessions: %v", err)
	}
}

func (a *App) watchWorkBuddyDesktop(ctx context.Context) {
	if goruntime.GOOS != "darwin" {
		return
	}
	path, err := workbuddymonitor.DefaultLogPath()
	if err != nil {
		log.Printf("resolve WorkBuddy desktop log: %v", err)
		return
	}
	if err := workbuddymonitor.Watch(ctx, path, func(event workbuddymonitor.Event) {
		if event.Event == "" {
			event.Event = "run_completed"
		}
		if event.Body == "" {
			event.Body = notify.DefaultBody(event.Event)
		}
		msg := notify.Message{
			Agent: "workbuddy", Event: event.Event, SessionID: event.SessionID,
			Workspace: event.Workspace, Title: notify.FormatTitle("workbuddy", event.Event), Body: event.Body, Origin: "desktop_monitor",
		}
		a.dispatchDesktopMessage(msg)
	}); err != nil && !os.IsNotExist(err) {
		log.Printf("watch WorkBuddy desktop log: %v", err)
	}
}

func (a *App) dispatchDesktopMessage(msg notify.Message) {
	cfg, err := a.service.GetConfig()
	if err != nil {
		log.Printf("read config for desktop event: %v", err)
		return
	}
	statePath, err := config.StatePath()
	if err != nil {
		log.Printf("resolve state path for desktop event: %v", err)
		return
	}
	logPath, err := config.LogPath()
	if err != nil {
		log.Printf("resolve log path for desktop event: %v", err)
		return
	}
	if err := agenthooks.Dispatch(context.Background(), cfg, statePath, logPath, msg); err != nil {
		log.Printf("dispatch desktop event: %v", err)
	}
}

func desktopOptions(app *App, frontend fs.FS) *options.App {
	return &options.App{
		Title:             "Agent Notify",
		Width:             980,
		Height:            700,
		StartHidden:       desktopStartsHidden(os.Args),
		HideWindowOnClose: true,
		OnStartup:         app.startTray,
		AssetServer:       &assetserver.Options{Assets: frontend},
		Mac:               &mac.Options{TitleBar: mac.TitleBarHiddenInset()},
		Bind:              []interface{}{app},
	}
}

// desktopStartsHidden keeps login and tray launches unobtrusive while letting
// the deploy command explicitly bring the control window to the foreground.
func desktopStartsHidden(args []string) bool {
	for _, arg := range args[1:] {
		if arg == "--show" {
			return false
		}
	}
	return true
}

func main() {
	statePath, err := config.StatePath()
	if err != nil {
		log.Printf("resolve desktop instance lock: %v", err)
		return
	}
	instanceLock := acquireDesktopInstance(desktopInstanceLockPath(statePath))
	if instanceLock == nil {
		log.Printf("another Agent Notify desktop instance is already running")
		return
	}
	defer instanceLock.Release()

	app, err := NewApp()
	if err != nil {
		panic(err)
	}
	frontend, err := assetRoot()
	if err != nil {
		panic(err)
	}
	if err := wails.Run(desktopOptions(app, frontend)); err != nil {
		log.Printf("desktop app failed: %v", err)
		os.Exit(1)
	}
}
