package codexmonitor

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var rolloutSessionPattern = regexp.MustCompile(`(?i)([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})$`)

type record struct {
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Payload   struct {
		Type             string `json:"type"`
		Message          string `json:"message"`
		Phase            string `json:"phase"`
		TurnID           string `json:"turn_id"`
		LastAgentMessage string `json:"last_agent_message"`
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
	TurnID    string
	Event     string
	Body      string
}

// ParseJournalEvent recognizes terminal Codex Desktop outcomes. Individual
// agent_message records are intermediate responses; task_complete is the
// terminal boundary and includes the final assistant message.
func ParseJournalEvent(line string) (Event, bool) {
	var value record
	if json.Unmarshal([]byte(line), &value) != nil || value.Type != "event_msg" {
		return Event{}, false
	}
	if value.Payload.Type != "task_complete" {
		return Event{}, false
	}
	if value.Error != nil {
		message := strings.TrimSpace(value.Error.Message)
		return Event{TurnID: value.Payload.TurnID, Event: "run_failed", Body: message}, message != ""
	}
	message := strings.TrimSpace(value.Payload.LastAgentMessage)
	if IsInternalControlPayload(message) {
		return Event{}, false
	}
	return Event{TurnID: value.Payload.TurnID, Event: "run_completed", Body: message}, message != ""
}

// IsInternalControlPayload identifies Codex UI bookkeeping completions. These
// are separate background turns (not duplicate deliveries) and do not describe
// a user-facing agent result.
func IsInternalControlPayload(message string) bool {
	var object map[string]json.RawMessage
	if json.Unmarshal([]byte(strings.TrimSpace(message)), &object) != nil || len(object) != 1 {
		return false
	}
	if excluded, ok := object["exclude"]; ok {
		return strings.TrimSpace(string(excluded)) == "[]"
	}
	suggestions, ok := object["suggestions"]
	if !ok {
		return false
	}
	var entries []struct {
		Title       string  `json:"title"`
		Description string  `json:"description"`
		Prompt      string  `json:"prompt"`
		AppID       *string `json:"appId"`
		PluginID    *string `json:"pluginId"`
	}
	if json.Unmarshal(suggestions, &entries) != nil || len(entries) == 0 {
		return false
	}
	for _, entry := range entries {
		if entry.Description == "" || entry.Prompt == "" || (entry.Title == "" && entry.AppID == nil && entry.PluginID == nil) {
			return false
		}
	}
	return true
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
	files := make(map[string]os.FileInfo)
	emittedTurns := make(map[string]bool)
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
			info, statErr := os.Stat(path)
			if statErr == nil {
				if previous, ok := files[path]; ok && !os.SameFile(previous, info) {
					positions[path] = 0
				}
				files[path] = info
			}
			position := positions[path]
			next, err := readNew(path, position, func(event Event) {
				event.SessionID = sessionID(path)
				if event.TurnID != "" {
					key := event.SessionID + "\x00" + event.TurnID
					if emittedTurns[key] {
						return
					}
					emittedTurns[key] = true
				}
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
		return info.Size(), nil
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
	base = strings.TrimPrefix(base, "rollout-")
	if match := rolloutSessionPattern.FindStringSubmatch(base); len(match) == 2 {
		return match[1]
	}
	return base
}
