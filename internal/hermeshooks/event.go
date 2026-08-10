package hermeshooks

import (
	"fmt"
	"io"
	"strings"

	"github.com/hellolib/agent-notify/internal/common"
	"github.com/hellolib/agent-notify/internal/notify"
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
	var event string
	switch e {
	case "agent:start", "session:start", "lifecycle:start":
		event = "session_start"
	case "agent:end", "session:end", "agent:complete":
		event = "run_completed"
	case "agent:error", "tool:error", "agent:failed", "tool:failed":
		event = "run_failed"
	case "approval:required", "input:required", "permission:required":
		event = "permission_required"
	default:
		return notify.Message{}, fmt.Errorf("unsupported hermes event: %s", e)
	}
	body := strings.TrimSpace(p.Message)
	if body == "" {
		body = strings.TrimSpace(p.Error)
	}
	if body == "" {
		body = notify.DefaultBody(event)
	}
	return notify.Message{Agent: "hermes", Event: event, SessionID: p.SessionID, Workspace: p.Workspace,
		Title: notify.FormatTitle("hermes", event), Body: body}, nil
}
