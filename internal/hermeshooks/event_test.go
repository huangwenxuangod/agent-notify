package hermeshooks

import (
	"strings"
	"testing"
)

func TestParseHermesAgentEnd(t *testing.T) {
	m, err := ParseMessage(strings.NewReader(`{"event":"agent:end","session_id":"s1"}`))
	if err != nil {
		t.Fatal(err)
	}
	if m.Agent != "hermes" || m.Event != "run_completed" || m.SessionID != "s1" {
		t.Fatalf("%+v", m)
	}
}
