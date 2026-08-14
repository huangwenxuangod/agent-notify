package state

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hellolib/agent-notify/internal/common"
)

// EventRecord is deliberately independent from notify.Message to avoid a package cycle.
type EventRecord struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Agent     string    `json:"agent"`
	Event     string    `json:"event"`
	SessionID string    `json:"session_id,omitempty"`
	Workspace string    `json:"workspace,omitempty"`
	Title     string    `json:"title,omitempty"`
	Body      string    `json:"body,omitempty"`
	Origin    string    `json:"origin,omitempty"`
	SourceApp string    `json:"source_app,omitempty"`
	Result    string    `json:"result"`
}

type EventJournal struct {
	path     string
	maxBytes int64
	mu       sync.Mutex
}

func NewEventJournal(path string, maxBytes int64) *EventJournal {
	if maxBytes < 1 {
		maxBytes = 5 << 20
	}
	return &EventJournal{path: path, maxBytes: maxBytes}
}

func (j *EventJournal) Append(record EventRecord) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(j.path), 0o700); err != nil {
		return err
	}
	if record.ID == "" {
		record.ID = uuid.NewString()
	}
	if record.Timestamp.IsZero() {
		record.Timestamp = time.Now().UTC()
	}
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	lock := common.AcquireFileLock(j.path+".lock", lockTimeout)
	defer lock.Release()
	if lock == nil {
		return errors.New("event journal lock timeout")
	}
	if info, statErr := os.Stat(j.path); statErr == nil && info.Size() > 0 && info.Size()+int64(len(data)) > j.maxBytes {
		_ = os.Remove(j.path + ".1")
		if err := os.Rename(j.path, j.path+".1"); err != nil {
			return err
		}
	}
	f, err := os.OpenFile(j.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := f.Chmod(0o600); err != nil {
		return err
	}
	_, err = f.Write(data)
	return err
}

func EventJournalPath(statePath string) string {
	return filepath.Join(filepath.Dir(statePath), "events.jsonl")
}

func (j *EventJournal) List(limit int) ([]EventRecord, error) {
	if limit <= 0 {
		limit = 100
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	var records []EventRecord
	for _, path := range []string{j.path + ".1", j.path} {
		f, err := os.Open(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			var record EventRecord
			if json.Unmarshal(scanner.Bytes(), &record) == nil {
				records = append(records, record)
			}
		}
		scanErr := scanner.Err()
		_ = f.Close()
		if scanErr != nil {
			return nil, scanErr
		}
	}
	if len(records) > limit {
		records = records[len(records)-limit:]
	}
	return records, nil
}
