package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	mcppkg "github.com/pedronauck/agh/internal/mcp"
	"github.com/pedronauck/agh/internal/version"
	"github.com/spf13/cobra"
)

func newToolCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "tool",
		Short:  "Internal tool runtime commands",
		Hidden: true,
	}
	cmd.AddCommand(newToolMCPCommand(deps))
	return cmd
}

func newToolMCPCommand(deps commandDeps) *cobra.Command {
	var sessionID string
	var bindNonce string
	cmd := &cobra.Command{
		Use:    "mcp",
		Short:  "Serve session-bound AGH tools over MCP stdio",
		Hidden: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if strings.TrimSpace(sessionID) == "" {
				return mcppkg.ErrHostedSessionRequired
			}
			if strings.TrimSpace(bindNonce) == "" {
				return mcppkg.ErrHostedNonceRequired
			}
			runtime, err := loadRuntimeContext(deps)
			if err != nil {
				return err
			}
			if err := deps.ensureHome(runtime.HomePaths); err != nil {
				return fmt.Errorf("cli: ensure AGH home: %w", err)
			}
			client, err := deps.newClient(runtime.Config.Daemon.Socket)
			if err != nil {
				return err
			}
			hostedClient, ok := client.(mcppkg.HostedProxyClient)
			if !ok {
				return errors.New("cli: daemon client does not support hosted MCP")
			}
			return mcppkg.RunHostedProxy(cmd.Context(), hostedClient, mcppkg.HostedProxyOptions{
				SessionID: sessionID,
				Nonce:     bindNonce,
				Stdin:     os.Stdin,
				Stdout:    os.Stdout,
				Stderr:    os.Stderr,
				Version:   version.Current().Version,
			})
		},
	}
	cmd.Flags().StringVar(&sessionID, "session", "", "Session id to bind")
	cmd.Flags().StringVar(&bindNonce, "bind-nonce", "", "Hosted MCP bind nonce")
	return cmd
}
