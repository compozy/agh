package cli

import (
	"github.com/spf13/cobra"
)

func newNetworkUsageCommand(deps commandDeps, workspaceRef *string) *cobra.Command {
	return &cobra.Command{
		Use:   "usage",
		Short: "Show workspace-scoped network wake usage from the ledger",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			workspaceID, err := resolveNetworkWorkspaceRef(cmd, deps, client, workspaceRef)
			if err != nil {
				return err
			}
			payload, err := client.GetNetworkUsage(cmd.Context(), workspaceID)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, networkUsageOutputBundle(payload))
		},
	}
}
