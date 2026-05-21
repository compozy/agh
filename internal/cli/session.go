package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/session"
	"github.com/spf13/cobra"
)

const (
	sessionTurnIDKey = "turn_id"
)

const (
	sessionSequenceKey = "sequence"
)

const (
	sessionAgentValue     = "Agent"
	sessionBackendValue   = "Backend"
	sessionChannelValue   = "Channel"
	sessionCreatedValue   = "Created"
	sessionNameValue      = "Name"
	sessionProfileValue   = "Profile"
	sessionProviderValue  = "Provider"
	sessionSessionValue   = "Session"
	sessionStateValue     = "State"
	sessionStatusValue    = "Status"
	sessionTimestampValue = "Timestamp"
	sessionUpdatedValue   = "Updated"
	sessionWorkspaceValue = "Workspace"
	sessionAgentNameKey   = "agent_name"
	sessionChannelKey     = "channel"
	sessionCreatedAtKey   = "created_at"
	sessionHistoryIDValue = "history <id>"
	sessionListKey        = "list"
	sessionNameKey        = "name"
	sessionNewKey         = "new"
	sessionProviderKey    = "provider"
	sessionSessionIDKey   = "session_id"
	sessionStateKey       = "state"
	sessionStatusKey      = "status"
	sessionUpdatedAtKey   = "updated_at"
)

func newSessionCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   sessionSessionKey,
		Short: "Manage AGH sessions",
	}

	cmd.AddCommand(newSessionCreateCommand(deps))
	cmd.AddCommand(newSessionListCommand(deps))
	cmd.AddCommand(newSessionStopCommand(deps))
	cmd.AddCommand(newSessionSoulCommand(deps))
	cmd.AddCommand(newSessionHealthCommand(deps))
	cmd.AddCommand(newSessionStatusCommand(deps))
	cmd.AddCommand(newSessionInspectCommand(deps))
	cmd.AddCommand(newSessionResumeCommand(deps))
	cmd.AddCommand(newSessionRecapCommand(deps))
	cmd.AddCommand(newSessionRepairCommand(deps))
	cmd.AddCommand(newSessionApproveCommand(deps))
	cmd.AddCommand(newSessionWaitCommand(deps))
	cmd.AddCommand(newSessionPromptCommand(deps))
	cmd.AddCommand(newSessionEventsCommand(deps))
	cmd.AddCommand(newSessionHistoryCommand(deps))

	return cmd
}

func newSessionCreateCommand(deps commandDeps) *cobra.Command {
	var (
		agentName       string
		cwd             string
		name            string
		channel         string
		provider        string
		model           string
		reasoningEffort string
		workspaceRef    string
	)

	cmd := &cobra.Command{
		Use:   sessionNewKey,
		Short: "Create a new session",
		Example: `  # Start a session in the current workspace using the configured default agent
  agh session new

  # Start a named session for a specific registered workspace and agent
  agh session new --workspace checkout-api --agent reviewer --name review-api

  # Override provider, model, and reasoning effort for this session only
  agh session new --provider codex --model gpt-5.4 --reasoning-effort high

  # Auto-register an absolute workspace path before creating the session
  agh session new --cwd "$PWD" --agent reviewer`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			workspace, workspacePath, err := resolveSessionCreateWorkspace(deps, workspaceRef, cwd)
			if err != nil {
				return err
			}

			created, err := client.CreateSession(cmd.Context(), CreateSessionRequest{
				AgentName:       agentName,
				Provider:        strings.TrimSpace(provider),
				Model:           strings.TrimSpace(model),
				ReasoningEffort: strings.TrimSpace(reasoningEffort),
				Name:            name,
				Workspace:       workspace,
				WorkspacePath:   workspacePath,
				Channel:         strings.TrimSpace(channel),
			})
			if err != nil {
				return err
			}

			return writeCommandOutput(cmd, sessionBundle(created, deps.now))
		},
	}
	cmd.Flags().StringVar(&agentName, "agent", "", "Agent definition name (defaults to config default)")
	cmd.Flags().StringVar(&workspaceRef, workspaceSkillSource, "", "Registered workspace name or ID")
	cmd.Flags().StringVar(&cwd, "cwd", "", "Absolute workspace directory to auto-register")
	cmd.Flags().StringVar(&name, sessionNameKey, "", "Optional session label")
	cmd.Flags().StringVar(&channel, sessionChannelKey, "", "Optional network channel opt-in for the session")
	cmd.Flags().StringVar(&provider, sessionProviderKey, "", "Optional provider override for this session")
	cmd.Flags().StringVar(&model, "model", "", "Optional model override for this session")
	cmd.Flags().StringVar(
		&reasoningEffort,
		"reasoning-effort",
		"",
		"Optional reasoning effort hint (minimal|low|medium|high|xhigh) for providers that support it",
	)
	return cmd
}

func newSessionListCommand(deps commandDeps) *cobra.Command {
	var (
		includeAll      bool
		workspaceFilter string
		resumable       bool
		limit           int
	)

	cmd := &cobra.Command{
		Use:   sessionListKey,
		Short: "List sessions",
		Example: `  # List active sessions
  agh session list

  # Include stopped sessions and filter to a workspace
  agh session list --all --workspace checkout-api`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			sessions, err := client.ListSessions(cmd.Context(), SessionListQuery{
				Workspace: workspaceFilter,
				Resumable: resumable,
				Limit:     limit,
				Sort:      sessionListSortKey(resumable),
			})
			if err != nil {
				return err
			}
			if !includeAll && !resumable {
				sessions = filterActiveSessions(sessions)
			}

			return writeCommandOutput(cmd, sessionListBundle(sessions, deps.now))
		},
	}
	cmd.Flags().BoolVar(&includeAll, "all", false, "Include stopped sessions")
	cmd.Flags().StringVar(&workspaceFilter, workspaceSkillSource, "", "Filter by workspace name or ID")
	cmd.Flags().BoolVar(&resumable, "resumable", false, "Show only sessions eligible for resume attach")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum sessions to return")
	return cmd
}

func sessionListSortKey(resumable bool) string {
	if resumable {
		return "last_activity"
	}
	return ""
}

func newSessionStopCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "stop <id>",
		Short: "Stop a session",
		Example: `  # Stop a running session
  agh session stop sess_1234`,
		Args: exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			if err := client.StopSession(cmd.Context(), args[0]); err != nil {
				return err
			}

			info, err := client.GetSession(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, sessionBundle(info, deps.now))
		},
	}
}

func newSessionStatusCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "status <id>",
		Short: "Show session status",
		Example: `  # Show current state for one session
  agh session status sess_1234

  # Read status as JSON for scripts
  agh session status sess_1234 -o json`,
		Args: exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			info, err := client.GetSessionStatus(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, sessionStatusBundle(info))
		},
	}
}

func newSessionResumeCommand(deps commandDeps) *cobra.Command {
	var (
		latest          bool
		workspaceFilter string
	)
	cmd := &cobra.Command{
		Use:   "resume [id]",
		Short: "Attach to a resumable session",
		Example: `  # Attach to a resumable session by ID
  agh session resume sess_1234

  # Attach to the latest eligible session in a workspace
  agh session resume --latest --workspace checkout-api`,
		Args: sessionResumeArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			if latest && len(args) > 0 {
				return errors.New("cli: --latest cannot be combined with a session id")
			}
			if !latest && len(args) == 0 {
				return errors.New("session_resume_ambiguous: pass a session id or --latest")
			}
			sessionID := ""
			if latest {
				candidates, err := client.ListSessions(cmd.Context(), SessionListQuery{
					Workspace: workspaceFilter,
					Resumable: true,
					Sort:      "last_activity",
					Limit:     1,
				})
				if err != nil {
					return err
				}
				if len(candidates) == 0 {
					return writeCommandOutput(cmd, sessionResumeEmptyBundle())
				}
				sessionID = candidates[0].ID
			} else {
				sessionID = args[0]
			}
			info, err := client.ResumeSession(cmd.Context(), sessionID)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, sessionBundle(info, deps.now))
		},
	}
	cmd.Flags().BoolVar(&latest, "latest", false, "Attach to the latest eligible session")
	cmd.Flags().StringVar(&workspaceFilter, workspaceSkillSource, "", "Filter --latest by workspace name or ID")
	return cmd
}

func sessionResumeArgs(_ *cobra.Command, args []string) error {
	if len(args) > 1 {
		return errors.New("cli: expected at most one session id")
	}
	if len(args) == 1 && strings.TrimSpace(args[0]) == "" {
		return errors.New("cli: session id is required")
	}
	return nil
}

func newSessionRecapCommand(deps commandDeps) *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:   "recap <id>",
		Short: "Show deterministic session recap",
		Example: `  # Show a bounded recap for one session
  agh session recap sess_1234 --limit 20`,
		Args: exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			recap, err := client.SessionRecap(cmd.Context(), args[0], limit)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, sessionRecapBundle(recap))
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum recent messages to include")
	return cmd
}

func newSessionRepairCommand(deps commandDeps) *cobra.Command {
	var (
		dryRun bool
		force  bool
	)

	cmd := &cobra.Command{
		Use:   "repair <id>",
		Short: "Inspect and repair an interrupted session transcript",
		Example: `  # Report the repair actions without writing new events
  agh session repair sess_1234 --dry-run

  # Force repair for a stopped session whose stop reason is not crash or error
  agh session repair sess_1234 --force`,
		Args: exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			result, err := client.RepairSession(cmd.Context(), args[0], SessionRepairQuery{
				DryRun: dryRun,
				Force:  force,
			})
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, sessionRepairBundle(result))
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Report planned repairs without persisting events")
	cmd.Flags().BoolVar(&force, "force", false, "Allow repair for stopped non-crash sessions")
	return cmd
}

func newSessionApproveCommand(deps commandDeps) *cobra.Command {
	var request SessionApprovalRequest
	cmd := &cobra.Command{
		Use:   "approve <id>",
		Short: "Approve or reject a pending session permission request",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			request.RequestID = strings.TrimSpace(request.RequestID)
			request.TurnID = strings.TrimSpace(request.TurnID)
			request.Decision = strings.TrimSpace(request.Decision)
			if request.RequestID == "" && request.TurnID == "" {
				return errors.New("cli: --request-id or --turn-id is required")
			}
			if request.Decision == "" {
				return errors.New("cli: --decision is required")
			}
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			result, err := client.ApproveSession(cmd.Context(), args[0], request)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, sessionApprovalBundle(args[0], result))
		},
	}
	cmd.Flags().StringVar(&request.RequestID, "request-id", "", "Pending permission request id")
	cmd.Flags().StringVar(&request.TurnID, "turn-id", "", "Pending permission turn id")
	cmd.Flags().
		StringVar(&request.Decision, "decision", "", "Decision: allow-once, allow-always, reject-once, or reject-always")
	mustMarkFlagRequired(cmd, "decision")
	return cmd
}

func newSessionWaitCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "wait <id>",
		Short: "Block until a session stops",
		Example: `  # Block until a session emits its stopped event
  agh session wait sess_1234`,
		Args: exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			info, err := client.GetSession(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if info.State != session.StateStopped {
				err = client.StreamSessionEvents(
					cmd.Context(),
					args[0],
					SessionEventQuery{},
					"",
					func(event SSEEvent) error {
						if event.Event == session.EventTypeSessionStopped {
							return errStopSSE
						}
						return nil
					},
				)
				if err != nil && !errors.Is(err, errStopSSE) {
					return err
				}
				info, err = client.GetSession(cmd.Context(), args[0])
				if err != nil {
					return err
				}
			}

			return writeCommandOutput(cmd, sessionBundle(info, deps.now))
		},
	}
}

func newSessionPromptCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "prompt <id> <message>",
		Short: "Send a prompt to a session",
		Example: `  # Send a follow-up prompt to an active session
  agh session prompt sess_1234 "Summarize the current changes."`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			mode, err := resolveOutputFormat(cmd)
			if err != nil {
				return err
			}
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			if mode == OutputJSONL {
				return streamPromptEventsJSONL(cmd, client, args[0], args[1])
			}

			events, err := client.PromptSession(cmd.Context(), args[0], args[1])
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, agentEventsBundle(events))
		},
	}
}

func newSessionEventsCommand(deps commandDeps) *cobra.Command {
	var (
		eventType string
		last      int
		sinceRaw  string
		follow    bool
	)

	cmd := &cobra.Command{
		Use:   "events <id>",
		Short: "Read session events",
		Example: `  # Read the latest stored events
  agh session events sess_1234 --last 20

  # Follow new events until interrupted
  agh session events sess_1234 --follow`,
		Args: exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			since, err := parseSinceFlag(sinceRaw, deps.now)
			if err != nil {
				return err
			}
			query := SessionEventQuery{
				Type:  eventType,
				Last:  last,
				Since: since,
			}

			if follow {
				return streamSessionEvents(cmd, client, args[0], query)
			}

			events, err := client.SessionEvents(cmd.Context(), args[0], query)
			if err != nil {
				return err
			}
			return writeSessionEventsOutput(cmd, events)
		},
	}
	cmd.Flags().StringVar(&eventType, extensionTypeKey, "", "Filter by event type")
	cmd.Flags().IntVar(&last, "last", 0, "Show only the most recent N events")
	cmd.Flags().StringVar(&sinceRaw, "since", "", "Show events since an RFC3339 timestamp or relative duration")
	cmd.Flags().BoolVar(&follow, "follow", false, "Stream new events over SSE")
	return cmd
}

func streamPromptEventsJSONL(cmd *cobra.Command, client DaemonClient, id string, message string) error {
	return client.StreamPromptSession(cmd.Context(), id, message, func(event SSEEvent) error {
		if len(event.Data) == 0 || strings.TrimSpace(string(event.Data)) == "[DONE]" {
			return nil
		}

		var payload any
		if err := json.Unmarshal(event.Data, &payload); err != nil {
			return fmt.Errorf("cli: decode prompt event: %w", err)
		}
		return writeJSONLine(cmd, payload)
	})
}

func writeSessionEventsOutput(cmd *cobra.Command, events []SessionEventRecord) error {
	mode, err := resolveOutputFormat(cmd)
	if err != nil {
		return err
	}
	if mode == OutputJSONL {
		return writeJSONLines(cmd, events)
	}
	return writeCommandOutput(cmd, sessionEventsBundle(events))
}

func newSessionHistoryCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   sessionHistoryIDValue,
		Short: "Show session history grouped by turn",
		Example: `  # Show replayable turn history for one session
  agh session history sess_1234`,
		Args: exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			history, err := client.SessionHistory(cmd.Context(), args[0], SessionEventQuery{})
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, sessionHistoryBundle(history))
		},
	}
}

func streamSessionEvents(cmd *cobra.Command, client DaemonClient, id string, query SessionEventQuery) error {
	mode, err := resolveOutputFormat(cmd)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetEscapeHTML(false)

	err = client.StreamSessionEvents(cmd.Context(), id, query, "", func(event SSEEvent) error {
		var payload SessionEventRecord
		if len(event.Data) > 0 {
			if err := json.Unmarshal(event.Data, &payload); err != nil {
				return fmt.Errorf("cli: decode session stream event: %w", err)
			}
		}
		if payload.Type == "" {
			payload.Type = event.Event
		}

		switch mode {
		case OutputJSON, OutputJSONL:
			if err := encoder.Encode(payload); err != nil {
				return err
			}
		case OutputToon:
			if err := writeRawCommandOutput(cmd, renderToonObject("event", []string{
				sessionSequenceKey,
				extensionTypeKey,
				sessionAgentNameKey,
				sessionTurnIDKey,
				networkTimestampKey,
				memoryContentKey,
			}, []string{
				strconv.FormatInt(payload.Sequence, 10),
				payload.Type,
				payload.AgentName,
				payload.TurnID,
				formatTime(payload.Timestamp),
				compactJSON(payload.Content),
			})); err != nil {
				return err
			}
		default:
			if err := writeRawCommandOutput(cmd, strings.Join([]string{
				stringOrDash(formatTime(payload.Timestamp)),
				stringOrDash(payload.Type),
				stringOrDash(payload.AgentName),
				stringOrDash(compactJSON(payload.Content)),
			}, "  ")); err != nil {
				return err
			}
		}

		if payload.Type == session.EventTypeSessionStopped {
			return errStopSSE
		}
		return nil
	})
	if errors.Is(err, errStopSSE) {
		return nil
	}
	return err
}

func sessionBundle(info SessionRecord, now func() time.Time) outputBundle {
	return outputBundle{
		jsonValue: info,
		human: func() (string, error) {
			base := renderHumanSection(sessionSessionValue, []keyValue{
				{Label: "ID", Value: stringOrDash(info.ID)},
				{Label: sessionNameValue, Value: stringOrDash(info.Name)},
				{Label: sessionAgentValue, Value: stringOrDash(info.AgentName)},
				{Label: sessionProviderValue, Value: stringOrDash(info.Provider)},
				{Label: sessionWorkspaceValue, Value: stringOrDash(displaySessionWorkspace(info))},
				{Label: sessionChannelValue, Value: stringOrDash(info.Channel)},
				{Label: sessionStateValue, Value: stringOrDash(string(info.State))},
				{Label: "Badge", Value: stringOrDash(string(info.Badge))},
				{Label: "Attached To", Value: stringOrDash(info.AttachedTo)},
				{Label: "Attach Expires", Value: stringOrDash(formatTimePtr(info.AttachExpiresAt))},
				{Label: "Stop Reason", Value: stringOrDash(string(info.StopReason))},
				{Label: "Stop Detail", Value: stringOrDash(info.StopDetail)},
				{Label: "Failure Kind", Value: stringOrDash(sessionFailureKind(info))},
				{Label: "Failure Summary", Value: stringOrDash(sessionFailureSummary(info))},
				{Label: "Crash Bundle", Value: stringOrDash(sessionCrashBundlePath(info))},
				{Label: "ACP Session", Value: stringOrDash(info.ACPSessionID)},
				{Label: sessionCreatedValue, Value: stringOrDash(formatTime(info.CreatedAt))},
				{Label: sessionUpdatedValue, Value: stringOrDash(formatTime(info.UpdatedAt))},
				{Label: "Age", Value: stringOrDash(formatAge(now, info.CreatedAt))},
			})

			blocks := []string{base}
			if info.Sandbox != nil {
				blocks = append(blocks, renderHumanSection("Sandbox", []keyValue{
					{Label: sessionBackendValue, Value: stringOrDash(info.Sandbox.Backend)},
					{Label: sessionProfileValue, Value: stringOrDash(info.Sandbox.Profile)},
					{Label: "Sandbox ID", Value: stringOrDash(info.Sandbox.SandboxID)},
					{Label: "Instance ID", Value: stringOrDash(info.Sandbox.InstanceID)},
					{Label: sessionStateValue, Value: stringOrDash(info.Sandbox.State)},
					{Label: "Last Sync Error", Value: stringOrDash(info.Sandbox.LastSyncError)},
				}))
			}
			if info.ACPCaps == nil {
				return renderHumanBlocks(blocks...), nil
			}
			caps := renderHumanSection("Capabilities", []keyValue{
				{Label: "Supports Load", Value: strconv.FormatBool(info.ACPCaps.SupportsLoadSession)},
				{Label: "Modes", Value: stringOrDash(strings.Join(info.ACPCaps.SupportedModes, ", "))},
				{Label: "Models", Value: stringOrDash(strings.Join(info.ACPCaps.SupportedModels, ", "))},
			})
			blocks = append(blocks, caps)
			return renderHumanBlocks(blocks...), nil
		},
		toon: func() (string, error) {
			return renderToonObject(sessionSessionKey, []string{
				"id",
				sessionNameKey,
				sessionAgentNameKey,
				sessionProviderKey,
				"sandbox_backend",
				workspaceSkillSource,
				sessionChannelKey,
				sessionStateKey,
				"badge",
				"attached_to",
				"attach_expires_at",
				"stop_reason",
				"failure_kind",
				"failure_summary",
				"crash_bundle_path",
				"acp_session_id",
				sessionCreatedAtKey,
				sessionUpdatedAtKey,
			}, []string{
				info.ID,
				info.Name,
				info.AgentName,
				info.Provider,
				sessionSandboxBackend(info),
				displaySessionWorkspace(info),
				info.Channel,
				string(info.State),
				string(info.Badge),
				info.AttachedTo,
				formatTimePtr(info.AttachExpiresAt),
				string(info.StopReason),
				sessionFailureKind(info),
				sessionFailureSummary(info),
				sessionCrashBundlePath(info),
				info.ACPSessionID,
				formatTime(info.CreatedAt),
				formatTime(info.UpdatedAt),
			}), nil
		},
	}
}

func sessionListBundle(items []SessionRecord, now func() time.Time) outputBundle {
	return listBundle(
		items,
		items,
		"Sessions",
		[]string{
			"ID",
			sessionNameValue,
			sessionAgentValue,
			sessionProviderValue,
			sessionBackendValue,
			sessionStateValue,
			"Badge",
			"Failure",
			sessionWorkspaceValue,
			sessionChannelValue,
			sessionUpdatedValue,
		},
		"sessions",
		[]string{
			"id",
			sessionNameKey,
			sessionAgentNameKey,
			sessionProviderKey,
			"sandbox_backend",
			sessionStateKey,
			"badge",
			"failure_kind",
			workspaceSkillSource,
			sessionChannelKey,
			sessionUpdatedAtKey,
		},
		func(item SessionRecord) []string {
			return []string{
				stringOrDash(item.ID),
				stringOrDash(item.Name),
				stringOrDash(item.AgentName),
				stringOrDash(item.Provider),
				stringOrDash(sessionSandboxBackend(item)),
				stringOrDash(string(item.State)),
				stringOrDash(string(item.Badge)),
				stringOrDash(sessionFailureKind(item)),
				stringOrDash(displaySessionWorkspace(item)),
				stringOrDash(item.Channel),
				stringOrDash(formatAge(now, item.UpdatedAt)),
			}
		},
		func(item SessionRecord) []string {
			return []string{
				item.ID,
				item.Name,
				item.AgentName,
				item.Provider,
				sessionSandboxBackend(item),
				string(item.State),
				string(item.Badge),
				sessionFailureKind(item),
				displaySessionWorkspace(item),
				item.Channel,
				formatTime(item.UpdatedAt),
			}
		},
	)
}

func sessionResumeEmptyBundle() outputBundle {
	payload := struct {
		Resumed *SessionRecord `json:"resumed"`
		Reason  string         `json:"reason"`
	}{
		Reason: "no_eligible_sessions",
	}
	return outputBundle{
		jsonValue: payload,
		human: func() (string, error) {
			return "No resumable sessions; start a new one with 'agh session new'.", nil
		},
		toon: func() (string, error) {
			return renderToonObject("resume", []string{"resumed", "reason"}, []string{"", payload.Reason}), nil
		},
	}
}

func sessionRecapBundle(record SessionRecapRecord) outputBundle {
	return outputBundle{
		jsonValue: record,
		human: func() (string, error) {
			blocks := []string{
				renderHumanSection("Session Recap", []keyValue{
					{Label: sessionSessionValue, Value: stringOrDash(record.Session.ID)},
					{Label: "Badge", Value: stringOrDash(string(record.Session.Badge))},
					{Label: "Markers", Value: strconv.Itoa(len(record.RecentMarkers))},
					{Label: "Messages", Value: strconv.Itoa(len(record.RecentMessages))},
					{Label: "Event Cursor", Value: strconv.FormatInt(record.Snapshot.EventCursor, 10)},
					{Label: "Consistency", Value: stringOrDash(record.Snapshot.Consistency)},
				}),
			}
			if len(record.RecentMarkers) > 0 {
				items := make([]keyValue, 0, len(record.RecentMarkers))
				for _, marker := range record.RecentMarkers {
					items = append(items, keyValue{
						Label: stringOrDash(marker.Kind),
						Value: stringOrDash(marker.Summary),
					})
				}
				blocks = append(blocks, renderHumanSection("Recent Markers", items))
			}
			return renderHumanBlocks(blocks...), nil
		},
		toon: func() (string, error) {
			return renderToonObject("recap", []string{
				sessionSessionIDKey,
				"badge",
				"markers",
				"messages",
				"event_cursor",
				"consistency",
			}, []string{
				record.Session.ID,
				string(record.Session.Badge),
				strconv.Itoa(len(record.RecentMarkers)),
				strconv.Itoa(len(record.RecentMessages)),
				strconv.FormatInt(record.Snapshot.EventCursor, 10),
				record.Snapshot.Consistency,
			}), nil
		},
	}
}

func sessionSandboxBackend(info SessionRecord) string {
	if info.Sandbox == nil {
		return ""
	}
	return strings.TrimSpace(info.Sandbox.Backend)
}

func sessionRepairBundle(record SessionRepairRecord) outputBundle {
	return outputBundle{
		jsonValue: record,
		human: func() (string, error) {
			return renderHumanSection("Session Repair", []keyValue{
				{Label: sessionSessionValue, Value: stringOrDash(record.SessionID)},
				{Label: "Persisted", Value: strconv.FormatBool(record.Persisted)},
				{Label: "Issues", Value: stringOrDash(sessionRepairIssueSummary(record.Issues))},
				{Label: "Actions", Value: stringOrDash(sessionRepairActionSummary(record.Actions))},
			}), nil
		},
		toon: func() (string, error) {
			return renderToonObject("repair", []string{
				sessionSessionIDKey,
				"persisted",
				"issues",
				"actions",
			}, []string{
				record.SessionID,
				strconv.FormatBool(record.Persisted),
				sessionRepairIssueSummary(record.Issues),
				sessionRepairActionSummary(record.Actions),
			}), nil
		},
	}
}

func sessionApprovalBundle(sessionID string, record SessionApprovalRecord) outputBundle {
	payload := struct {
		SessionID string `json:"session_id"`
		Status    string `json:"status"`
	}{
		SessionID: strings.TrimSpace(sessionID),
		Status:    strings.TrimSpace(record.Status),
	}
	return outputBundle{
		jsonValue: payload,
		human: func() (string, error) {
			return renderHumanSection("Session Approval", []keyValue{
				{Label: sessionSessionValue, Value: stringOrDash(payload.SessionID)},
				{Label: sessionStatusValue, Value: stringOrDash(payload.Status)},
			}), nil
		},
		toon: func() (string, error) {
			return renderToonObject(
				"session_approval",
				[]string{sessionSessionIDKey, sessionStatusKey},
				[]string{payload.SessionID, payload.Status},
			), nil
		},
	}
}

func sessionRepairIssueSummary(items []SessionRepairIssueRecord) string {
	if len(items) == 0 {
		return ""
	}
	parts := make([]string, 0, len(items))
	for _, item := range items {
		parts = append(parts, repairSummaryPart(item.Code, item.TurnID, item.EventID))
	}
	return strings.Join(parts, ", ")
}

func sessionRepairActionSummary(items []SessionRepairActionRecord) string {
	if len(items) == 0 {
		return ""
	}
	parts := make([]string, 0, len(items))
	for _, item := range items {
		ref := item.EventID
		if ref == "" {
			ref = item.ToolCallID
		}
		parts = append(parts, repairSummaryPart(item.Code, item.TurnID, ref))
	}
	return strings.Join(parts, ", ")
}

func repairSummaryPart(code string, turnID string, ref string) string {
	value := strings.TrimSpace(code)
	if trimmedTurn := strings.TrimSpace(turnID); trimmedTurn != "" {
		value += ":" + trimmedTurn
	}
	if trimmedRef := strings.TrimSpace(ref); trimmedRef != "" {
		value += ":" + trimmedRef
	}
	return value
}

func sessionFailureKind(info SessionRecord) string {
	if info.Failure == nil {
		return ""
	}
	return strings.TrimSpace(string(info.Failure.Kind))
}

func sessionFailureSummary(info SessionRecord) string {
	if info.Failure == nil {
		return ""
	}
	return strings.TrimSpace(info.Failure.Summary)
}

func sessionCrashBundlePath(info SessionRecord) string {
	if info.Failure == nil {
		return ""
	}
	return strings.TrimSpace(info.Failure.CrashBundlePath)
}

func sessionEventsBundle(events []SessionEventRecord) outputBundle {
	return listBundle(
		events,
		events,
		"Session Events",
		[]string{"Seq", sessionTypeValue, sessionAgentValue, "Turn", sessionTimestampValue, "Content"},
		"events",
		[]string{
			sessionSequenceKey,
			extensionTypeKey,
			sessionAgentNameKey,
			sessionTurnIDKey,
			networkTimestampKey,
			memoryContentKey,
		},
		func(event SessionEventRecord) []string {
			return []string{
				strconv.FormatInt(event.Sequence, 10),
				stringOrDash(event.Type),
				stringOrDash(event.AgentName),
				stringOrDash(event.TurnID),
				stringOrDash(formatTime(event.Timestamp)),
				stringOrDash(compactJSON(event.Content)),
			}
		},
		func(event SessionEventRecord) []string {
			return []string{
				strconv.FormatInt(event.Sequence, 10),
				event.Type,
				event.AgentName,
				event.TurnID,
				formatTime(event.Timestamp),
				compactJSON(event.Content),
			}
		},
	)
}

func sessionHistoryBundle(history []TurnHistoryRecord) outputBundle {
	flattened := flattenHistory(history)
	return listBundle(
		history,
		flattened,
		"Session History",
		[]string{"Turn", "Seq", sessionTypeValue, sessionAgentValue, sessionTimestampValue, "Content"},
		"history",
		[]string{
			sessionTurnIDKey,
			sessionSequenceKey,
			extensionTypeKey,
			sessionAgentNameKey,
			networkTimestampKey,
			memoryContentKey,
		},
		func(event SessionEventRecord) []string {
			return []string{
				stringOrDash(event.TurnID),
				strconv.FormatInt(event.Sequence, 10),
				stringOrDash(event.Type),
				stringOrDash(event.AgentName),
				stringOrDash(formatTime(event.Timestamp)),
				stringOrDash(compactJSON(event.Content)),
			}
		},
		func(event SessionEventRecord) []string {
			return []string{
				event.TurnID,
				strconv.FormatInt(event.Sequence, 10),
				event.Type,
				event.AgentName,
				formatTime(event.Timestamp),
				compactJSON(event.Content),
			}
		},
	)
}

func agentEventsBundle(events []AgentEventRecord) outputBundle {
	return listBundle(
		events,
		events,
		"Prompt Events",
		[]string{sessionTimestampValue, sessionTypeValue, "Detail", "Stop"},
		"prompt_events",
		[]string{networkTimestampKey, extensionTypeKey, "detail", "stop_reason"},
		func(event AgentEventRecord) []string {
			return []string{
				stringOrDash(formatTime(event.Timestamp)),
				stringOrDash(event.Type),
				stringOrDash(firstNonEmpty(event.Text, event.Title, event.Error, compactJSON(event.Raw))),
				stringOrDash(event.StopReason),
			}
		},
		func(event AgentEventRecord) []string {
			return []string{
				formatTime(event.Timestamp),
				event.Type,
				firstNonEmpty(event.Text, event.Title, event.Error, compactJSON(event.Raw)),
				event.StopReason,
			}
		},
	)
}

func filterActiveSessions(items []SessionRecord) []SessionRecord {
	filtered := make([]SessionRecord, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(string(item.State)) == string(session.StateStopped) {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func displaySessionWorkspace(info SessionRecord) string {
	return firstNonEmpty(strings.TrimSpace(info.WorkspacePath), strings.TrimSpace(info.WorkspaceID))
}

func resolveSessionCreateWorkspace(deps commandDeps, workspaceRef string, cwd string) (string, string, error) {
	trimmedWorkspace := strings.TrimSpace(workspaceRef)
	trimmedCWD := strings.TrimSpace(cwd)

	switch {
	case trimmedWorkspace != "" && trimmedCWD != "":
		return "", "", errors.New("cli: --workspace and --cwd are mutually exclusive")
	case trimmedWorkspace != "":
		return trimmedWorkspace, "", nil
	case trimmedCWD != "":
		if !filepath.IsAbs(trimmedCWD) {
			return "", "", fmt.Errorf("cli: --cwd must be an absolute path: %q", trimmedCWD)
		}
		return "", filepath.Clean(trimmedCWD), nil
	default:
		workspacePath, err := currentWorkingDirectory(deps)
		if err != nil {
			return "", "", err
		}
		return "", workspacePath, nil
	}
}

func flattenHistory(history []TurnHistoryRecord) []SessionEventRecord {
	flattened := make([]SessionEventRecord, 0)
	for _, turn := range history {
		for _, event := range turn.Events {
			if strings.TrimSpace(event.TurnID) == "" {
				event.TurnID = turn.TurnID
			}
			flattened = append(flattened, event)
		}
	}
	return flattened
}

func parseSinceFlag(raw string, now func() time.Time) (time.Time, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}, nil
	}

	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.UTC(), nil
		}
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return time.Time{}, fmt.Errorf("cli: invalid since value %q", value)
	}
	if duration < 0 {
		return time.Time{}, fmt.Errorf("cli: relative since must be positive: %q", value)
	}

	current := time.Now().UTC()
	if now != nil {
		current = now().UTC()
	}
	return current.Add(-duration), nil
}
