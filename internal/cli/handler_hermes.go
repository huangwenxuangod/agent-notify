package cli

import (
	"context"
	"github.com/hellolib/agent-notify/internal/config"
	"github.com/hellolib/agent-notify/internal/hermeshooks"
	"github.com/spf13/cobra"
)

func newHandleHermesHookCmd(ctx context.Context, streams Streams) *cobra.Command {
	return &cobra.Command{Use: "handle-hermes-hook", Hidden: true, RunE: func(cmd *cobra.Command, args []string) error {
		cp, e := config.DefaultPath()
		if e != nil {
			return e
		}
		cfg, e := config.Load(cp)
		if e != nil {
			return e
		}
		sp, e := config.StatePath()
		if e != nil {
			return e
		}
		lp, e := config.LogPath()
		if e != nil {
			return e
		}
		return hermeshooks.Handle(ctx, cfg, sp, lp, cmd.InOrStdin())
	}}
}
