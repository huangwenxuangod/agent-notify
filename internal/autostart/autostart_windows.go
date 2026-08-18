//go:build windows

package autostart

import (
	"golang.org/x/sys/windows/registry"
	"os"
	"strings"
)

type windowsManager struct{ binary string }

func newWindows(binary string) Manager { return &windowsManager{binary: binary} }
func (w *windowsManager) Status() (Status, error) {
	k, e := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.QUERY_VALUE)
	if e != nil {
		return Status{Supported: true, Platform: "windows"}, nil
	}
	defer k.Close()
	_, _, e = k.GetStringValue("AgentNotifyTray")
	return Status{Supported: true, Enabled: e == nil, Platform: "windows", Path: w.binary}, nil
}
func (w *windowsManager) Enable() error {
	k, e := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if e != nil {
		return e
	}
	defer k.Close()
	return k.SetStringValue("AgentNotifyTray", startupCommand(w.binary))
}
func (w *windowsManager) Disable() error {
	k, e := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if e != nil {
		return e
	}
	defer k.Close()
	e = k.DeleteValue("AgentNotifyTray")
	if os.IsNotExist(e) {
		return nil
	}
	return e
}

func startupCommand(binary string) string {
	binary = strings.TrimSpace(binary)
	if strings.HasPrefix(binary, `"`) && strings.HasSuffix(binary, `"`) {
		return binary + " tray"
	}
	return `"` + strings.ReplaceAll(binary, `"`, `\"`) + `" tray`
}
