package autostart

import "runtime"

type Status struct {
	Supported bool   `json:"supported"`
	Enabled   bool   `json:"enabled"`
	Platform  string `json:"platform"`
	Path      string `json:"path,omitempty"`
}
type Manager interface {
	Status() (Status, error)
	Enable() error
	Disable() error
}

func New(binary string) Manager {
	switch runtime.GOOS {
	case "darwin":
		return newDarwin(binary)
	case "windows":
		return newWindows(binary)
	default:
		return unsupported{platform: runtime.GOOS}
	}
}

type unsupported struct{ platform string }

func (u unsupported) Status() (Status, error) { return Status{Platform: u.platform}, nil }
func (u unsupported) Enable() error           { return errUnsupported }
func (u unsupported) Disable() error          { return errUnsupported }
