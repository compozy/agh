package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/spf13/cobra"
)

const (
	configValidKey = "valid"
)

const (
	providerAuthStatePresent = "present"
)

const (
	automationEnabledKey             = "enabled"
	authoredContextPreviousDigestKey = "previous_digest"
	workspaceSkillSource             = "workspace"
)

const (
	authoredContextDigestKey    = "digest"
	authoredContextNewDigestKey = "new_digest"
	networkStateKey             = "state"
)

const (
	automationCreatedAtKey = "created_at"
	memoryReasonKey        = "reason"
	sessionSessionKey      = "session"
)

const (
	authoredContextActionValue         = "Action"
	authoredContextActiveValue         = "Active"
	authoredContextAgentValue          = "Agent"
	authoredContextConfigDigestValue   = "Config Digest"
	authoredContextCreatedValue        = "Created"
	authoredContextDigestValue         = "Digest"
	authoredContextEnabledValue        = "Enabled"
	authoredContextEventValue          = "Event"
	authoredContextHealthValue         = "Health"
	authoredContextLastActivityValue   = "Last Activity"
	authoredContextMessageValue        = "Message"
	authoredContextNewDigestValue      = "New Digest"
	authoredContextOperationValue      = "Operation"
	authoredContextPresentValue        = "Present"
	authoredContextPreviousDigestValue = "Previous Digest"
	authoredContextReasonValue         = "Reason"
	authoredContextResultValue         = "Result"
	authoredContextSessionValue        = "Session"
	authoredContextSnapshotValue       = "Snapshot"
	authoredContextSourceValue         = "Source"
	authoredContextStateValue          = "State"
	authoredContextSummaryValue        = "Summary"
	authoredContextUpdatedValue        = "Updated"
	authoredContextValidValue          = "Valid"
	authoredContextWorkspaceValue      = "Workspace"
	authoredContextActionKey           = "action"
	authoredContextActiveKey           = "active"
	authoredContextConfigDigestKey     = "config_digest"
	authoredContextEventKey            = "event"
	authoredContextHealthKey           = "health"
	authoredContextHeartbeatKey        = "heartbeat"
	authoredContextOperationKey        = "operation"
	authoredContextSoulKey             = "soul"
)

type authoredBodyInput struct {
	file  string
	stdin bool
}

func newAgentSoulCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   authoredContextSoulKey,
		Short: "Inspect and manage agent SOUL.md files",
	}
	cmd.AddCommand(newAgentSoulInspectCommand(deps))
	cmd.AddCommand(newAgentSoulValidateCommand(deps))
	cmd.AddCommand(newAgentSoulWriteCommand(deps))
	cmd.AddCommand(newAgentSoulDeleteCommand(deps))
	cmd.AddCommand(newAgentSoulHistoryCommand(deps))
	cmd.AddCommand(newAgentSoulRollbackCommand(deps))
	return cmd
}

func newAgentSoulInspectCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "inspect <agent>",
		Short:   "Inspect one agent's resolved Soul",
		Example: "  agh agent soul inspect coder --workspace checkout-api --json",
		Args:    exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			query, err := agentQueryFromCommand(cmd)
			if err != nil {
				return err
			}
			record, err := client.GetAgentSoul(cmd.Context(), args[0], query)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, agentSoulBundle(record))
		},
	}
	addWorkspaceFlag(cmd)
	return cmd
}

func newAgentSoulValidateCommand(deps commandDeps) *cobra.Command {
	var input authoredBodyInput
	cmd := &cobra.Command{
		Use:     "validate <agent>",
		Short:   "Validate a proposed Soul body or the current SOUL.md",
		Example: "  agh agent soul validate coder --file SOUL.md --workspace checkout-api --json",
		Args:    exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			workspace, err := commandWorkspaceFlag(cmd)
			if err != nil {
				return err
			}
			body, err := readAuthoredBody(cmd, input, false)
			if err != nil {
				return err
			}
			record, err := client.ValidateAgentSoul(cmd.Context(), args[0], AgentSoulValidateRequest{
				WorkspaceID: workspace,
				AgentName:   args[0],
				Body:        body,
			})
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, agentSoulBundle(record))
		},
	}
	addWorkspaceFlag(cmd)
	addAuthoredBodyFlags(cmd, &input)
	return cmd
}

func newAgentSoulWriteCommand(deps commandDeps) *cobra.Command {
	var (
		input          authoredBodyInput
		expectedDigest string
		idempotencyKey string
	)
	cmd := &cobra.Command{
		Use:     "write <agent>",
		Short:   "Create or replace SOUL.md through managed authoring",
		Example: "  agh agent soul write coder --file SOUL.md --expected-digest sha256:old --workspace checkout-api --json",
		Args:    exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			workspace, err := commandWorkspaceFlag(cmd)
			if err != nil {
				return err
			}
			body, err := readAuthoredBody(cmd, input, true)
			if err != nil {
				return err
			}
			digest := optionalStringFlag(cmd, "expected-digest", expectedDigest)
			record, err := client.PutAgentSoul(cmd.Context(), args[0], AgentSoulPutRequest{
				WorkspaceID:    workspace,
				AgentName:      args[0],
				Body:           body,
				ExpectedDigest: digest,
				IdempotencyKey: strings.TrimSpace(idempotencyKey),
			})
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, agentSoulMutationBundle(&record))
		},
	}
	addWorkspaceFlag(cmd)
	addAuthoredBodyFlags(cmd, &input)
	cmd.Flags().StringVar(&expectedDigest, "expected-digest", "", "Expected current Soul digest for CAS")
	cmd.Flags().StringVar(&idempotencyKey, "idempotency-key", "", "Optional idempotency key")
	return cmd
}

func newAgentSoulDeleteCommand(deps commandDeps) *cobra.Command {
	var expectedDigest string
	cmd := &cobra.Command{
		Use:     "delete <agent>",
		Short:   "Delete SOUL.md through managed authoring",
		Example: "  agh agent soul delete coder --expected-digest sha256:old --workspace checkout-api --json",
		Args:    exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			workspace, err := commandWorkspaceFlag(cmd)
			if err != nil {
				return err
			}
			digest, err := changedStringFlag(cmd, "expected-digest", expectedDigest)
			if err != nil {
				return err
			}
			record, err := client.DeleteAgentSoul(cmd.Context(), args[0], AgentSoulDeleteRequest{
				WorkspaceID:    workspace,
				AgentName:      args[0],
				ExpectedDigest: digest,
			})
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, agentSoulMutationBundle(&record))
		},
	}
	addWorkspaceFlag(cmd)
	cmd.Flags().StringVar(&expectedDigest, "expected-digest", "", "Expected current Soul digest for CAS")
	return cmd
}

func newAgentSoulHistoryCommand(deps commandDeps) *cobra.Command {
	var (
		limit  int
		cursor string
	)
	cmd := &cobra.Command{
		Use:     "history <agent>",
		Short:   "List managed Soul authoring revisions",
		Example: "  agh agent soul history coder --limit 10 --workspace checkout-api --json",
		Args:    exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			workspace, err := commandWorkspaceFlag(cmd)
			if err != nil {
				return err
			}
			record, err := client.ListAgentSoulHistory(cmd.Context(), args[0], AgentSoulHistoryRequest{
				WorkspaceID: workspace,
				AgentName:   args[0],
				Limit:       limit,
				Cursor:      strings.TrimSpace(cursor),
			})
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, agentSoulHistoryBundle(record))
		},
	}
	addWorkspaceFlag(cmd)
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of revisions to return")
	cmd.Flags().StringVar(&cursor, "cursor", "", "Revision cursor")
	return cmd
}

func newAgentSoulRollbackCommand(deps commandDeps) *cobra.Command {
	var (
		revisionID     string
		expectedDigest string
		idempotencyKey string
	)
	cmd := &cobra.Command{
		Use:     "rollback <agent>",
		Short:   "Rollback SOUL.md to a managed revision",
		Example: "  agh agent soul rollback coder --revision-id rev_123 --expected-digest sha256:old --json",
		Args:    exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			workspace, err := commandWorkspaceFlag(cmd)
			if err != nil {
				return err
			}
			revision, err := changedNonEmptyStringFlag(cmd, "revision-id", revisionID)
			if err != nil {
				return err
			}
			digest, err := changedStringFlag(cmd, "expected-digest", expectedDigest)
			if err != nil {
				return err
			}
			record, err := client.RollbackAgentSoul(cmd.Context(), args[0], AgentSoulRollbackRequest{
				WorkspaceID:    workspace,
				AgentName:      args[0],
				RevisionID:     revision,
				ExpectedDigest: digest,
				IdempotencyKey: strings.TrimSpace(idempotencyKey),
			})
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, agentSoulMutationBundle(&record))
		},
	}
	addWorkspaceFlag(cmd)
	cmd.Flags().StringVar(&revisionID, "revision-id", "", "Managed Soul revision id to restore")
	cmd.Flags().StringVar(&expectedDigest, "expected-digest", "", "Expected current Soul digest for CAS")
	cmd.Flags().StringVar(&idempotencyKey, "idempotency-key", "", "Optional idempotency key")
	return cmd
}

func newAgentHeartbeatCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   authoredContextHeartbeatKey,
		Short: "Inspect and manage agent HEARTBEAT.md files",
	}
	cmd.AddCommand(newAgentHeartbeatInspectCommand(deps))
	cmd.AddCommand(newAgentHeartbeatValidateCommand(deps))
	cmd.AddCommand(newAgentHeartbeatWriteCommand(deps))
	cmd.AddCommand(newAgentHeartbeatDeleteCommand(deps))
	cmd.AddCommand(newAgentHeartbeatHistoryCommand(deps))
	cmd.AddCommand(newAgentHeartbeatRollbackCommand(deps))
	cmd.AddCommand(newAgentHeartbeatStatusCommand(deps))
	cmd.AddCommand(newAgentHeartbeatWakeCommand(deps))
	return cmd
}

func newAgentHeartbeatInspectCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "inspect <agent>",
		Short:   "Inspect one agent's resolved Heartbeat policy",
		Example: "  agh agent heartbeat inspect coder --workspace checkout-api --json",
		Args:    exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			query, err := agentQueryFromCommand(cmd)
			if err != nil {
				return err
			}
			record, err := client.GetAgentHeartbeat(cmd.Context(), args[0], query)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, agentHeartbeatBundle(&record))
		},
	}
	addWorkspaceFlag(cmd)
	return cmd
}

func newAgentHeartbeatValidateCommand(deps commandDeps) *cobra.Command {
	var input authoredBodyInput
	cmd := &cobra.Command{
		Use:     "validate <agent>",
		Short:   "Validate a proposed Heartbeat policy body",
		Example: "  agh agent heartbeat validate coder --file HEARTBEAT.md --workspace checkout-api --json",
		Args:    exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			workspace, err := commandWorkspaceFlag(cmd)
			if err != nil {
				return err
			}
			body, err := readAuthoredBody(cmd, input, true)
			if err != nil {
				return err
			}
			record, err := client.ValidateAgentHeartbeat(cmd.Context(), args[0], AgentHeartbeatValidateRequest{
				WorkspaceID: workspace,
				AgentName:   args[0],
				Body:        body,
			})
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, agentHeartbeatBundle(&record))
		},
	}
	addWorkspaceFlag(cmd)
	addAuthoredBodyFlags(cmd, &input)
	return cmd
}

func newAgentHeartbeatWriteCommand(deps commandDeps) *cobra.Command {
	var (
		input          authoredBodyInput
		expectedDigest string
		ifMatchDigest  string
		idempotencyKey string
	)
	cmd := &cobra.Command{
		Use:     "write <agent>",
		Short:   "Create or replace HEARTBEAT.md through managed authoring",
		Example: "  agh agent heartbeat write coder --file HEARTBEAT.md --expected-digest sha256:old --json",
		Args:    exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			workspace, err := commandWorkspaceFlag(cmd)
			if err != nil {
				return err
			}
			body, err := readAuthoredBody(cmd, input, true)
			if err != nil {
				return err
			}
			digest, err := optionalExpectedDigestFlag(cmd, expectedDigest, ifMatchDigest)
			if err != nil {
				return err
			}
			record, err := client.PutAgentHeartbeat(cmd.Context(), args[0], AgentHeartbeatPutRequest{
				WorkspaceID:    workspace,
				AgentName:      args[0],
				Body:           body,
				ExpectedDigest: digest,
				IdempotencyKey: strings.TrimSpace(idempotencyKey),
			})
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, agentHeartbeatMutationBundle(&record))
		},
	}
	addWorkspaceFlag(cmd)
	addAuthoredBodyFlags(cmd, &input)
	cmd.Flags().StringVar(&expectedDigest, "expected-digest", "", "Expected current Heartbeat digest for CAS")
	cmd.Flags().StringVar(&ifMatchDigest, "if-match", "", "Alias for --expected-digest")
	cmd.Flags().StringVar(&idempotencyKey, "idempotency-key", "", "Optional idempotency key")
	return cmd
}

func newAgentHeartbeatDeleteCommand(deps commandDeps) *cobra.Command {
	var expectedDigest string
	var ifMatchDigest string
	cmd := &cobra.Command{
		Use:     "delete <agent>",
		Short:   "Delete HEARTBEAT.md through managed authoring",
		Example: "  agh agent heartbeat delete coder --expected-digest sha256:old --json",
		Args:    exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			workspace, err := commandWorkspaceFlag(cmd)
			if err != nil {
				return err
			}
			digest, err := changedExpectedDigestFlag(cmd, expectedDigest, ifMatchDigest)
			if err != nil {
				return err
			}
			record, err := client.DeleteAgentHeartbeat(cmd.Context(), args[0], AgentHeartbeatDeleteRequest{
				WorkspaceID:    workspace,
				AgentName:      args[0],
				ExpectedDigest: digest,
			})
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, agentHeartbeatMutationBundle(&record))
		},
	}
	addWorkspaceFlag(cmd)
	cmd.Flags().StringVar(&expectedDigest, "expected-digest", "", "Expected current Heartbeat digest for CAS")
	cmd.Flags().StringVar(&ifMatchDigest, "if-match", "", "Alias for --expected-digest")
	return cmd
}

func newAgentHeartbeatHistoryCommand(deps commandDeps) *cobra.Command {
	var (
		limit  int
		cursor string
	)
	cmd := &cobra.Command{
		Use:     "history <agent>",
		Short:   "List managed Heartbeat authoring revisions",
		Example: "  agh agent heartbeat history coder --limit 10 --workspace checkout-api --json",
		Args:    exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			workspace, err := commandWorkspaceFlag(cmd)
			if err != nil {
				return err
			}
			record, err := client.ListAgentHeartbeatHistory(cmd.Context(), args[0], AgentHeartbeatHistoryRequest{
				WorkspaceID: workspace,
				AgentName:   args[0],
				Limit:       limit,
				Cursor:      strings.TrimSpace(cursor),
			})
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, agentHeartbeatHistoryBundle(record))
		},
	}
	addWorkspaceFlag(cmd)
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of revisions to return")
	cmd.Flags().StringVar(&cursor, "cursor", "", "Revision cursor")
	return cmd
}

func newAgentHeartbeatRollbackCommand(deps commandDeps) *cobra.Command {
	var (
		revisionID     string
		targetDigest   string
		expectedDigest string
		ifMatchDigest  string
		idempotencyKey string
	)
	cmd := &cobra.Command{
		Use:     "rollback <agent>",
		Short:   "Rollback HEARTBEAT.md to a managed revision or snapshot digest",
		Example: "  agh agent heartbeat rollback coder --revision-id rev_123 --expected-digest sha256:old --json",
		Args:    exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			workspace, err := commandWorkspaceFlag(cmd)
			if err != nil {
				return err
			}
			revisionChanged := cmd.Flags().Changed("revision-id")
			targetChanged := cmd.Flags().Changed("target-digest")
			if revisionChanged == targetChanged {
				return errors.New("cli: exactly one of --revision-id or --target-digest is required")
			}
			revision := strings.TrimSpace(revisionID)
			target := strings.TrimSpace(targetDigest)
			if revisionChanged {
				if revision == "" {
					return errors.New("cli: --revision-id cannot be empty")
				}
			}
			if targetChanged {
				if target == "" {
					return errors.New("cli: --target-digest cannot be empty")
				}
			}
			digest, err := changedExpectedDigestFlag(cmd, expectedDigest, ifMatchDigest)
			if err != nil {
				return err
			}
			record, err := client.RollbackAgentHeartbeat(cmd.Context(), args[0], AgentHeartbeatRollbackRequest{
				WorkspaceID:    workspace,
				AgentName:      args[0],
				RevisionID:     revision,
				TargetDigest:   target,
				ExpectedDigest: digest,
				IdempotencyKey: strings.TrimSpace(idempotencyKey),
			})
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, agentHeartbeatMutationBundle(&record))
		},
	}
	addWorkspaceFlag(cmd)
	cmd.Flags().StringVar(&revisionID, "revision-id", "", "Managed Heartbeat revision id to restore")
	cmd.Flags().StringVar(&targetDigest, "target-digest", "", "Heartbeat snapshot digest to restore")
	cmd.Flags().StringVar(&expectedDigest, "expected-digest", "", "Expected current Heartbeat digest for CAS")
	cmd.Flags().StringVar(&ifMatchDigest, "if-match", "", "Alias for --expected-digest")
	cmd.Flags().StringVar(&idempotencyKey, "idempotency-key", "", "Optional idempotency key")
	return cmd
}

func newAgentHeartbeatStatusCommand(deps commandDeps) *cobra.Command {
	var (
		sessionID     string
		includeHealth bool
		includeEvents bool
	)
	cmd := &cobra.Command{
		Use:     "status <agent>",
		Short:   "Read Heartbeat policy status and wake eligibility",
		Example: "  agh agent heartbeat status coder --session sess_123 --workspace checkout-api --json",
		Args:    exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			workspace, err := commandWorkspaceFlag(cmd)
			if err != nil {
				return err
			}
			session := strings.TrimSpace(sessionID)
			record, err := client.GetAgentHeartbeatStatus(cmd.Context(), args[0], AgentHeartbeatStatusRequest{
				WorkspaceID:             workspace,
				AgentName:               args[0],
				SessionID:               session,
				IncludeSessionHealth:    includeHealth || session != "",
				IncludeRecentWakeEvents: includeEvents,
			})
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, agentHeartbeatStatusBundle(record))
		},
	}
	addWorkspaceFlag(cmd)
	cmd.Flags().StringVar(&sessionID, sessionSessionKey, "", "Session id for wake state and health")
	cmd.Flags().BoolVar(
		&includeHealth,
		"include-session-health",
		false,
		"Include session health when a session id is supplied",
	)
	cmd.Flags().BoolVar(&includeEvents, "include-wake-events", false, "Include recent wake audit rows")
	return cmd
}

func newAgentHeartbeatWakeCommand(deps commandDeps) *cobra.Command {
	var (
		sessionID      string
		source         string
		dryRun         bool
		idempotencyKey string
	)
	cmd := &cobra.Command{
		Use:     "wake <agent>",
		Short:   "Request one manual advisory Heartbeat wake",
		Example: "  agh agent heartbeat wake coder --session sess_123 --dry-run --json",
		Args:    exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			workspace, err := commandWorkspaceFlag(cmd)
			if err != nil {
				return err
			}
			session, err := changedNonEmptyStringFlag(cmd, sessionSessionKey, sessionID)
			if err != nil {
				return err
			}
			wakeSource := contract.HeartbeatWakeSource(strings.TrimSpace(source))
			if wakeSource == "" {
				wakeSource = contract.HeartbeatWakeSourceManual
			}
			record, err := client.WakeAgentHeartbeat(cmd.Context(), args[0], AgentHeartbeatWakeRequest{
				WorkspaceID:    workspace,
				AgentName:      args[0],
				SessionID:      session,
				Source:         wakeSource,
				DryRun:         dryRun,
				IdempotencyKey: strings.TrimSpace(idempotencyKey),
			})
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, agentHeartbeatWakeBundle(record))
		},
	}
	addWorkspaceFlag(cmd)
	cmd.Flags().StringVar(&sessionID, sessionSessionKey, "", "Session id to wake")
	cmd.Flags().StringVar(&source, "source", string(contract.HeartbeatWakeSourceManual), "Wake source")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Evaluate wake gates without sending a prompt")
	cmd.Flags().StringVar(&idempotencyKey, "idempotency-key", "", "Optional idempotency key")
	return cmd
}

func newSessionSoulCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   authoredContextSoulKey,
		Short: "Manage session Soul snapshots",
	}
	cmd.AddCommand(newSessionSoulRefreshCommand(deps))
	return cmd
}

func newSessionSoulRefreshCommand(deps commandDeps) *cobra.Command {
	var (
		expectedDigest string
		idempotencyKey string
	)
	cmd := &cobra.Command{
		Use:     "refresh <session-id>",
		Short:   "Refresh an idle session's Soul snapshot",
		Example: "  agh session soul refresh sess_123 --expected-digest sha256:old --json",
		Args:    exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			digest, err := changedStringFlag(cmd, "expected-digest", expectedDigest)
			if err != nil {
				return err
			}
			record, err := client.RefreshSessionSoul(cmd.Context(), args[0], SessionSoulRefreshRequest{
				ExpectedDigest: digest,
				IdempotencyKey: strings.TrimSpace(idempotencyKey),
			})
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, agentSoulBundle(record))
		},
	}
	cmd.Flags().StringVar(&expectedDigest, "expected-digest", "", "Expected current session Soul digest for CAS")
	cmd.Flags().StringVar(&idempotencyKey, "idempotency-key", "", "Optional idempotency key")
	return cmd
}

func newSessionHealthCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "health <session-id>",
		Short:   "Read session health and wake eligibility",
		Example: "  agh session health sess_123 --json",
		Args:    exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			record, err := client.GetSessionHealth(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, sessionHealthBundle(record))
		},
	}
	return cmd
}

func newSessionInspectCommand(deps commandDeps) *cobra.Command {
	var includeEvents bool
	cmd := &cobra.Command{
		Use:     "inspect <session-id>",
		Short:   "Inspect session health, wake audit, and policy correlation",
		Example: "  agh session inspect sess_123 --include-wake-events --json",
		Args:    exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			record, err := client.InspectSession(cmd.Context(), args[0], SessionInspectQuery{
				IncludeRecentWakeEvents: includeEvents,
			})
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, sessionInspectBundle(record))
		},
	}
	cmd.Flags().BoolVar(&includeEvents, "include-wake-events", false, "Include recent wake audit rows")
	return cmd
}

func addWorkspaceFlag(cmd *cobra.Command) {
	cmd.Flags().String(workspaceSkillSource, "", "Resolve the agent from a workspace id, name, or path")
}

func addAuthoredBodyFlags(cmd *cobra.Command, input *authoredBodyInput) {
	cmd.Flags().StringVar(&input.file, "file", "", "Read authored context body from a file")
	cmd.Flags().BoolVar(&input.stdin, "stdin", false, "Read authored context body from stdin")
}

func readAuthoredBody(cmd *cobra.Command, input authoredBodyInput, required bool) (string, error) {
	if cmd.Flags().Changed("file") && input.stdin {
		return "", errors.New("cli: --file and --stdin cannot be combined")
	}
	if input.stdin {
		body, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return "", fmt.Errorf("cli: read authored body from stdin: %w", err)
		}
		return string(body), nil
	}
	if cmd.Flags().Changed("file") {
		path := strings.TrimSpace(input.file)
		if path == "" {
			return "", errors.New("cli: --file cannot be empty")
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("cli: read authored body file: %w", err)
		}
		return string(body), nil
	}
	if required {
		return "", errors.New("cli: provide authored body with --file or --stdin")
	}
	return "", nil
}

func changedStringFlag(cmd *cobra.Command, name string, value string) (string, error) {
	if !cmd.Flags().Changed(name) {
		return "", fmt.Errorf("cli: --%s is required", name)
	}
	return strings.TrimSpace(value), nil
}

func optionalStringFlag(cmd *cobra.Command, name string, value string) string {
	if !cmd.Flags().Changed(name) {
		return ""
	}
	return strings.TrimSpace(value)
}

func optionalExpectedDigestFlag(cmd *cobra.Command, expectedDigest string, ifMatchDigest string) (string, error) {
	expectedChanged := cmd.Flags().Changed("expected-digest")
	ifMatchChanged := cmd.Flags().Changed("if-match")
	if expectedChanged && ifMatchChanged {
		return "", errors.New("cli: use only one of --expected-digest or --if-match")
	}
	if ifMatchChanged {
		return strings.TrimSpace(ifMatchDigest), nil
	}
	if expectedChanged {
		return strings.TrimSpace(expectedDigest), nil
	}
	return "", nil
}

func changedExpectedDigestFlag(cmd *cobra.Command, expectedDigest string, ifMatchDigest string) (string, error) {
	digest, err := optionalExpectedDigestFlag(cmd, expectedDigest, ifMatchDigest)
	if err != nil {
		return "", err
	}
	if !cmd.Flags().Changed("expected-digest") && !cmd.Flags().Changed("if-match") {
		return "", errors.New("cli: --expected-digest or --if-match is required")
	}
	return digest, nil
}

func changedNonEmptyStringFlag(cmd *cobra.Command, name string, value string) (string, error) {
	trimmed, err := changedStringFlag(cmd, name, value)
	if err != nil {
		return "", err
	}
	if trimmed == "" {
		return "", fmt.Errorf("cli: --%s cannot be empty", name)
	}
	return trimmed, nil
}

func agentSoulBundle(record AgentSoulRecord) outputBundle {
	return outputBundle{
		jsonValue: record,
		human: func() (string, error) {
			summary := renderHumanSection("Agent Soul", []keyValue{
				{Label: authoredContextAgentValue, Value: stringOrDash(record.AgentName)},
				{Label: authoredContextEnabledValue, Value: boolString(record.Enabled)},
				{Label: authoredContextPresentValue, Value: boolString(record.Present)},
				{Label: authoredContextActiveValue, Value: boolString(record.Active)},
				{Label: authoredContextValidValue, Value: boolString(record.Valid)},
				{Label: "Validation", Value: stringOrDash(string(record.ValidationStatus))},
				{Label: authoredContextDigestValue, Value: stringOrDash(record.Digest)},
				{Label: authoredContextSnapshotValue, Value: stringOrDash(record.SnapshotID)},
				{Label: "Revision", Value: stringOrDash(record.RevisionID)},
				{Label: authoredContextSourceValue, Value: stringOrDash(record.SourcePath)},
			})
			return renderHumanBlocks(summary, diagnosticsTable(record.Diagnostics)), nil
		},
		toon: func() (string, error) {
			return renderToonObject("agent_soul", []string{
				agentAgentKey,
				automationEnabledKey,
				providerAuthStatePresent,
				authoredContextActiveKey,
				configValidKey,
				"validation_status",
				authoredContextDigestKey,
				"source_path",
			}, []string{
				record.AgentName,
				boolString(record.Enabled),
				boolString(record.Present),
				boolString(record.Active),
				boolString(record.Valid),
				string(record.ValidationStatus),
				record.Digest,
				record.SourcePath,
			}), nil
		},
	}
}

func agentSoulMutationBundle(record *AgentSoulMutationRecord) outputBundle {
	if record == nil {
		empty := AgentSoulMutationRecord{}
		record = &empty
	}
	return outputBundle{
		jsonValue: record,
		human: func() (string, error) {
			revision := renderHumanSection("Soul Revision", []keyValue{
				{Label: "ID", Value: stringOrDash(record.Revision.ID)},
				{Label: authoredContextActionValue, Value: stringOrDash(string(record.Revision.Action))},
				{Label: authoredContextPreviousDigestValue, Value: stringOrDash(record.Revision.PreviousDigest)},
				{Label: authoredContextNewDigestValue, Value: stringOrDash(record.Revision.NewDigest)},
				{Label: authoredContextCreatedValue, Value: stringOrDash(formatTime(record.Revision.CreatedAt))},
			})
			soul, err := agentSoulBundle(record.Soul).human()
			if err != nil {
				return "", err
			}
			return renderHumanBlocks(soul, revision), nil
		},
		toon: func() (string, error) {
			return renderToonObject("agent_soul_revision", []string{
				"id",
				authoredContextActionKey,
				authoredContextPreviousDigestKey,
				authoredContextNewDigestKey,
				automationCreatedAtKey,
			}, []string{
				record.Revision.ID,
				string(record.Revision.Action),
				record.Revision.PreviousDigest,
				record.Revision.NewDigest,
				formatTime(record.Revision.CreatedAt),
			}), nil
		},
	}
}

func agentSoulHistoryBundle(record AgentSoulHistoryRecord) outputBundle {
	return listBundle(
		record,
		record.Revisions,
		"Soul Revisions",
		[]string{
			"ID",
			authoredContextAgentValue,
			authoredContextActionValue,
			authoredContextPreviousDigestValue,
			authoredContextNewDigestValue,
			authoredContextCreatedValue,
		},
		"agent_soul_revisions",
		[]string{
			"id",
			agentAgentKey,
			authoredContextActionKey,
			authoredContextPreviousDigestKey,
			authoredContextNewDigestKey,
			automationCreatedAtKey,
		},
		func(item AgentSoulRevisionRecord) []string {
			return []string{
				stringOrDash(item.ID),
				stringOrDash(item.AgentName),
				stringOrDash(string(item.Action)),
				stringOrDash(item.PreviousDigest),
				stringOrDash(item.NewDigest),
				stringOrDash(formatTime(item.CreatedAt)),
			}
		},
		func(item AgentSoulRevisionRecord) []string {
			return []string{
				item.ID,
				item.AgentName,
				string(item.Action),
				item.PreviousDigest,
				item.NewDigest,
				formatTime(item.CreatedAt),
			}
		},
	)
}

func agentHeartbeatBundle(record *AgentHeartbeatRecord) outputBundle {
	if record == nil {
		empty := AgentHeartbeatRecord{}
		record = &empty
	}
	return outputBundle{
		jsonValue: record,
		human: func() (string, error) {
			summary := renderHumanSection("Agent Heartbeat", []keyValue{
				{Label: authoredContextAgentValue, Value: stringOrDash(record.AgentName)},
				{Label: authoredContextEnabledValue, Value: boolString(record.Enabled)},
				{Label: authoredContextPresentValue, Value: boolString(record.Present)},
				{Label: authoredContextActiveValue, Value: boolString(record.Active)},
				{Label: authoredContextValidValue, Value: boolString(record.Valid)},
				{Label: "Validation", Value: stringOrDash(string(record.ValidationStatus))},
				{Label: authoredContextDigestValue, Value: stringOrDash(record.Digest)},
				{Label: authoredContextConfigDigestValue, Value: stringOrDash(record.ConfigDigest)},
				{Label: authoredContextSnapshotValue, Value: stringOrDash(record.SnapshotID)},
				{Label: authoredContextSourceValue, Value: stringOrDash(record.SourcePath)},
				{Label: authoredContextSummaryValue, Value: stringOrDash(record.Summary)},
			})
			return renderHumanBlocks(summary, diagnosticsTable(record.Diagnostics)), nil
		},
		toon: func() (string, error) {
			return renderToonObject("agent_heartbeat", []string{
				agentAgentKey,
				automationEnabledKey,
				providerAuthStatePresent,
				authoredContextActiveKey,
				configValidKey,
				"validation_status",
				authoredContextDigestKey,
				authoredContextConfigDigestKey,
			}, []string{
				record.AgentName,
				boolString(record.Enabled),
				boolString(record.Present),
				boolString(record.Active),
				boolString(record.Valid),
				string(record.ValidationStatus),
				record.Digest,
				record.ConfigDigest,
			}), nil
		},
	}
}

func agentHeartbeatMutationBundle(record *AgentHeartbeatMutationRecord) outputBundle {
	if record == nil {
		empty := AgentHeartbeatMutationRecord{}
		record = &empty
	}
	return outputBundle{
		jsonValue: record,
		human: func() (string, error) {
			revision := renderHumanSection("Heartbeat Revision", []keyValue{
				{Label: "ID", Value: stringOrDash(record.Revision.ID)},
				{Label: authoredContextOperationValue, Value: stringOrDash(string(record.Revision.Operation))},
				{Label: authoredContextPreviousDigestValue, Value: stringOrDash(record.Revision.PreviousDigest)},
				{Label: authoredContextNewDigestValue, Value: stringOrDash(record.Revision.NewDigest)},
				{Label: authoredContextSnapshotValue, Value: stringOrDash(record.Revision.NewSnapshotID)},
				{Label: authoredContextCreatedValue, Value: stringOrDash(formatTime(record.Revision.CreatedAt))},
			})
			heartbeat, err := agentHeartbeatBundle(&record.Heartbeat).human()
			if err != nil {
				return "", err
			}
			return renderHumanBlocks(heartbeat, revision), nil
		},
		toon: func() (string, error) {
			return renderToonObject("agent_heartbeat_revision", []string{
				"id",
				authoredContextOperationKey,
				authoredContextPreviousDigestKey,
				authoredContextNewDigestKey,
				"snapshot",
				automationCreatedAtKey,
			}, []string{
				record.Revision.ID,
				string(record.Revision.Operation),
				record.Revision.PreviousDigest,
				record.Revision.NewDigest,
				record.Revision.NewSnapshotID,
				formatTime(record.Revision.CreatedAt),
			}), nil
		},
	}
}

func agentHeartbeatHistoryBundle(record AgentHeartbeatHistoryRecord) outputBundle {
	return listBundle(
		record,
		record.Revisions,
		"Heartbeat Revisions",
		[]string{
			"ID",
			authoredContextAgentValue,
			authoredContextOperationValue,
			authoredContextPreviousDigestValue,
			authoredContextNewDigestValue,
			authoredContextCreatedValue,
		},
		"agent_heartbeat_revisions",
		[]string{
			"id",
			agentAgentKey,
			authoredContextOperationKey,
			authoredContextPreviousDigestKey,
			authoredContextNewDigestKey,
			automationCreatedAtKey,
		},
		func(item AgentHeartbeatRevisionRecord) []string {
			return []string{
				stringOrDash(item.ID),
				stringOrDash(item.AgentName),
				stringOrDash(string(item.Operation)),
				stringOrDash(item.PreviousDigest),
				stringOrDash(item.NewDigest),
				stringOrDash(formatTime(item.CreatedAt)),
			}
		},
		func(item AgentHeartbeatRevisionRecord) []string {
			return []string{
				item.ID,
				item.AgentName,
				string(item.Operation),
				item.PreviousDigest,
				item.NewDigest,
				formatTime(item.CreatedAt),
			}
		},
	)
}

func agentHeartbeatStatusBundle(record AgentHeartbeatStatusRecord) outputBundle {
	return outputBundle{
		jsonValue: record,
		human: func() (string, error) {
			summary := renderHumanSection("Heartbeat Status", []keyValue{
				{Label: authoredContextAgentValue, Value: stringOrDash(record.AgentName)},
				{Label: authoredContextEnabledValue, Value: boolString(record.Enabled)},
				{Label: authoredContextPresentValue, Value: boolString(record.Present)},
				{Label: authoredContextActiveValue, Value: boolString(record.Active)},
				{Label: authoredContextValidValue, Value: boolString(record.Valid)},
				{Label: authoredContextDigestValue, Value: stringOrDash(record.Digest)},
				{Label: authoredContextConfigDigestValue, Value: stringOrDash(record.ConfigDigest)},
				{Label: authoredContextSummaryValue, Value: stringOrDash(record.Summary)},
			})
			blocks := []string{summary, diagnosticsTable(record.Diagnostics)}
			if record.WakeState != nil {
				blocks = append(blocks, wakeStateSection("Wake State", *record.WakeState))
			}
			if record.SessionHealth != nil {
				blocks = append(blocks, sessionHealthSection("Session Health", *record.SessionHealth))
			}
			if len(record.WakeEvents) > 0 {
				blocks = append(blocks, wakeEventsTable(record.WakeEvents))
			}
			return renderHumanBlocks(blocks...), nil
		},
		toon: func() (string, error) {
			return renderToonObject("agent_heartbeat_status", []string{
				agentAgentKey,
				automationEnabledKey,
				providerAuthStatePresent,
				authoredContextActiveKey,
				configValidKey,
				authoredContextDigestKey,
				authoredContextConfigDigestKey,
			}, []string{
				record.AgentName,
				boolString(record.Enabled),
				boolString(record.Present),
				boolString(record.Active),
				boolString(record.Valid),
				record.Digest,
				record.ConfigDigest,
			}), nil
		},
	}
}

func agentHeartbeatWakeBundle(record AgentHeartbeatWakeDecisionRecord) outputBundle {
	return outputBundle{
		jsonValue: record,
		human: func() (string, error) {
			return renderHumanSection("Heartbeat Wake", []keyValue{
				{Label: authoredContextEventValue, Value: stringOrDash(record.WakeEventID)},
				{Label: authoredContextResultValue, Value: stringOrDash(string(record.Result))},
				{Label: authoredContextReasonValue, Value: stringOrDash(string(record.Reason))},
				{Label: "Policy Snapshot", Value: stringOrDash(record.PolicySnapshotID)},
				{Label: "Policy Digest", Value: stringOrDash(record.PolicyDigest)},
				{Label: authoredContextConfigDigestValue, Value: stringOrDash(record.ConfigDigest)},
				{Label: "Synthetic Prompt", Value: stringOrDash(record.SyntheticPromptID)},
			}), nil
		},
		toon: func() (string, error) {
			return renderToonObject("agent_heartbeat_wake", []string{
				authoredContextEventKey, "result", memoryReasonKey, "policy_digest", authoredContextConfigDigestKey,
			}, []string{
				record.WakeEventID,
				string(record.Result),
				string(record.Reason),
				record.PolicyDigest,
				record.ConfigDigest,
			}), nil
		},
	}
}

func sessionHealthBundle(record SessionHealthRecord) outputBundle {
	return outputBundle{
		jsonValue: record,
		human: func() (string, error) {
			return sessionHealthSection("Session Health", record), nil
		},
		toon: func() (string, error) {
			return renderToonObject("session_health", []string{
				sessionSessionKey,
				workspaceSkillSource,
				agentAgentKey,
				networkStateKey,
				authoredContextHealthKey,
				"eligible_for_wake",
				memoryReasonKey,
			}, []string{
				record.SessionID,
				record.WorkspaceID,
				record.AgentName,
				string(record.State),
				string(record.Health),
				boolString(record.EligibleForWake),
				string(record.IneligibilityReason),
			}), nil
		},
	}
}

func sessionStatusBundle(record SessionStatusRecord) outputBundle {
	return outputBundle{
		jsonValue: record,
		human: func() (string, error) {
			status := renderHumanSection("Session Status", []keyValue{
				{Label: authoredContextSessionValue, Value: stringOrDash(record.SessionID)},
				{Label: authoredContextWorkspaceValue, Value: stringOrDash(record.WorkspaceID)},
				{Label: authoredContextAgentValue, Value: stringOrDash(record.AgentName)},
				{Label: authoredContextStateValue, Value: stringOrDash(string(record.State))},
				{Label: authoredContextHealthValue, Value: stringOrDash(string(record.Health))},
				{Label: "Active Prompt", Value: boolString(record.ActivePrompt)},
				{Label: "Attachable", Value: boolString(record.Attachable)},
				{Label: "Eligible For Wake", Value: boolString(record.EligibleForWake)},
				{Label: "Ineligibility Reason", Value: stringOrDash(string(record.IneligibilityReason))},
				{Label: authoredContextUpdatedValue, Value: stringOrDash(formatTime(record.UpdatedAt))},
			})
			if record.WakeState == nil {
				return status, nil
			}
			return renderHumanBlocks(status, wakeStateSection("Wake State", *record.WakeState)), nil
		},
		toon: func() (string, error) {
			return renderToonObject("session_status", []string{
				sessionSessionKey,
				workspaceSkillSource,
				agentAgentKey,
				networkStateKey,
				authoredContextHealthKey,
				"eligible_for_wake",
				memoryReasonKey,
			}, []string{
				record.SessionID,
				record.WorkspaceID,
				record.AgentName,
				string(record.State),
				string(record.Health),
				boolString(record.EligibleForWake),
				string(record.IneligibilityReason),
			}), nil
		},
	}
}

func sessionInspectBundle(record SessionInspectRecord) outputBundle {
	return outputBundle{
		jsonValue: record,
		human: func() (string, error) {
			blocks := []string{
				sessionHealthSection("Session Health", record.Health),
				renderHumanSection("Policy Correlation", []keyValue{
					{Label: "Policy Digest", Value: stringOrDash(record.PolicyDigest)},
					{Label: authoredContextConfigDigestValue, Value: stringOrDash(record.ConfigDigest)},
				}),
				diagnosticsTable(record.Diagnostics),
			}
			if record.WakeState != nil {
				blocks = append(blocks, wakeStateSection("Wake State", *record.WakeState))
			}
			if len(record.WakeEvents) > 0 {
				blocks = append(blocks, wakeEventsTable(record.WakeEvents))
			}
			return renderHumanBlocks(blocks...), nil
		},
		toon: func() (string, error) {
			return renderToonObject("session_inspect", []string{
				sessionSessionKey, "policy_digest", authoredContextConfigDigestKey, "wake_events",
			}, []string{
				record.SessionID,
				record.PolicyDigest,
				record.ConfigDigest,
				strconv.Itoa(len(record.WakeEvents)),
			}), nil
		},
	}
}

func sessionHealthSection(title string, record SessionHealthRecord) string {
	return renderHumanSection(title, []keyValue{
		{Label: authoredContextSessionValue, Value: stringOrDash(record.SessionID)},
		{Label: authoredContextWorkspaceValue, Value: stringOrDash(record.WorkspaceID)},
		{Label: authoredContextAgentValue, Value: stringOrDash(record.AgentName)},
		{Label: authoredContextStateValue, Value: stringOrDash(string(record.State))},
		{Label: authoredContextHealthValue, Value: stringOrDash(string(record.Health))},
		{Label: "Active Prompt", Value: boolString(record.ActivePrompt)},
		{Label: "Attachable", Value: boolString(record.Attachable)},
		{Label: "Eligible For Wake", Value: boolString(record.EligibleForWake)},
		{Label: "Ineligibility Reason", Value: stringOrDash(string(record.IneligibilityReason))},
		{Label: authoredContextLastActivityValue, Value: stringOrDash(formatTimePtr(record.LastActivityAt))},
		{Label: "Last Presence", Value: stringOrDash(formatTimePtr(record.LastPresenceAt))},
		{Label: authoredContextUpdatedValue, Value: stringOrDash(formatTime(record.UpdatedAt))},
	})
}

func wakeStateSection(title string, state contract.HeartbeatWakeStatePayload) string {
	return renderHumanSection(title, []keyValue{
		{Label: authoredContextSessionValue, Value: stringOrDash(state.SessionID)},
		{Label: authoredContextAgentValue, Value: stringOrDash(state.AgentName)},
		{Label: "Policy Snapshot", Value: stringOrDash(state.PolicySnapshotID)},
		{Label: "Last Wake", Value: stringOrDash(formatTimePtr(state.LastWakeAt))},
		{Label: "Next Allowed", Value: stringOrDash(formatTimePtr(state.NextAllowedAt))},
		{Label: "Coalesced", Value: strconv.Itoa(state.CoalescedCount)},
		{Label: "Last Result", Value: stringOrDash(string(state.LastResult))},
		{Label: "Last Reason", Value: stringOrDash(string(state.LastReason))},
		{Label: authoredContextUpdatedValue, Value: stringOrDash(formatTime(state.UpdatedAt))},
	})
}

func diagnosticsTable(items []contract.AuthoredContextDiagnosticPayload) string {
	if len(items) == 0 {
		return ""
	}
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		location := firstNonEmpty(item.Field, item.Section)
		rows = append(rows, []string{
			stringOrDash(string(item.Severity)),
			stringOrDash(item.Code),
			stringOrDash(location),
			stringOrDash(item.Message),
		})
	}
	return renderHumanTable(
		"Diagnostics",
		[]string{"Severity", cliCodeValue, "Location", authoredContextMessageValue},
		rows,
	)
}

func wakeEventsTable(items []contract.HeartbeatWakeEventPayload) string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			stringOrDash(item.ID),
			stringOrDash(item.SessionID),
			stringOrDash(string(item.Source)),
			stringOrDash(string(item.Result)),
			stringOrDash(string(item.Reason)),
			stringOrDash(formatTime(item.CreatedAt)),
		})
	}
	return renderHumanTable(
		"Wake Events",
		[]string{
			"ID",
			authoredContextSessionValue,
			authoredContextSourceValue,
			authoredContextResultValue,
			authoredContextReasonValue,
			authoredContextCreatedValue,
		},
		rows,
	)
}
