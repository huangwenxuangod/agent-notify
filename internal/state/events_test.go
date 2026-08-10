package state_test

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hellolib/agent-notify/internal/state"
)

func TestEventJournalAppendAndList(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	j := state.NewEventJournal(path, 1024)
	want := state.EventRecord{ID: "evt-1", Timestamp: time.Unix(10, 0).UTC(), Agent: "codex", Event: "run_completed", Result: "sent"}
	if err := j.Append(want); err != nil {
		t.Fatal(err)
	}
	got, err := j.List(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != want.ID || got[0].Agent != "codex" || got[0].Result != "sent" {
		t.Fatalf("records = %#v, want %#v", got, want)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o, want 600", info.Mode().Perm())
	}
}

func TestEventJournalConcurrentAppendProducesCompleteLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	j := state.NewEventJournal(path, 1<<20)
	const n = 20
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if err := j.Append(state.EventRecord{ID: string(rune('a' + i)), Agent: "test"}); err != nil {
				t.Errorf("append %d: %v", i, err)
			}
		}(i)
	}
	wg.Wait()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	lines := 0
	for scanner.Scan() {
		var record state.EventRecord
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			t.Fatalf("invalid JSONL: %v", err)
		}
		lines++
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if lines != n {
		t.Fatalf("lines = %d, want %d", lines, n)
	}
}

func TestEventJournalRotatesAndSkipsMalformedLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	j := state.NewEventJournal(path, 180)
	for i := 0; i < 3; i++ {
		if err := j.Append(state.EventRecord{ID: "event-" + string(rune('0'+i)), Body: strings.Repeat("x", 80)}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(path + ".1"); err != nil {
		t.Fatalf("rotation file missing: %v", err)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString("not-json\n")
	_ = f.Close()
	records, err := j.List(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) == 0 {
		t.Fatal("List returned no valid records")
	}
}
