package cli

import (
	"context"
	"github.com/hellolib/agent-notify/internal/config"
	"github.com/hellolib/agent-notify/internal/workbuddyhooks"
	"github.com/spf13/cobra"
)

func newHandleWorkBuddyHookCmd(ctx context.Context, streams Streams) *cobra.Command {
	return &cobra.Command{Use: "handle-workbuddy-hook", Hidden: true, RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, err := config.DefaultPath()
		if err != nil {
			return err
		}
		cfg, err := config.Load(cfgPath)
		if err != nil {
			return err
		}
		statePath, err := config.StatePath()
		if err != nil {
			return err
		}
		logPath, err := config.LogPath()
		if err != nil {
			return err
		}
		return workbuddyhooks.Handle(ctx, cfg, statePath, logPath, cmd.InOrStdin())
	}}
}
