//go:build darwin

package autostart

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

type darwin struct{ binary, path string }

func newWindows(string) Manager { return unsupported{platform: "windows"} }

func newDarwin(binary string) Manager {
	h, _ := os.UserHomeDir()
	return &darwin{binary: binary, path: filepath.Join(h, "Library", "LaunchAgents", "com.agentnotify.tray.plist")}
}
func (d *darwin) Status() (Status, error) {
	_, err := os.Stat(d.path)
	return Status{Supported: true, Enabled: err == nil, Platform: "darwin", Path: d.path}, nil
}
func (d *darwin) Enable() error {
	if err := os.MkdirAll(filepath.Dir(d.path), 0700); err != nil {
		return err
	}
	xml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd"><plist version="1.0"><dict><key>Label</key><string>com.agentnotify.tray</string><key>ProgramArguments</key><array><string>%s</string><string>tray</string></array><key>RunAtLoad</key><true/></dict></plist>`, d.binary)
	if err := os.WriteFile(d.path, []byte(xml), 0600); err != nil {
		return err
	}
	return exec.Command("launchctl", "bootstrap", "gui/"+strconv.Itoa(os.Getuid()), d.path).Run()
}
func (d *darwin) Disable() error {
	_ = exec.Command("launchctl", "bootout", "gui/"+strconv.Itoa(os.Getuid()), d.path).Run()
	if err := os.Remove(d.path); os.IsNotExist(err) {
		return nil
	} else {
		return err
	}
}
