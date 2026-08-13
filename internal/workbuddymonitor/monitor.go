package workbuddymonitor

import (
	"bufio"
	"context"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type Event struct {
	SessionID string
	Workspace string
	Body      string
	Event     string
}

type session struct {
	workspace string
	title     string
}

type Parser struct {
	sessions map[string]session
	emitted  map[string]bool
}

var (
	sessionIDPattern = regexp.MustCompile(`\b(?:sid|cid)=([^\s]+)`)
	workspacePattern = regexp.MustCompile(`\bcwd="([^"]+)"`)
	titlePattern     = regexp.MustCompile(`\btitle="([^"]+)"`)
)

func NewParser() *Parser {
	return &Parser{sessions: make(map[string]session), emitted: make(map[string]bool)}
}

func (p *Parser) Consume(line string) (Event, bool) {
	sessionID := capture(sessionIDPattern, line)
	if sessionID == "" {
		return Event{}, false
	}
	current := p.sessions[sessionID]
	if workspace := capture(workspacePattern, line); workspace != "" {
		current.workspace = workspace
	}
	if title := capture(titlePattern, line); title != "" {
		current.title = title
	}
	p.sessions[sessionID] = current

	if p.emitted[sessionID] {
		return Event{}, false
	}
	event := ""
	switch {
	case strings.Contains(line, "reason=session_end_turn") || strings.Contains(line, "session_end_turn:"):
		event = "run_completed"
	case strings.Contains(line, "reason=session_cancelled"), strings.Contains(line, "reason=session_aborted"), strings.Contains(line, "reason=cancelled"):
		event = "run_failed"
	case strings.Contains(line, "[ERROR]"), strings.Contains(strings.ToLower(line), " error="):
		event = "run_failed"
	}
	if event == "" {
		return Event{}, false
	}
	p.emitted[sessionID] = true
	body := current.title
	if event == "run_failed" {
		body = strings.TrimSpace(line)
	}
	return Event{SessionID: sessionID, Workspace: current.workspace, Body: body, Event: event}, true
}

func capture(pattern *regexp.Regexp, line string) string {
	match := pattern.FindStringSubmatch(line)
	if len(match) != 2 {
		return ""
	}
	return match[1]
}

// Watch follows WorkBuddy's desktop log. The UI currently completes ACP
// sessions without invoking CodeBuddy's native Stop hook, so this is the
// narrow compatibility source for those UI-only completions.
func Watch(ctx context.Context, path string, emit func(Event)) error {
	parser := NewParser()
	var f *os.File
	var reader *bufio.Reader
	for {
		if f == nil || logReplaced(path, f) {
			if f != nil {
				_ = f.Close()
			}
			next, err := os.Open(path)
			if err != nil {
				if os.IsNotExist(err) {
					if !wait(ctx) {
						return nil
					}
					continue
				}
				return err
			}
			f = next
			if _, err := f.Stat(); err != nil {
				_ = f.Close()
				return err
			}
			// The first file is pre-existing history and must not replay. A new
			// file after rotation starts at offset zero so no completion is lost.
			if reader != nil {
				if _, err := f.Seek(0, io.SeekStart); err != nil {
					_ = f.Close()
					return err
				}
			} else if _, err := f.Seek(0, io.SeekEnd); err != nil {
				_ = f.Close()
				return err
			}
			reader = bufio.NewReader(f)
		}
		line, err := reader.ReadString('\n')
		if line != "" {
			if event, ok := parser.Consume(strings.TrimSpace(line)); ok {
				emit(event)
			}
		}
		if err == nil {
			continue
		}
		if !wait(ctx) {
			return nil
		}
	}
}

func logReplaced(path string, current *os.File) bool {
	currentInfo, err := current.Stat()
	if err != nil {
		return true
	}
	pathInfo, err := os.Stat(path)
	if err != nil {
		return !os.IsNotExist(err)
	}
	return !os.SameFile(currentInfo, pathInfo)
}

func wait(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(500 * time.Millisecond):
		return true
	}
}

func DefaultLogPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "Logs", "WorkBuddy", "main.log"), nil
}
