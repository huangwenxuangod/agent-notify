package cli

import (
	"fmt"

	"github.com/hellolib/agent-notify/internal/agentintegrations"
	"github.com/hellolib/agent-notify/internal/common"
	"github.com/hellolib/agent-notify/internal/pihooks"
	"github.com/spf13/cobra"
)

func newPiCmd(streams Streams) *cobra.Command {
	cmd := &cobra.Command{Use: "pi", Short: "Manage Pi extension integration"}
	cmd.AddCommand(newPiPrintExtensionCmd(streams), newPiInstallExtensionCmd())
	return cmd
}

func newPiPrintExtensionCmd(streams Streams) *cobra.Command {
	var binaryPath string
	cmd := &cobra.Command{Use: "print-extension", Short: "Print Pi extension source", RunE: func(cmd *cobra.Command, args []string) error {
		_, err := fmt.Fprint(streams.Stdout, pihooks.BuildExtension(common.ResolveBinaryPath(binaryPath)))
		return err
	}}
	cmd.Flags().StringVar(&binaryPath, "binary", common.ResolveBinaryPath(""), "agent-notify binary path")
	return cmd
}

func newPiInstallExtensionCmd() *cobra.Command {
	var binaryPath, scope string
	cmd := &cobra.Command{Use: "install-extension", Short: "Install Pi extension", RunE: func(cmd *cobra.Command, args []string) error {
		integration := agentintegrations.NewPiIntegration()
		path, err := integration.SettingsPath(scope)
		if err != nil {
			return err
		}
		return integration.Install(path, common.ResolveBinaryPath(binaryPath))
	}}
	cmd.Flags().StringVar(&binaryPath, "binary", common.ResolveBinaryPath(""), "agent-notify binary path")
	cmd.Flags().StringVar(&scope, "scope", "user", "install scope")
	return cmd
}
