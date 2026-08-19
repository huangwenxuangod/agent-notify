package notify

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

type Message struct {
	EventID     string
	Agent       string
	Event       string
	SessionID   string
	TurnID      string
	RunID       string
	SourceEvent string
	Workspace   string
	Title       string
	Body        string
	Origin      string
	SourceApp   SourceApp
	// FocusWindowID 是 Linux 点击聚焦的目标 X11 窗口 ID（十进制字符串），
	// 由 dispatch 依据 SessionStart 缓存填充；为空则回退按进程树定位。仅 Linux 使用。
	FocusWindowID string
	// FocusCapture 是 macOS / Windows 在 SessionStart 时刻抓取的窗口快照（各平台自己的
	// 原始 JSON），由 dispatch 依据 session 缓存填充。非空时优先用它构造点击载荷，避免
	// 发通知时用户已切走窗口导致抓错（send-time 抓取会取到已漂移的当前焦点窗）。仅 macOS /
	// Windows 使用。
	FocusCapture string
}

// StableEventID creates a deterministic identity shared by hooks, monitors,
// the event journal, and remote retries.
func StableEventID(msg Message) string {
	identity := msg.RunID
	if identity == "" {
		identity = msg.TurnID
	}
	if identity == "" {
		identity = msg.SessionID
	}
	parts := []string{msg.Agent, msg.Event, identity}
	if identity == "" {
		parts = append(parts, msg.Title, msg.Body)
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:])
}

// SourceApp 描述触发事件的宿主应用（终端 / IDE），用于系统通知点击后跳转聚焦。
type SourceApp struct {
	BundleID         string // macOS bundle identifier，激活目标（主信号解析结果）
	TermProgram      string // TERM_PROGRAM 原始值，诊断/扩展用
	TerminalEmulator string // TERMINAL_EMULATOR 原始值，诊断/扩展用
}

type Sender interface {
	Name() string
	Send(ctx context.Context, msg Message) error
}
