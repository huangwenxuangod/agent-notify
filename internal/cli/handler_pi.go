package cli

import (
	"context"

	"github.com/hellolib/agent-notify/internal/config"
	"github.com/hellolib/agent-notify/internal/pihooks"
	"github.com/spf13/cobra"
)

func newHandlePiHookCmd(ctx context.Context, streams Streams) *cobra.Command {
	return &cobra.Command{Use: "handle-pi-hook", Hidden: true, RunE: func(cmd *cobra.Command, args []string) error {
		configPath, err := config.DefaultPath()
		if err != nil {
			return err
		}
		cfg, err := config.Load(configPath)
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
		return pihooks.Handle(ctx, cfg, statePath, logPath, streams.Stdin)
	}}
}
