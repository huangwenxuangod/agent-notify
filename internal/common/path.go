package common

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const DefaultBinaryPath = "agent-notify"

func hookBinaryName() string {
	if runtime.GOOS == "windows" {
		return DefaultBinaryPath + ".exe"
	}
	return DefaultBinaryPath
}

// HookBinaryPath returns the managed host binary used by agent hooks. Desktop
// development builds live in temporary directories, so they must not be
// written into persistent agent settings.
func HookBinaryPath() string {
	if home, err := os.UserHomeDir(); err == nil {
		path := filepath.Join(home, ".agent-notify", hookBinaryName())
		if info, statErr := os.Stat(path); statErr == nil && !info.IsDir() {
			return toUnixStylePath(path)
		}
	}
	return ResolveBinaryPath("")
}

func ResolveBinaryPath(input string) string {
	input = strings.TrimSpace(input)
	if input != "" {
		return toUnixStylePath(input)
	}

	executablePath, err := os.Executable()
	if err == nil {
		if resolved, resolveErr := filepath.EvalSymlinks(executablePath); resolveErr == nil {
			return toUnixStylePath(resolved)
		}
		return toUnixStylePath(executablePath)
	}

	return DefaultBinaryPath
}

// toUnixStylePath converts backslashes to forward slashes (e.g., C:\Users\... -> C:/Users/...)
// This format works in cmd.exe, PowerShell, and Git Bash on Windows.
func toUnixStylePath(path string) string {
	return strings.ReplaceAll(path, "\\", "/")
}
