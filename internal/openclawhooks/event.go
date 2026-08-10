package openclawhooks

import (
	"fmt"
	"github.com/hellolib/agent-notify/internal/common"
	"github.com/hellolib/agent-notify/internal/notify"
	"io"
	"strings"
)

type payload struct {
	Event     string `json:"event"`
	SessionID string `json:"session_id"`
	Workspace string `json:"workspace"`
	Message   string `json:"message"`
	Error     string `json:"error"`
}

func ParseMessage(r io.Reader) (notify.Message, error) {
	var p payload
	if err := common.DecodeHookPayload(r, &p); err != nil {
		return notify.Message{}, err
	}
	e := strings.TrimSpace(p.Event)
	event := ""
	switch e {
	case "gateway_start", "session_start", "agent_start":
		event = "session_start"
	case "gateway_stop", "agent_end", "run_completed", "session_idle":
		event = "run_completed"
	case "model_error", "tool_error", "agent_error", "run_failed":
		event = "run_failed"
	case "approval_required", "permission_required":
		event = "permission_required"
	default:
		return notify.Message{}, fmt.Errorf("unsupported OpenClaw event: %s", e)
	}
	body := strings.TrimSpace(p.Message)
	if body == "" {
		body = strings.TrimSpace(p.Error)
	}
	if body == "" {
		body = notify.DefaultBody(event)
	}
	return notify.Message{Agent: "openclaw", Event: event, SessionID: p.SessionID, Workspace: p.Workspace, Title: notify.FormatTitle("openclaw", event), Body: body}, nil
}
