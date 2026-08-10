package hermeshooks

import (
	"context"
	"fmt"
	"github.com/hellolib/agent-notify/internal/agenthooks"
	"github.com/hellolib/agent-notify/internal/config"
	"github.com/hellolib/agent-notify/internal/state"
	"io"
)

func Handle(ctx context.Context, cfg config.Config, statePath, logPath string, r io.Reader) error {
	msg, err := ParseMessage(r)
	if err != nil {
		return state.AppendLog(logPath, fmt.Sprintf("skip Hermes event: %v", err))
	}
	return agenthooks.Dispatch(ctx, cfg, statePath, logPath, msg)
}
