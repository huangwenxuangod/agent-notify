package workbuddyhooks

import (
	"errors"
	"os"

	"github.com/hellolib/agent-notify/internal/common"
)

const hookCommandMarker = "handle-workbuddy-hook"

var managedEvents = []string{
	"SessionStart",
	"PermissionRequest",
	"Notification",
	"Stop",
	"StopFailure",
	"PostToolUseFailure",
}

func BuildHookSettings(binaryPath string) map[string]any {
	command := common.QuotePathForShell(common.ResolveBinaryPath(binaryPath)) + " " + hookCommandMarker
	hooks := map[string]any{}
	for _, event := range managedEvents {
		hooks[event] = []map[string]any{{
			"hooks": []map[string]any{{"type": "command", "command": command}},
		}}
	}
	return map[string]any{"hooks": hooks}
}

func Install(path, binaryPath string) error {
	settings, err := common.ReadOrderedSettings(path)
	if err != nil {
		return err
	}
	hooks, err := common.ChildObject(settings, "hooks")
	if err != nil {
		return err
	}
	command := common.QuotePathForShell(common.ResolveBinaryPath(binaryPath)) + " " + hookCommandMarker
	if err := common.InstallManagedHooks(&hooks, managedEvents, hookCommandMarker, command, common.RefuseNonArrayEvent("hooks")); err != nil {
		return err
	}
	if err := common.SetChildObject(&settings, "hooks", hooks); err != nil {
		return err
	}
	return common.WriteOrderedSettings(path, settings)
}

func IsInstalled(path string) (bool, error) {
	settings, err := common.ReadOrderedSettings(path)
	if err != nil {
		return false, err
	}
	hooks, err := common.ChildObject(settings, "hooks")
	if err != nil {
		return false, nil
	}
	return common.HasManagedHook(hooks, managedEvents, hookCommandMarker), nil
}

// IsInstalledWithBinary verifies every managed event uses the current hook
// command. A marker-only check would treat a stale development binary as a
// valid installation and prevent the desktop app from repairing it.
func IsInstalledWithBinary(path, binaryPath string) (bool, error) {
	settings, err := common.ReadOrderedSettings(path)
	if err != nil {
		return false, err
	}
	hooks, err := common.ChildObject(settings, "hooks")
	if err != nil {
		return false, nil
	}
	want := common.QuotePathForShell(common.ResolveBinaryPath(binaryPath)) + " " + hookCommandMarker
	for _, event := range managedEvents {
		raw, ok := hooks.Get(event)
		if !ok {
			return false, nil
		}
		entries, err := common.HookEntries(raw)
		if err != nil {
			return false, nil
		}
		found := false
		for _, entry := range entries {
			updated, hit, _ := common.SyncEntryCommand(entry, hookCommandMarker, want)
			if hit && string(updated) == string(entry) {
				found = true
				break
			}
		}
		if !found {
			return false, nil
		}
	}
	return true, nil
}

func Uninstall(path string) error {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return nil
	}
	settings, err := common.ReadOrderedSettings(path)
	if err != nil {
		return err
	}
	hooks, err := common.ChildObject(settings, "hooks")
	if err != nil {
		return nil
	}
	if err := common.UninstallManagedHooks(&hooks, hookCommandMarker); err != nil {
		return err
	}
	if hooks.Len() == 0 {
		settings.Delete("hooks")
	} else if err := common.SetChildObject(&settings, "hooks", hooks); err != nil {
		return err
	}
	return common.WriteOrderedSettings(path, settings)
}
