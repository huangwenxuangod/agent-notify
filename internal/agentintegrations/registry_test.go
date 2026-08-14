package agentintegrations

import (
	"slices"
	"testing"

	"github.com/hellolib/agent-notify/internal/config"
)

func TestAllDescriptorsCoverEveryConfiguredAgent(t *testing.T) {
	cfg := config.Default()
	got := All()

	ids := make([]string, 0, len(got))
	seen := make(map[string]bool, len(got))
	for _, descriptor := range got {
		if descriptor.ID == "" || descriptor.Name == "" || descriptor.Integration == nil {
			t.Fatalf("invalid descriptor: %#v", descriptor)
		}
		if seen[descriptor.ID] {
			t.Fatalf("duplicate descriptor ID: %s", descriptor.ID)
		}
		seen[descriptor.ID] = true
		ids = append(ids, descriptor.ID)

		if descriptor.Target(&cfg) == nil {
			t.Fatalf("%s has no agent config accessor", descriptor.ID)
		}
		if descriptor.Notify(&cfg) == nil {
			t.Fatalf("%s has no notify config accessor", descriptor.ID)
		}
	}

	want := []string{"claude_code", "codex", "zcode", "grok", "droid", "opencode", "workbuddy", "hermes", "openclaw"}
	if !slices.Equal(ids, want) {
		t.Fatalf("descriptor IDs = %v, want %v", ids, want)
	}
}
