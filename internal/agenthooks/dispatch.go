package agenthooks

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/hellolib/agent-notify/internal/bridgeclient"
	"github.com/hellolib/agent-notify/internal/config"
	"github.com/hellolib/agent-notify/internal/linuxfocus"
	"github.com/hellolib/agent-notify/internal/notify"
	"github.com/hellolib/agent-notify/internal/state"
	"github.com/hellolib/agent-notify/internal/winfocus"
)

func Dispatch(ctx context.Context, cfg config.Config, statePath, logPath string, msg notify.Message) error {
	return dispatch(ctx, cfg, statePath, logPath, msg, true)
}

// DispatchRemote is used by the Docker control plane. It deliberately excludes
// system notifications because those must run in the host desktop session.
func DispatchRemote(ctx context.Context, cfg config.Config, statePath, logPath string, msg notify.Message) error {
	return dispatch(ctx, cfg, statePath, logPath, msg, false)
}

func dispatch(ctx context.Context, cfg config.Config, statePath, logPath string, msg notify.Message, host bool) error {
	// hook 进程由终端 / IDE 启动，此处能从继承的环境变量识别宿主应用
	if host {
		msg.SourceApp = notify.DetectSourceApp()
	}
	// Windows 上 hook stdin 偶发把路径中的中文变成 '?'；用 env / Getwd 纠正
	msg.Workspace = notify.ResolveWorkspace(msg.Workspace)
	// session_start 是纯副作用事件：仅在 Linux / macOS / Windows 捕获点击聚焦的目标窗口并缓存，
	// 永不产生通知，也不受任何 agent 的事件配置控制。其它平台直接返回。
	if msg.Event == "session_start" {
		if host {
			captureFocusWindow(ctx, statePath, logPath, msg)
		}
		return nil
	}

	// 非 session_start：若命中 SessionStart 缓存，填充精确窗口供点击聚焦复用。
	// Linux 存的是 X11 window id（→ FocusWindowID）；macOS / Windows 存的是窗口快照
	// JSON（→ FocusCapture）：mac 是 mac-focus-helper --capture 的输出，Windows 是
	// winfocus 的 {"hwnd","title"}。
	if data, ok := state.NewFocusStore(state.FocusWindowsPath(statePath)).Load(msg.SessionID); ok {
		applyFocusCache(&msg, runtime.GOOS, data)
	}

	store := state.NewStore(statePath)
	timeout := time.Duration(cfg.Behavior.SendTimeoutSeconds) * time.Second
	systemDelivered := false
	if host {
		if sender := buildSystemSender(cfg, msg); sender != nil {
			sendCtx, cancel := context.WithTimeout(ctx, systemSendTimeout(timeout))
			if err := notify.NewDispatcher(store, time.Duration(cfg.Behavior.DedupeSeconds)*time.Second, sender).SendAll(sendCtx, msg); err != nil {
				_ = state.AppendLog(logPath, fmt.Sprintf("system dispatch error event=%s err=%v", msg.Event, err))
			} else {
				systemDelivered = true
			}
			cancel()
		}
	}

	senders := buildRemoteSenders(cfg, msg)
	if len(senders) == 0 {
		result := overallDeliveryResult(systemDelivered, nil, false)
		appendEventRecord(statePath, logPath, msg, result)
		recordBridgeEvent(ctx, host, logPath, msg, result)
		if result == "no_sender" {
			return state.AppendLog(logPath, fmt.Sprintf("no sender enabled for event=%s", msg.Event))
		}
		return nil
	}

	// 临时冻结：在 ReserveSend 之前按渠道静默丢弃，不占去重名额、不改 agent hooks。
	senders = filterFrozenSenders(statePath, logPath, msg.Event, senders, time.Now())
	if len(senders) == 0 {
		appendEventRecord(statePath, logPath, msg, "frozen")
		recordBridgeEvent(ctx, host, logPath, msg, "frozen")
		return state.AppendLog(logPath, fmt.Sprintf("all senders frozen for event=%s", msg.Event))
	}

	dispatcher := notify.NewDispatcher(store, time.Duration(cfg.Behavior.DedupeSeconds)*time.Second, senders...)
	sendCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := dispatcher.SendAll(sendCtx, msg); err != nil {
		result := overallDeliveryResult(systemDelivered, err, true)
		enqueueRemoteRetry(statePath, logPath, msg, failedSenderNames(err, senders), err)
		appendEventRecord(statePath, logPath, msg, result)
		recordBridgeEvent(ctx, host, logPath, msg, result)
		return state.AppendLog(logPath, fmt.Sprintf("dispatch error event=%s session=%s err=%v", msg.Event, msg.SessionID, err))
	}
	result := overallDeliveryResult(systemDelivered, nil, true)
	appendEventRecord(statePath, logPath, msg, result)
	recordBridgeEvent(ctx, host, logPath, msg, result)

	return nil
}

func overallDeliveryResult(systemDelivered bool, remoteErr error, hadRemote bool) string {
	if remoteErr == nil && (systemDelivered || hadRemote) {
		return "sent"
	}
	if systemDelivered {
		return "partial"
	}
	if remoteErr != nil {
		return "error"
	}
	return "no_sender"
}

func failedSenderNames(err error, senders []notify.Sender) []string {
	var delivery *notify.DeliveryError
	if !errors.As(err, &delivery) {
		return senderNames(senders)
	}
	failed := make(map[string]bool)
	for _, detail := range delivery.Details {
		if name, _, ok := strings.Cut(detail, ":"); ok {
			failed[strings.TrimSpace(name)] = true
		}
	}
	names := make([]string, 0, len(failed))
	for name := range failed {
		names = append(names, name)
	}
	return names
}

func senderNames(senders []notify.Sender) []string {
	names := make([]string, 0, len(senders))
	for _, sender := range senders {
		names = append(names, sender.Name())
	}
	return names
}

func enqueueRemoteRetry(statePath, logPath string, msg notify.Message, channels []string, err error) {
	if len(channels) == 0 {
		return
	}
	item := state.RemoteOutboxItem{Agent: msg.Agent, Event: msg.Event, SessionID: msg.SessionID, Workspace: msg.Workspace, Title: msg.Title, Body: msg.Body, Channels: channels, LastError: err.Error()}
	if saveErr := state.NewRemoteOutbox(state.RemoteOutboxPath(statePath)).Enqueue(item); saveErr != nil {
		_ = state.AppendLog(logPath, fmt.Sprintf("remote retry enqueue error: %v", saveErr))
	}
}

func deliveryResult(err error) string {
	if notify.HasSuccessfulDelivery(err) {
		return "partial"
	}
	return "error"
}

func recordBridgeEvent(ctx context.Context, host bool, logPath string, msg notify.Message, result string) {
	if !host {
		return
	}
	if _, err := bridgeclient.RecordEvent(ctx, "", "", msg, result); err != nil {
		_ = state.AppendLog(logPath, fmt.Sprintf("bridge event record failed: %v", err))
	}
}

// systemSendTimeout keeps native macOS notification helpers enough time to
// launch. Remote delivery continues to honor Behavior.SendTimeoutSeconds.
func systemSendTimeout(timeout time.Duration) time.Duration {
	const minimum = 15 * time.Second
	if timeout < minimum {
		return minimum
	}
	return timeout
}

func appendEventRecord(statePath, logPath string, msg notify.Message, result string) {
	sourceApp := msg.SourceApp.BundleID
	if sourceApp == "" {
		sourceApp = msg.SourceApp.TermProgram
	}
	record := state.EventRecord{
		Timestamp: time.Now().UTC(), Agent: msg.Agent, Event: msg.Event,
		SessionID: msg.SessionID, Workspace: msg.Workspace, Title: msg.Title,
		Body: msg.Body, SourceApp: sourceApp, Result: result,
	}
	if err := state.NewEventJournal(state.EventJournalPath(statePath), 5<<20).Append(record); err != nil {
		_ = state.AppendLog(logPath, fmt.Sprintf("event journal append error event=%s err=%v", msg.Event, err))
	}
}

func applyFocusCache(msg *notify.Message, goos, data string) {
	switch goos {
	case "darwin", "windows":
		msg.FocusCapture = data
	case "linux":
		msg.FocusWindowID = data
	}
}

// captureFocusWindow 在 SessionStart 时刻抓取"启动本 agent 的那个窗口"并按 session 缓存，
// 供后续事件（permission/input/completed…）点击聚焦复用，避免发通知时用户已切走导致抓错窗口。
// Linux 用 EWMH 抓 active window（经 PID 祖先校验）；macOS 用 mac-focus-helper --capture 走进程树定位；
// Windows 用 winfocus 抓前台窗（经 PID 祖先校验）。抓取失败（如焦点在别的应用、helper 缺失）时
// 不覆盖已有缓存。其它平台不做。
func captureFocusWindow(ctx context.Context, statePath, logPath string, msg notify.Message) {
	if msg.SessionID == "" {
		return
	}
	var windowData string
	switch runtime.GOOS {
	case "linux":
		windowID, err := linuxfocus.CaptureActiveWindow(ctx)
		if err != nil || windowID == "" {
			return
		}
		windowData = windowID
	case "darwin":
		capture, err := notify.CaptureMacWindow()
		if err != nil || capture == "" {
			return
		}
		windowData = capture
	case "windows":
		capture, err := winfocus.Capture()
		if err != nil || capture == "" {
			return
		}
		windowData = capture
	default:
		return
	}
	if err := state.NewFocusStore(state.FocusWindowsPath(statePath)).Save(msg.SessionID, windowData, time.Now()); err != nil {
		_ = state.AppendLog(logPath, fmt.Sprintf("focus capture save error session=%s err=%v", msg.SessionID, err))
	}
}

// filterFrozenSenders 按 freeze.json 剔除被冻渠道；未冻结或读失败时原样返回。
func filterFrozenSenders(statePath, logPath, event string, senders []notify.Sender, now time.Time) []notify.Sender {
	freeze := state.NewFreezeStore(state.FreezePath(statePath)).Load()
	if !freeze.Active(now) {
		return senders
	}
	filtered := senders[:0]
	for _, s := range senders {
		if freeze.Blocks(s.Name(), now) {
			_ = state.AppendLog(logPath, fmt.Sprintf(
				"freeze skip sender=%s until=%s event=%s",
				s.Name(), freeze.Until.Format(time.RFC3339), event,
			))
			continue
		}
		filtered = append(filtered, s)
	}
	return filtered
}

func buildRemoteSenders(cfg config.Config, msg notify.Message) []notify.Sender {
	var senders []notify.Sender

	notifyCfg := cfg.Notify.ClaudeCode
	switch msg.Agent {
	case "codex":
		notifyCfg = cfg.Notify.Codex
	case "zcode":
		notifyCfg = cfg.Notify.ZCode
	case "grok":
		notifyCfg = cfg.Notify.Grok
	case "droid":
		notifyCfg = cfg.Notify.Droid
	case "opencode":
		notifyCfg = cfg.Notify.OpenCode
	case "workbuddy":
		notifyCfg = cfg.Notify.WorkBuddy
	case "hermes":
		notifyCfg = cfg.Notify.Hermes
	case "openclaw":
		notifyCfg = cfg.Notify.OpenClaw
	}

	if !contains(notifyCfg.Events, msg.Event) {
		return senders
	}

	remote := cfg.Remote
	if remote.Feishu.Enabled && remote.Feishu.WebhookURL != "" {
		senders = append(senders, notify.NewFeishuWebhookSenderWithSecret(remote.Feishu.WebhookURL, remote.Feishu.SigningSecret))
	}
	if remote.Wechat.Enabled && remote.Wechat.WebhookURL != "" {
		senders = append(senders, notify.NewWechatSender(remote.Wechat.WebhookURL))
	}
	if remote.WechatWork.Enabled && remote.WechatWork.WebhookURL != "" {
		senders = append(senders, notify.NewWechatWorkSender(remote.WechatWork.WebhookURL))
	}
	if remote.DingTalk.Enabled && remote.DingTalk.WebhookURL != "" {
		senders = append(senders, notify.NewDingTalkSenderWithSecret(remote.DingTalk.WebhookURL, remote.DingTalk.SigningSecret))
	}
	if remote.Bark.Enabled && remote.Bark.WebhookURL != "" {
		senders = append(senders, notify.NewBarkSender(remote.Bark.WebhookURL))
	}
	if remote.Ntfy.Enabled && remote.Ntfy.TopicURL != "" {
		senders = append(senders, notify.NewNtfySenderWithAccessToken(remote.Ntfy.TopicURL, remote.Ntfy.AccessToken))
	}
	if remote.Slack.Enabled && remote.Slack.WebhookURL != "" {
		senders = append(senders, notify.NewSlackSender(remote.Slack.WebhookURL))
	}

	return senders
}

// buildSenders preserves the legacy per-Agent configuration shape for local
// fallback callers and compatibility tests. Docker dispatch uses
// buildRemoteSenders directly and therefore never includes system delivery.
func buildSenders(cfg config.Config, msg notify.Message) []notify.Sender {
	legacy := cfg
	notifyCfg := cfg.Notify.ClaudeCode
	switch msg.Agent {
	case "codex":
		notifyCfg = cfg.Notify.Codex
	case "zcode":
		notifyCfg = cfg.Notify.ZCode
	case "grok":
		notifyCfg = cfg.Notify.Grok
	case "droid":
		notifyCfg = cfg.Notify.Droid
	case "opencode":
		notifyCfg = cfg.Notify.OpenCode
	case "workbuddy":
		notifyCfg = cfg.Notify.WorkBuddy
	case "hermes":
		notifyCfg = cfg.Notify.Hermes
	case "openclaw":
		notifyCfg = cfg.Notify.OpenClaw
	}
	legacyFeishuURL := ""
	if notifyCfg.Channels.Feishu.Enabled {
		legacyFeishuURL = "legacy://feishu"
	}
	legacy.Remote = config.RemoteDeliveryConfig{
		Feishu: config.FeishuWebhookConfig{Enabled: notifyCfg.Channels.Feishu.Enabled, WebhookURL: legacyFeishuURL}, Wechat: notifyCfg.Channels.Wechat,
		WechatWork: notifyCfg.Channels.WechatWork, DingTalk: notifyCfg.Channels.DingTalk,
		Bark: notifyCfg.Channels.Bark, Ntfy: notifyCfg.Channels.Ntfy, Slack: notifyCfg.Channels.Slack,
	}
	senders := buildRemoteSenders(legacy, msg)
	if system := buildSystemSender(cfg, msg); system != nil {
		return append([]notify.Sender{system}, senders...)
	}
	return senders
}

func buildSystemSender(cfg config.Config, msg notify.Message) notify.Sender {
	notifyCfg := cfg.Notify.ClaudeCode
	switch msg.Agent {
	case "codex":
		notifyCfg = cfg.Notify.Codex
	case "zcode":
		notifyCfg = cfg.Notify.ZCode
	case "grok":
		notifyCfg = cfg.Notify.Grok
	case "droid":
		notifyCfg = cfg.Notify.Droid
	case "opencode":
		notifyCfg = cfg.Notify.OpenCode
	case "workbuddy":
		notifyCfg = cfg.Notify.WorkBuddy
	case "hermes":
		notifyCfg = cfg.Notify.Hermes
	case "openclaw":
		notifyCfg = cfg.Notify.OpenClaw
	}
	if !contains(notifyCfg.Events, msg.Event) || !notifyCfg.Channels.System.Enabled {
		return nil
	}
	return notify.NewSystemSender(notify.DefaultRunner, notifyCfg.Channels.System.ClickToFocus, config.FocusPrecisionFromEnv(), notifyCfg.Channels.System.EffectiveFocusDebug())
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
