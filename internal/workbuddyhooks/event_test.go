package workbuddyhooks

import (
	"strings"
	"testing"

	"github.com/hellolib/agent-notify/internal/notify"
)

func TestParseMessageMapsPermissionRequest(t *testing.T) {
	msg, err := ParseMessage(strings.NewReader(`{"hook_event_name":"PermissionRequest","session_id":"s1","cwd":"/repo","tool_name":"Bash"}`))
	if err != nil {
		t.Fatal(err)
	}
	if msg.Agent != "workbuddy" || msg.Event != "permission_required" {
		t.Fatalf("message = %#v, want workbuddy permission_required", msg)
	}
	if msg.SessionID != "s1" || msg.Workspace != "/repo" {
		t.Fatalf("context = (%q, %q), want s1 and /repo", msg.SessionID, msg.Workspace)
	}
}

func TestParseMessageMapsNotificationMatchers(t *testing.T) {
	tests := []struct {
		name    string
		matcher string
		want    string
	}{
		{name: "permission", matcher: "permission_prompt", want: "permission_required"},
		{name: "input", matcher: "idle_prompt", want: "input_required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := ParseMessage(strings.NewReader(`{"hook_event_name":"Notification","notification_type":"` + tt.matcher + `","session_id":"s1","cwd":"/repo","message":"hello"}`))
			if err != nil {
				t.Fatal(err)
			}
			if msg.Event != tt.want {
				t.Fatalf("event = %q, want %q", msg.Event, tt.want)
			}
		})
	}
}

func TestParseMessageMapsStopAndFailure(t *testing.T) {
	completed, err := ParseMessage(strings.NewReader(`{"hook_event_name":"Stop","session_id":"s1","cwd":"/repo","last_assistant_message":"done"}`))
	if err != nil {
		t.Fatal(err)
	}
	if completed.Event != "run_completed" || completed.Body != "done" {
		t.Fatalf("completed = %#v", completed)
	}

	failed, err := ParseMessage(strings.NewReader(`{"hook_event_name":"StopFailure","session_id":"s1","cwd":"/repo","error":"broken"}`))
	if err != nil {
		t.Fatal(err)
	}
	if failed.Event != "run_failed" || !strings.Contains(failed.Body, "broken") {
		t.Fatalf("failed = %#v", failed)
	}
}

func TestParseMessageMapsSessionLifecycle(t *testing.T) {
	msg, err := ParseMessage(strings.NewReader(`{"hook_event_name":"SessionStart","session_id":"s1","cwd":"/repo"}`))
	if err != nil {
		t.Fatal(err)
	}
	if msg.Event != "session_start" {
		t.Fatalf("event = %q, want session_start", msg.Event)
	}
	if msg.Title == "" || msg.Body != notify.DefaultBody("session_start") {
		t.Fatalf("message = %#v", msg)
	}
}

