package notify

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	ntfyHTTPTimeout      = 10 * time.Second
	maxNtfyErrorBodySize = 512
)

// NtfySender sends notifications through a ntfy.sh (or self-hosted ntfy) topic.
type NtfySender struct {
	topicURL    string
	accessToken string
	httpClient  *http.Client
}

// NewNtfySender creates a NtfySender with the provided ntfy topic URL.
// The URL should be in the format https://ntfy.sh/<topic> or a self-hosted ntfy server URL.
func NewNtfySender(topicURL string) *NtfySender {
	return NewNtfySenderWithAccessToken(topicURL, "")
}

// NewNtfySenderWithAccessToken creates a ntfy sender with optional bearer auth.
func NewNtfySenderWithAccessToken(topicURL, accessToken string) *NtfySender {
	return &NtfySender{
		topicURL:    strings.TrimSpace(topicURL),
		accessToken: strings.TrimSpace(accessToken),
		httpClient:  &http.Client{Timeout: ntfyHTTPTimeout},
	}
}

func (s *NtfySender) Name() string { return "ntfy" }

func (s *NtfySender) Send(ctx context.Context, msg Message) error {
	endpoint, err := ntfyEndpoint(s.topicURL)
	if err != nil {
		return err
	}

	title, body := ntfyPayloadText(msg)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("ntfy: create request: %w", err)
	}
	req.Header.Set("Title", title)
	req.Header.Set("Tags", ntfyTagsForEvent(msg.Event))
	if s.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.accessToken)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ntfy: send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxNtfyErrorBodySize))
		return fmt.Errorf("ntfy: unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func ntfyPayloadText(msg Message) (string, string) {
	emoji := eventEmoji(msg.Event)
	title := fmt.Sprintf("%s %s", emoji, msg.Title)

	var body strings.Builder
	body.WriteString(msg.Body)
	if msg.Workspace != "" && msg.Agent != "codex" {
		body.WriteString(fmt.Sprintf("\n\nWorkspace: %s", msg.Workspace))
	}
	return title, body.String()
}

func ntfyTagsForEvent(event string) string {
	switch event {
	case "permission_required":
		return "warning,lock"
	case "input_required":
		return "speech_balloon,question"
	case "run_completed":
		return "white_check_mark,done"
	case "run_failed":
		return "x,failure"
	default:
		return "bell"
	}
}

func ntfyEndpoint(rawURL string) (string, error) {
	if rawURL == "" {
		return "", fmt.Errorf("ntfy: topic_url is empty")
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("ntfy: parse topic_url: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("ntfy: topic_url must include scheme and host")
	}

	// Ensure there's a topic (path segment)
	topic := firstPathSegment(u.Path)
	if topic == "" {
		return "", fmt.Errorf("ntfy: topic_url missing topic name")
	}

	// Normalize to /<topic>
	u.Path = "/" + topic
	u.RawPath = ""
	u.RawQuery = ""
	return u.String(), nil
}
