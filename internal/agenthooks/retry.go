package agenthooks

import (
	"context"
	"fmt"
	"strings"
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
	if len(items) == 0 {
		return 0, nil
	}
	// Process one record per tick. Remote providers can rate-limit a burst of
	// historical notifications even when each item has its own retry delay.
	item := items[0]
	msg := notify.Message{EventID: item.EventID, Agent: item.Agent, Event: item.Event, SessionID: item.SessionID, TurnID: item.TurnID, RunID: item.RunID, SourceEvent: item.SourceEvent, Workspace: item.Workspace, Title: item.Title, Body: item.Body}
	senders := selectSenders(buildRemoteSenders(cfg, msg), item.Channels)
	if len(senders) == 0 {
		if err := outbox.Remove(item.ID); err != nil {
			return 0, err
		}
		return 1, nil
	}
	if err := send(ctx, msg, senders); err != nil {
		if isPermanentRemotePrecondition(err) {
			if removeErr := outbox.Remove(item.ID); removeErr != nil {
				return 0, removeErr
			}
			return 1, nil
		}
		failed := failedSenderNames(err, senders)
		if len(failed) == 0 {
			if removeErr := outbox.Remove(item.ID); removeErr != nil {
				return 0, removeErr
			}
			return 1, nil
		}
		item.Channels = failed
		if item.Attempts+1 >= maxRemoteRetryAttempts {
			if removeErr := outbox.Remove(item.ID); removeErr != nil {
				return 0, removeErr
			}
			return 1, nil
		}
		if saveErr := outbox.Reschedule(item, err.Error(), time.Now()); saveErr != nil {
			return 0, fmt.Errorf("reschedule remote retry: %w", saveErr)
		}
		return 0, nil
	}
	if err := outbox.Remove(item.ID); err != nil {
		return 0, err
	}
	return 1, nil
}

func isPermanentRemotePrecondition(err error) bool {
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "returned 401") ||
		strings.Contains(text, "returned 409") ||
		strings.Contains(text, "prepare failed")
}

const maxRemoteRetryAttempts = 3

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
