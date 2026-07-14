package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

const (
	networkInvitationScopeWorkspace = "workspace"
	networkInvitationScopeTask      = "task"
)

func registerNetworkPublicSurfaceCommands(cmd *cobra.Command, deps commandDeps, workspaceRef *string) {
	cmd.AddCommand(newNetworkCoordinationCommand(deps, workspaceRef))
	cmd.AddCommand(newNetworkInvitationCommand(deps, workspaceRef))
	cmd.AddCommand(newNetworkUsageCommand(deps, workspaceRef))
}

func newNetworkCoordinationCommand(deps commandDeps, workspaceRef *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "coordination",
		Short: "Inspect or change workspace network coordination",
	}
	cmd.AddCommand(newNetworkCoordinationStatusCommand(deps, workspaceRef))
	cmd.AddCommand(newNetworkCoordinationEnableCommand(deps, workspaceRef))
	cmd.AddCommand(newNetworkCoordinationDisableCommand(deps, workspaceRef))
	return cmd
}

func newNetworkCoordinationStatusCommand(deps commandDeps, workspaceRef *string) *cobra.Command {
	var taskID string
	cmd := &cobra.Command{
		Use:   configStatusKey,
		Short: "Show workspace network coordination and invitation state",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			workspaceID, err := resolveNetworkWorkspaceRef(cmd, deps, client, workspaceRef)
			if err != nil {
				return err
			}
			payload, err := client.GetNetworkCoordination(cmd.Context(), workspaceID, taskID)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, coordinationOutputBundle(payload))
		},
	}
	cmd.Flags().StringVar(&taskID, "task-id", "", "Optional task scope for invitation state")
	return cmd
}

func newNetworkCoordinationEnableCommand(deps commandDeps, workspaceRef *string) *cobra.Command {
	return newNetworkCoordinationToggleCommand(
		deps,
		workspaceRef,
		true,
		"enable",
		"Enable workspace coordination conversations",
	)
}

func newNetworkCoordinationDisableCommand(deps commandDeps, workspaceRef *string) *cobra.Command {
	return newNetworkCoordinationToggleCommand(
		deps,
		workspaceRef,
		false,
		"disable",
		"Disable workspace coordination conversations",
	)
}

func newNetworkCoordinationToggleCommand(
	deps commandDeps,
	workspaceRef *string,
	enabled bool,
	use string,
	short string,
) *cobra.Command {
	var taskID string
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			workspaceID, err := resolveNetworkWorkspaceRef(cmd, deps, client, workspaceRef)
			if err != nil {
				return err
			}
			payload, err := client.PutNetworkCoordination(
				cmd.Context(),
				workspaceID,
				PutNetworkCoordinationRequest{Enabled: enabled},
				taskID,
			)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, coordinationOutputBundle(payload))
		},
	}
	cmd.Flags().StringVar(&taskID, "task-id", "", "Optional task scope for invitation state")
	return cmd
}

func newNetworkInvitationCommand(deps commandDeps, workspaceRef *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "invitation",
		Short: "Dismiss or reset the coordination invitation",
	}
	cmd.AddCommand(newNetworkInvitationDismissCommand(deps, workspaceRef))
	cmd.AddCommand(newNetworkInvitationResetCommand(deps, workspaceRef))
	return cmd
}

func newNetworkInvitationDismissCommand(deps commandDeps, workspaceRef *string) *cobra.Command {
	return newNetworkInvitationMutationCommand(
		deps,
		workspaceRef,
		true,
		"dismiss",
		"Dismiss the coordination invitation for a scope",
	)
}

func newNetworkInvitationResetCommand(deps commandDeps, workspaceRef *string) *cobra.Command {
	return newNetworkInvitationMutationCommand(
		deps,
		workspaceRef,
		false,
		"reset",
		"Reset the coordination invitation for a scope",
	)
}

func newNetworkInvitationMutationCommand(
	deps commandDeps,
	workspaceRef *string,
	dismissed bool,
	use string,
	short string,
) *cobra.Command {
	var scope string
	var taskID string
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			workspaceID, err := resolveNetworkWorkspaceRef(cmd, deps, client, workspaceRef)
			if err != nil {
				return err
			}
			normalizedScope := strings.TrimSpace(strings.ToLower(scope))
			if normalizedScope == "" {
				normalizedScope = networkInvitationScopeWorkspace
			}
			if normalizedScope != networkInvitationScopeWorkspace &&
				normalizedScope != networkInvitationScopeTask {
				return fmt.Errorf("cli: --scope must be workspace or task")
			}
			payload, err := client.PutNetworkCoordinationInvitation(
				cmd.Context(),
				workspaceID,
				PutNetworkCoordinationInvitationRequest{
					Scope:     normalizedScope,
					TaskID:    strings.TrimSpace(taskID),
					Dismissed: dismissed,
				},
			)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, coordinationOutputBundle(payload))
		},
	}
	cmd.Flags().StringVar(
		&scope,
		"scope",
		networkInvitationScopeWorkspace,
		"Invitation scope: workspace or task",
	)
	cmd.Flags().StringVar(&taskID, "task-id", "", "Task id when --scope=task")
	return cmd
}
