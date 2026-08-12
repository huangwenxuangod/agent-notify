package notify

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNtfySenderName(t *testing.T) {
	s := NewNtfySender("https://ntfy.sh/mytopic")
	if got := s.Name(); got != "ntfy" {
		t.Fatalf("Name() = %q, want ntfy", got)
	}
}

func TestNtfySenderSendEmptyURL(t *testing.T) {
	s := NewNtfySender("")
	err := s.Send(context.Background(), Message{Title: "t", Body: "b"})
	if err == nil {
		t.Fatal("Send() error = nil, want error for empty URL")
	}
	if !strings.Contains(err.Error(), "topic_url is empty") {
		t.Fatalf("Send() error = %v, want topic_url is empty", err)
	}
}

func TestNtfySenderSendSuccess(t *testing.T) {
	var gotBody string
	var gotTitle string
	var gotTags string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/mytopic" {
			t.Errorf("path = %s, want /mytopic", r.URL.Path)
		}
		gotTitle = r.Header.Get("Title")
		gotTags = r.Header.Get("Tags")
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	s := NewNtfySender(srv.URL + "/mytopic")
	msg := Message{Title: "Test Title", Body: "Test Body", Event: "run_completed"}
	if err := s.Send(context.Background(), msg); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if !strings.Contains(gotTitle, "Test Title") {
		t.Fatalf("Title header = %q, want to contain Test Title", gotTitle)
	}
	if gotBody != "Test Body" {
		t.Fatalf("body = %q, want Test Body", gotBody)
	}
	if gotTags == "" {
		t.Fatal("Tags header is empty, want non-empty tags for run_completed")
	}
}

func TestNtfySenderUsesBearerToken(t *testing.T) {
	var authorization string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	if err := NewNtfySenderWithAccessToken(srv.URL+"/topic", "tk_test").Send(context.Background(), Message{Title: "T", Body: "B"}); err != nil {
		t.Fatal(err)
	}
	if authorization != "Bearer tk_test" {
		t.Fatalf("authorization = %q, want bearer token", authorization)
	}
}

func TestNtfySenderSendWithWorkspace(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	s := NewNtfySender(srv.URL + "/mytopic")
	msg := Message{Title: "t", Body: "b", Workspace: "/home/user/project", Agent: "claude_code"}
	if err := s.Send(context.Background(), msg); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if !strings.Contains(gotBody, "/home/user/project") {
		t.Fatalf("body = %q, want to contain workspace path", gotBody)
	}
}

func TestNtfySenderSendHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("forbidden"))
	}))
	defer srv.Close()

	s := NewNtfySender(srv.URL + "/mytopic")
	err := s.Send(context.Background(), Message{Title: "t", Body: "b"})
	if err == nil {
		t.Fatal("Send() error = nil, want HTTP error")
	}
	if !strings.Contains(err.Error(), "unexpected status 403") {
		t.Fatalf("Send() error = %v, want unexpected status 403", err)
	}
}

func TestNtfyEndpoint(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "simple topic on ntfy.sh",
			input: "https://ntfy.sh/mytopic",
			want:  "https://ntfy.sh/mytopic",
		},
		{
			name:  "topic with extra path segments",
			input: "https://ntfy.sh/mytopic/subpath",
			want:  "https://ntfy.sh/mytopic",
		},
		{
			name:  "self-hosted ntfy with port",
			input: "http://localhost:8080/alerts",
			want:  "http://localhost:8080/alerts",
		},
		{
			name:  "url with trailing slash",
			input: "https://ntfy.sh/mytopic/",
			want:  "https://ntfy.sh/mytopic",
		},
		{
			name:  "url with query params",
			input: "https://ntfy.sh/mytopic?foo=bar",
			want:  "https://ntfy.sh/mytopic",
		},
		{
			name:    "empty url",
			input:   "",
			wantErr: true,
		},
		{
			name:    "missing scheme",
			input:   "ntfy.sh/mytopic",
			wantErr: true,
		},
		{
			name:    "missing topic",
			input:   "https://ntfy.sh",
			wantErr: true,
		},
		{
			name:    "missing topic with trailing slash",
			input:   "https://ntfy.sh/",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ntfyEndpoint(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("ntfyEndpoint() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ntfyEndpoint() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("ntfyEndpoint() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNtfyTagsForEvent(t *testing.T) {
	tests := []struct {
		event string
		want  string
	}{
		{"permission_required", "warning,lock"},
		{"input_required", "speech_balloon,question"},
		{"run_completed", "white_check_mark,done"},
		{"run_failed", "x,failure"},
		{"unknown_event", "bell"},
	}

	for _, tt := range tests {
		t.Run(tt.event, func(t *testing.T) {
			got := ntfyTagsForEvent(tt.event)
			if got != tt.want {
				t.Fatalf("ntfyTagsForEvent(%q) = %q, want %q", tt.event, got, tt.want)
			}
		})
	}
}
