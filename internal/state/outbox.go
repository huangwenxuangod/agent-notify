package state

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/hellolib/agent-notify/internal/common"
)

// RemoteOutboxItem is a host-local retry record. It deliberately stores only
// the normalized event fields so the state package stays independent of notify.
type RemoteOutboxItem struct {
	ID          string    `json:"id"`
	Agent       string    `json:"agent"`
	Event       string    `json:"event"`
	SessionID   string    `json:"session_id,omitempty"`
	TurnID      string    `json:"turn_id,omitempty"`
	RunID       string    `json:"run_id,omitempty"`
	SourceEvent string    `json:"source_event,omitempty"`
	Workspace   string    `json:"workspace,omitempty"`
	Title       string    `json:"title,omitempty"`
	Body        string    `json:"body,omitempty"`
	Channels    []string  `json:"channels"`
	Attempts    int       `json:"attempts"`
	NextTry     time.Time `json:"next_try"`
	LastError   string    `json:"last_error,omitempty"`
}

type RemoteOutbox struct{ path string }

func NewRemoteOutbox(path string) *RemoteOutbox { return &RemoteOutbox{path: path} }

func (o *RemoteOutbox) Enqueue(item RemoteOutboxItem) error {
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	if item.NextTry.IsZero() {
		item.NextTry = time.Now().UTC()
	}
	items, err := o.load()
	if err != nil {
		return err
	}
	items = append(items, item)
	return o.save(items)
}

func (o *RemoteOutbox) Due(now time.Time) ([]RemoteOutboxItem, error) {
	items, err := o.load()
	if err != nil {
		return nil, err
	}
	due := make([]RemoteOutboxItem, 0, len(items))
	for _, item := range items {
		if !item.NextTry.After(now) {
			due = append(due, item)
		}
	}
	return due, nil
}

func (o *RemoteOutbox) List() ([]RemoteOutboxItem, error) { return o.load() }

func (o *RemoteOutbox) Remove(id string) error {
	items, err := o.load()
	if err != nil {
		return err
	}
	kept := items[:0]
	for _, item := range items {
		if item.ID != id {
			kept = append(kept, item)
		}
	}
	return o.save(kept)
}

func (o *RemoteOutbox) Reschedule(item RemoteOutboxItem, errText string, now time.Time) error {
	items, err := o.load()
	if err != nil {
		return err
	}
	for i := range items {
		if items[i].ID != item.ID {
			continue
		}
		items[i].Attempts++
		items[i].LastError = errText
		items[i].NextTry = now.Add(retryDelay(items[i].Attempts)).UTC()
		break
	}
	return o.save(items)
}

func retryDelay(attempts int) time.Duration {
	if attempts < 1 {
		return time.Minute
	}
	delay := time.Minute << min(attempts-1, 5)
	if delay > 30*time.Minute {
		return 30 * time.Minute
	}
	return delay
}

func (o *RemoteOutbox) load() ([]RemoteOutboxItem, error) {
	data, err := os.ReadFile(o.path)
	if errors.Is(err, os.ErrNotExist) {
		return []RemoteOutboxItem{}, nil
	}
	if err != nil {
		return nil, err
	}
	var items []RemoteOutboxItem
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (o *RemoteOutbox) save(items []RemoteOutboxItem) error {
	data, err := json.Marshal(items)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(o.path), 0o700); err != nil {
		return err
	}
	return common.WriteFileAtomic(o.path, append(data, '\n'), 0o600)
}

func RemoteOutboxPath(statePath string) string {
	return filepath.Join(filepath.Dir(statePath), "remote-outbox.json")
}
