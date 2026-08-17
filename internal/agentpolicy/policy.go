package agentpolicy

// Policy describes a primary Hook source and optional fallback sources for
// terminal lifecycle events. It has no integration-package dependency so the
// dispatch path can use it without importing agent-specific Hook packages.
type Policy struct {
	Events   []string
	Primary  string
	Fallback []string
}

func (p Policy) Applies(event, origin string) bool {
	if event == "" || origin == "" || p.Primary == "" {
		return false
	}
	for _, supported := range p.Events {
		if supported != event {
			continue
		}
		if origin == p.Primary {
			return true
		}
		for _, fallback := range p.Fallback {
			if fallback == origin {
				return true
			}
		}
	}
	return false
}

// For returns the source policy for agents with a native Hook and a monitor
// fallback. Agents without two terminal sources intentionally return empty.
func For(agent string) Policy {
	switch agent {
	case "codex", "workbuddy", "pi":
		return Policy{Events: []string{"run_completed", "run_failed"}, Primary: "native_hook", Fallback: []string{"desktop_monitor"}}
	default:
		return Policy{}
	}
}
