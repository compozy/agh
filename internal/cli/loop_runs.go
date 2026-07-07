package cli

import (
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/api/contract"
	"github.com/spf13/cobra"
)

func newLoopStatusCommand(deps commandDeps) *cobra.Command {
	var workspaceRef, runID string
	cmd := &cobra.Command{
		Use:   loopStatusKey,
		Short: "Inspect one Loop run",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, workspaceID, err := loopClientAndWorkspace(cmd, deps, workspaceRef)
			if err != nil {
				return err
			}
			id, err := requiredLoopFlag(loopRunIDKey, runID)
			if err != nil {
				return err
			}
			response, err := client.GetLoopRun(cmd.Context(), workspaceID, id)
			if err != nil {
				return err
			}
			message := fmt.Sprintf("Loop run %s is %s", id, response.Run.Status)
			return writeCommandOutput(cmd, loopOutputBundle(response, message))
		},
	}
	addLoopRunIDFlags(cmd, &workspaceRef, &runID)
	return cmd
}

func newLoopRunsCommand(deps commandDeps) *cobra.Command {
	var workspaceRef, loopName, status string
	var limit int
	cmd := &cobra.Command{
		Use:   loopRunsKey,
		Short: "List workspace Loop runs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, workspaceID, err := loopClientAndWorkspace(cmd, deps, workspaceRef)
			if err != nil {
				return err
			}
			response, err := client.ListLoopRuns(cmd.Context(), workspaceID, LoopRunListQuery{
				LoopName: strings.TrimSpace(loopName),
				Status:   strings.TrimSpace(status),
				Limit:    limit,
			})
			if err != nil {
				return err
			}
			message := fmt.Sprintf("%d loop runs", len(response.Runs))
			return writeCommandOutput(cmd, loopOutputBundle(response, message))
		},
	}
	cmd.Flags().StringVar(&workspaceRef, loopWorkspaceKey, "", "Workspace path, name, or ID")
	cmd.Flags().StringVar(&loopName, loopLoopKey, "", "Filter by Loop name")
	cmd.Flags().StringVar(&status, loopStatusKey, "", "Filter by Loop run status")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum runs to return")
	mustMarkFlagRequired(cmd, loopWorkspaceKey)
	return cmd
}

func newLoopRunActionCommand(deps commandDeps, verb string, short string) *cobra.Command {
	var workspaceRef, runID string
	cmd := &cobra.Command{
		Use:   verb,
		Short: short,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, workspaceID, err := loopClientAndWorkspace(cmd, deps, workspaceRef)
			if err != nil {
				return err
			}
			id, err := requiredLoopFlag(loopRunIDKey, runID)
			if err != nil {
				return err
			}
			if err := executeLoopRunAction(
				cmd.Context(),
				client,
				verb,
				workspaceID,
				id,
				agentCredentialsFromEnv(deps),
			); err != nil {
				return err
			}
			return writeLoopMutationOK(cmd, verb, id)
		},
	}
	addLoopRunIDFlags(cmd, &workspaceRef, &runID)
	return cmd
}

func newLoopApproveCommand(deps commandDeps) *cobra.Command {
	var workspaceRef, runID, gateID, decision string
	cmd := &cobra.Command{
		Use:   loopApproveKey,
		Short: "Apply one Loop human-gate decision",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, workspaceID, err := loopClientAndWorkspace(cmd, deps, workspaceRef)
			if err != nil {
				return err
			}
			id, err := requiredLoopFlag(loopRunIDKey, runID)
			if err != nil {
				return err
			}
			gate, err := requiredLoopFlag(loopGateIDKey, gateID)
			if err != nil {
				return err
			}
			normalizedDecision, err := parseLoopGateDecision(decision)
			if err != nil {
				return err
			}
			if err := client.ApproveLoopRun(cmd.Context(), workspaceID, id, contract.ApproveLoopRunRequest{
				GateID:   gate,
				Decision: contract.LoopGateDecision(normalizedDecision),
			}, agentCredentialsFromEnv(deps)); err != nil {
				return err
			}
			return writeLoopMutationOK(cmd, "approved", id)
		},
	}
	addLoopRunIDFlags(cmd, &workspaceRef, &runID)
	cmd.Flags().StringVar(&gateID, loopGateIDKey, "", "Gate node ID")
	cmd.Flags().StringVar(&decision, loopDecisionKey, "", "Decision: approve, request_changes, or reject")
	mustMarkFlagRequired(cmd, loopGateIDKey)
	mustMarkFlagRequired(cmd, loopDecisionKey)
	return cmd
}

func addLoopRunIDFlags(cmd *cobra.Command, workspaceRef *string, runID *string) {
	cmd.Flags().StringVar(workspaceRef, loopWorkspaceKey, "", "Workspace path, name, or ID")
	cmd.Flags().StringVar(runID, loopRunIDKey, "", "Loop run ID")
	mustMarkFlagRequired(cmd, loopWorkspaceKey)
	mustMarkFlagRequired(cmd, loopRunIDKey)
}
