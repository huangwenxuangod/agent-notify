package workbuddymonitor

import (
	"bufio"
	"context"
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

	if !(strings.Contains(line, "reason=session_end_turn") || strings.Contains(line, "session_end_turn:")) || p.emitted[sessionID] {
		return Event{}, false
	}
	p.emitted[sessionID] = true
	return Event{SessionID: sessionID, Workspace: current.workspace, Body: current.title}, true
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
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Seek(0, os.SEEK_END); err != nil {
		return err
	}

	parser := NewParser()
	reader := bufio.NewReader(f)
	for {
		line, err := reader.ReadString('\n')
		if line != "" {
			if event, ok := parser.Consume(strings.TrimSpace(line)); ok {
				emit(event)
			}
		}
		if err == nil {
			continue
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(500 * time.Millisecond):
		}
	}
}

func DefaultLogPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "Logs", "WorkBuddy", "main.log"), nil
}
