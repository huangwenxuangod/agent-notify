package codexmonitor

import (
	"bufio"
	"context"
	"encoding/json"
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
}

func ParseFinalAnswer(line string) (string, bool) {
	var value record
	if json.Unmarshal([]byte(line), &value) != nil || value.Type != "event_msg" || value.Payload.Type != "agent_message" || value.Payload.Phase != "final_answer" {
		return "", false
	}
	message := strings.TrimSpace(value.Payload.Message)
	return message, message != ""
}

type Event struct {
	SessionID string
	Body      string
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
			next, err := readNew(path, position, func(message string) {
				emit(Event{SessionID: sessionID(path), Body: message})
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

func readNew(path string, offset int64, emit func(string)) (int64, error) {
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
	if _, err := f.Seek(offset, os.SEEK_SET); err != nil {
		return offset, err
	}
	reader := bufio.NewScanner(f)
	for reader.Scan() {
		if message, ok := ParseFinalAnswer(reader.Text()); ok {
			emit(message)
		}
	}
	if err := reader.Err(); err != nil {
		return offset, err
	}
	next, err := f.Seek(0, os.SEEK_CUR)
	return next, err
}

func sessionID(path string) string {
	base := strings.TrimSuffix(filepath.Base(path), ".jsonl")
	return strings.TrimPrefix(base, "rollout-")
}
