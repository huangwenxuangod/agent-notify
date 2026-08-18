package state

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hellolib/agent-notify/internal/common"
)

func AppendLog(path, line string) error {
	return appendLogAt(path, line, time.Now())
}

const logRetention = 72 * time.Hour

func appendLogAt(path, line string, now time.Time) error {
	return writeLogAt(path, []byte(line+"\n"), now)
}

// LogWriter adapts child-process stdout/stderr to the same bounded log file.
// A new file descriptor per write avoids stale offsets after rotation.
type LogWriter struct{ path string }

func NewLogWriter(path string) *LogWriter { return &LogWriter{path: path} }

func (w *LogWriter) Write(data []byte) (int, error) {
	return len(data), writeLogAt(w.path, data, time.Now())
}

func writeLogAt(path string, data []byte, now time.Time) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	lock := common.AcquireFileLock(path+".lock", lockTimeout)
	defer lock.Release()
	if lock == nil {
		return errors.New("log file lock timeout")
	}
	if err := rotateLogAt(path, now); err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
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

func rotateLogAt(path string, now time.Time) error {
	marker := path + ".started-at"
	started := now
	markerMissing := false
	if data, err := os.ReadFile(marker); err == nil {
		if parsed, parseErr := time.Parse(time.RFC3339Nano, strings.TrimSpace(string(data))); parseErr == nil {
			started = parsed
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	} else {
		markerMissing = true
	}
	if markerMissing {
		return os.WriteFile(marker, []byte(now.UTC().Format(time.RFC3339Nano)), 0o600)
	}
	if now.Sub(started) < logRetention {
		return nil
	}
	if err := os.WriteFile(path, nil, 0o600); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.WriteFile(marker, []byte(now.UTC().Format(time.RFC3339Nano)), 0o600)
}
