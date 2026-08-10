//go:build !darwin && !windows

package autostart

func newDarwin(string) Manager  { return unsupported{platform: "darwin"} }
func newWindows(string) Manager { return unsupported{platform: "windows"} }
