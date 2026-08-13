package codexmonitor

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type record struct {
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Payload   struct {
		Type    string `json:"type"`
		Message string `json:"message"`
		Phase   string `json:"phase"`
	} `json:"payload"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func ParseFinalAnswer(line string) (string, bool) {
	event, ok := ParseJournalEvent(line)
	if !ok || event.Event != "run_completed" {
		return "", false
	}
	return event.Body, true
}

type Event struct {
	SessionID string
	Event     string
	Body      string
}

// ParseJournalEvent recognizes terminal Codex Desktop outcomes. The desktop UI
// does not invoke CLI hooks, so final answers and task errors come from its
// session journal.
func ParseJournalEvent(line string) (Event, bool) {
	var value record
	if json.Unmarshal([]byte(line), &value) != nil || value.Type != "event_msg" {
		return Event{}, false
	}
	if value.Payload.Type == "agent_message" && value.Payload.Phase == "final_answer" {
		message := strings.TrimSpace(value.Payload.Message)
		return Event{Event: "run_completed", Body: message}, message != ""
	}
	if value.Payload.Type == "task_complete" && value.Error != nil {
		message := strings.TrimSpace(value.Error.Message)
		return Event{Event: "run_failed", Body: message}, message != ""
	}
	return Event{}, false
}

func DefaultSessionsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".codex", "sessions"), nil
}

// Watch scans the official Codex desktop session journal directory and follows
// appended final answers. Codex Desktop does not invoke CLI hooks for UI turns.
func Watch(ctx context.Context, root string, emit func(Event)) error {
	positions := make(map[string]int64)
	initial, err := sessionFiles(root)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	for _, path := range initial {
		if info, statErr := os.Stat(path); statErr == nil {
			positions[path] = info.Size()
		}
	}
	for {
		paths, err := sessionFiles(root)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		for _, path := range paths {
			position := positions[path]
			next, err := readNew(path, position, func(event Event) {
				event.SessionID = sessionID(path)
				emit(event)
			})
			if err == nil {
				positions[path] = next
			}
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(750 * time.Millisecond):
		}
	}
}

func sessionFiles(root string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			return err
		}
		paths = append(paths, path)
		return nil
	})
	return paths, err
}

func readNew(path string, offset int64, emit func(Event)) (int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return offset, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return offset, err
	}
	if offset > info.Size() {
		offset = 0
	}
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return offset, err
	}
	reader := bufio.NewReader(f)
	next := offset
	for {
		line, err := reader.ReadString('\n')
		if line != "" && err == nil {
			next += int64(len(line))
			if event, ok := ParseJournalEvent(strings.TrimSpace(line)); ok {
				emit(event)
			}
		}
		if err == nil {
			continue
		}
		if err == io.EOF {
			return next, nil
		}
		return next, err
	}
}

func sessionID(path string) string {
	base := strings.TrimSuffix(filepath.Base(path), ".jsonl")
	return strings.TrimPrefix(base, "rollout-")
}
