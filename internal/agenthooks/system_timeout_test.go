package agenthooks

import (
	"testing"
	"time"
)

func TestSystemSendTimeoutUsesNativeHelperMinimum(t *testing.T) {
	if got := systemSendTimeout(5 * time.Second); got != 15*time.Second {
		t.Fatalf("systemSendTimeout(5s) = %s, want 15s", got)
	}
	if got := systemSendTimeout(20 * time.Second); got != 20*time.Second {
		t.Fatalf("systemSendTimeout(20s) = %s, want 20s", got)
	}
}
