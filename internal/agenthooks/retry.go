package agenthooks

import (
	"context"
	"fmt"
	"time"

	"github.com/hellolib/agent-notify/internal/config"
	"github.com/hellolib/agent-notify/internal/notify"
	"github.com/hellolib/agent-notify/internal/state"
)

// RetryRemoteOutbox retries remote channels that failed during a host Hook
// dispatch. The caller owns scheduling so Hooks never wait for retries.
func RetryRemoteOutbox(ctx context.Context, cfg config.Config, statePath string, send func(context.Context, notify.Message, []notify.Sender) error) (int, error) {
	outbox := state.NewRemoteOutbox(state.RemoteOutboxPath(statePath))
	items, err := outbox.Due(time.Now())
	if err != nil {
		return 0, err
	}
	completed := 0
	for _, item := range items {
		msg := notify.Message{Agent: item.Agent, Event: item.Event, SessionID: item.SessionID, Workspace: item.Workspace, Title: item.Title, Body: item.Body}
		senders := selectSenders(buildRemoteSenders(cfg, msg), item.Channels)
		if len(senders) == 0 {
			if err := outbox.Remove(item.ID); err != nil {
				return completed, err
			}
			completed++
			continue
		}
		if err := send(ctx, msg, senders); err != nil {
			if saveErr := outbox.Reschedule(item, err.Error(), time.Now()); saveErr != nil {
				return completed, fmt.Errorf("reschedule remote retry: %w", saveErr)
			}
			continue
		}
		if err := outbox.Remove(item.ID); err != nil {
			return completed, err
		}
		completed++
	}
	return completed, nil
}

func selectSenders(senders []notify.Sender, names []string) []notify.Sender {
	allowed := make(map[string]bool, len(names))
	for _, name := range names {
		allowed[name] = true
	}
	selected := make([]notify.Sender, 0, len(names))
	for _, sender := range senders {
		if allowed[sender.Name()] {
			selected = append(selected, sender)
		}
	}
	return selected
}

// DispatchRemoteOutboxItem applies the same timeout and de-duplication policy
// as a normal remote send without re-enqueueing failures.
func DispatchRemoteOutboxItem(ctx context.Context, cfg config.Config, statePath string, msg notify.Message, senders []notify.Sender) error {
	timeout := time.Duration(cfg.Behavior.SendTimeoutSeconds) * time.Second
	sendCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return notify.NewDispatcher(state.NewStore(statePath), time.Duration(cfg.Behavior.DedupeSeconds)*time.Second, senders...).SendAll(sendCtx, msg)
}
