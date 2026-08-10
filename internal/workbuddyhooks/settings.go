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
