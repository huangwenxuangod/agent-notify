package pihooks

import (
	"fmt"
	"io"
	"strings"

	"github.com/hellolib/agent-notify/internal/common"
	"github.com/hellolib/agent-notify/internal/notify"
)

type payload struct {
	Event      string `json:"event"`
	SessionID  string `json:"session_id"`
	TurnID     string `json:"turn_id"`
	RunID      string `json:"run_id"`
	Workspace  string `json:"cwd"`
	Reason     string `json:"reason"`
	Message    string `json:"message"`
	Error      string `json:"error"`
	StopReason string `json:"stop_reason"`
}

func ParseMessage(r io.Reader) (notify.Message, error) {
	var p payload
	if err := common.DecodeHookPayload(r, &p); err != nil {
		return notify.Message{}, err
	}
	event := strings.TrimSpace(p.Event)
	normalized := ""
	switch event {
	case "session_start":
		normalized = "session_start"
	case "agent_end":
		if strings.TrimSpace(p.Error) != "" || strings.EqualFold(strings.TrimSpace(p.StopReason), "error") {
			normalized = "run_failed"
		} else {
			normalized = "run_completed"
		}
	case "session_shutdown":
		if !strings.EqualFold(strings.TrimSpace(p.Reason), "quit") {
			return notify.Message{}, fmt.Errorf("ignore pi session shutdown reason=%s", p.Reason)
		}
		normalized = "run_failed"
	default:
		return notify.Message{}, fmt.Errorf("unsupported pi event: %s", event)
	}

	body := strings.TrimSpace(p.Message)
	if body == "" {
		body = strings.TrimSpace(p.Error)
	}
	if body == "" {
		body = notify.DefaultBody(normalized)
	}
	return notify.Message{
		Agent: "pi", Event: normalized, SessionID: p.SessionID, TurnID: p.TurnID, RunID: p.RunID,
		SourceEvent: event, Workspace: p.Workspace, Title: notify.FormatTitle("pi", normalized), Body: common.TruncateRunes(body, 200),
	}, nil
}
