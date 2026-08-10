package workbuddyhooks

import (
	"fmt"
	"io"
	"strings"

	"github.com/hellolib/agent-notify/internal/common"
	"github.com/hellolib/agent-notify/internal/notify"
)

type payload struct {
	HookEventName        string `json:"hook_event_name"`
	SessionID            string `json:"session_id"`
	CWD                  string `json:"cwd"`
	ToolName             string `json:"tool_name"`
	Message              string `json:"message"`
	NotificationType     string `json:"notification_type"`
	LastAssistantMessage string `json:"last_assistant_message"`
	Error                string `json:"error"`
}

func ParseMessage(stdin io.Reader) (notify.Message, error) {
	var p payload
	if err := common.DecodeHookPayload(stdin, &p); err != nil {
		return notify.Message{}, err
	}

	base := func(event, body string) notify.Message {
		return notify.Message{
			Agent:     "workbuddy",
			Event:     event,
			SessionID: p.SessionID,
			Workspace: p.CWD,
			Title:     notify.FormatTitle("workbuddy", event),
			Body:      body,
		}
	}

	switch p.HookEventName {
	case "SessionStart":
		return base("session_start", notify.DefaultBody("session_start")), nil
	case "PermissionRequest":
		return base("permission_required", fmt.Sprintf("工具: %s\n操作需要您的授权许可", fallbackToolName(p.ToolName))), nil
	case "Notification":
		switch strings.ToLower(strings.TrimSpace(p.NotificationType)) {
		case "permission_prompt":
			return base("permission_required", notificationBody(p.Message, "操作需要您的授权许可")), nil
		case "idle_prompt":
			return base("input_required", notificationBody(p.Message, "等待您的输入")), nil
		default:
			return notify.Message{}, fmt.Errorf("unsupported WorkBuddy notification type: %s", p.NotificationType)
		}
	case "Stop":
		body := strings.TrimSpace(p.LastAssistantMessage)
		if body == "" {
			body = notify.DefaultBody("run_completed")
		}
		return base("run_completed", common.TruncateRunes(body, 200)), nil
	case "StopFailure", "PostToolUseFailure":
		body := strings.TrimSpace(p.Error)
		if body == "" {
			body = "操作失败"
		}
		if p.ToolName != "" {
			body = fmt.Sprintf("工具: %s\n错误: %s", p.ToolName, body)
		}
		return base("run_failed", common.TruncateRunes(body, 240)), nil
	default:
		return notify.Message{}, fmt.Errorf("unsupported WorkBuddy hook event: %s", p.HookEventName)
	}
}

func fallbackToolName(name string) string {
	if name = strings.TrimSpace(name); name != "" {
		return name
	}
	return "未知工具"
}

func notificationBody(message, fallback string) string {
	if message = strings.TrimSpace(message); message != "" {
		return common.TruncateRunes(message, 200)
	}
	return fallback
}
