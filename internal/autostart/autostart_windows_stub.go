//go:build windows

package autostart

func newDarwin(string) Manager { return unsupported{platform: "darwin"} }
