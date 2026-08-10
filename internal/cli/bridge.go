package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/hellolib/agent-notify/internal/bridge"
	"github.com/hellolib/agent-notify/internal/common"
	"github.com/hellolib/agent-notify/internal/config"
	"github.com/spf13/cobra"
)

const defaultBridgePort = 45174

func newBridgeCmd(streams Streams) *cobra.Command {
	var port int
	cmd := &cobra.Command{
		Use:   "bridge",
		Short: "Run the local Agent Notify control-plane API",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBridge(cmd.Context(), streams, port)
		},
	}
	cmd.Flags().IntVar(&port, "port", defaultBridgePort, "loopback TCP port")
	return cmd
}

func newTrayCmd(streams Streams) *cobra.Command {
	cmd := newBridgeCmd(streams)
	cmd.Use = "tray"
	cmd.Short = "Run the background Agent Notify host process"
	return cmd
}

func runBridge(ctx context.Context, streams Streams, port int) error {
	if port < 0 || port > 65535 {
		return fmt.Errorf("invalid bridge port %d", port)
	}
	tokenPath, err := config.BridgeTokenPath()
	if err != nil {
		return err
	}
	token, err := bridge.EnsureToken(tokenPath)
	if err != nil {
		return err
	}
	cfgPath, err := config.DefaultPath()
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
	svc, err := bridge.NewService(bridge.Options{
		ConfigPath: cfgPath,
		StatePath:  statePath,
		LogPath:    logPath,
		BinaryPath: common.ResolveBinaryPath(""),
	})
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return err
	}
	defer listener.Close()

	server := &http.Server{Handler: bridge.NewHTTPHandler(svc, token), ReadHeaderTimeout: 5 * time.Second}
	errCh := make(chan error, 1)
	go func() { errCh <- server.Serve(listener) }()
	fmt.Fprintf(streams.Stdout, "Bridge listening on http://%s\n", listener.Addr())

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return err
		}
		if err := <-errCh; err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func newSetupCmd(streams Streams) *cobra.Command {
	var jsonOutput bool
	var agents []string
	var scope, binaryPath string
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Install Agent Notify hooks without prompts",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !jsonOutput {
				return fmt.Errorf("setup requires --json; use agent-notify init for interactive setup")
			}
			cfgPath, err := config.DefaultPath()
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
			svc, err := bridge.NewService(bridge.Options{ConfigPath: cfgPath, StatePath: statePath, LogPath: logPath, BinaryPath: common.ResolveBinaryPath(binaryPath)})
			if err != nil {
				return err
			}
			result, err := svc.InstallAgents(bridge.SetupRequest{Agents: agents, Scope: scope, BinaryPath: common.ResolveBinaryPath(binaryPath)})
			if err != nil {
				return err
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "write machine-readable setup results")
	cmd.Flags().StringSliceVar(&agents, "agents", nil, "agent IDs to install (defaults to all supported agents)")
	cmd.Flags().StringVar(&scope, "scope", "user", "install scope: user or project")
	cmd.Flags().StringVar(&binaryPath, "binary", "", "agent-notify binary path")
	return cmd
}
