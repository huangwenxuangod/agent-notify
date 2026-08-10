package openclawhooks

import (
	"strings"
	"testing"
)

func TestParseOpenClawToolError(t *testing.T) {
	m, err := ParseMessage(strings.NewReader(`{"event":"tool_error","error":"bad"}`))
	if err != nil {
		t.Fatal(err)
	}
	if m.Agent != "openclaw" || m.Event != "run_failed" || m.Body != "bad" {
		t.Fatalf("%+v", m)
	}
}
