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

func newSessionCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "session",
		Short: "Manage AGH sessions",
	}

	cmd.AddCommand(newSessionCreateCommand(deps))
	cmd.AddCommand(newSessionListCommand(deps))
	cmd.AddCommand(newSessionStopCommand(deps))
	cmd.AddCommand(newSessionStatusCommand(deps))
	cmd.AddCommand(newSessionResumeCommand(deps))
	cmd.AddCommand(newSessionWaitCommand(deps))
	cmd.AddCommand(newSessionPromptCommand(deps))
	cmd.AddCommand(newSessionEventsCommand(deps))
	cmd.AddCommand(newSessionHistoryCommand(deps))

	return cmd
}

func newSessionCreateCommand(deps commandDeps) *cobra.Command {
	var (
		agentName    string
		cwd          string
		name         string
		workspaceRef string
	)

	cmd := &cobra.Command{
		Use:   "new",
		Short: "Create a new session",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, _, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			workspace, workspacePath, err := resolveSessionCreateWorkspace(deps, workspaceRef, cwd)
			if err != nil {
				return err
			}

			created, err := client.CreateSession(cmd.Context(), CreateSessionRequest{
				AgentName:     agentName,
				Name:          name,
				Workspace:     workspace,
				WorkspacePath: workspacePath,
			})
			if err != nil {
				return err
			}

			return writeCommandOutput(cmd, sessionBundle(created, deps.now))
		},
	}
	cmd.Flags().StringVar(&agentName, "agent", "", "Agent definition name (defaults to config default)")
	cmd.Flags().StringVar(&workspaceRef, "workspace", "", "Registered workspace name or ID")
	cmd.Flags().StringVar(&cwd, "cwd", "", "Absolute workspace directory to auto-register")
	cmd.Flags().StringVar(&name, "name", "", "Optional session label")
	return cmd
}

func newSessionListCommand(deps commandDeps) *cobra.Command {
	var (
		includeAll      bool
		workspaceFilter string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List sessions",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, _, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			sessions, err := client.ListSessions(cmd.Context(), SessionListQuery{
				Workspace: workspaceFilter,
			})
			if err != nil {
				return err
			}
			if !includeAll {
				sessions = filterActiveSessions(sessions)
			}

			return writeCommandOutput(cmd, sessionListBundle(sessions, deps.now))
		},
	}
	cmd.Flags().BoolVar(&includeAll, "all", false, "Include stopped sessions")
	cmd.Flags().StringVar(&workspaceFilter, "workspace", "", "Filter by workspace name or ID")
	return cmd
}

func newSessionStopCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "stop <id>",
		Short: "Stop a session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := clientFromDeps(deps)
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
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := clientFromDeps(deps)
			if err != nil {
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

func newSessionResumeCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "resume <id>",
		Short: "Resume a stopped session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			info, err := client.ResumeSession(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, sessionBundle(info, deps.now))
		},
	}
}

func newSessionWaitCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "wait <id>",
		Short: "Block until a session stops",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			info, err := client.GetSession(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if info.State != string(session.StateStopped) {
				err = client.StreamSessionEvents(cmd.Context(), args[0], SessionEventQuery{}, "", func(event SSEEvent) error {
					if event.Event == session.EventTypeSessionStopped {
						return errStopSSE
					}
					return nil
				})
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
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := clientFromDeps(deps)
			if err != nil {
				return err
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
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := clientFromDeps(deps)
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
			return writeCommandOutput(cmd, sessionEventsBundle(events))
		},
	}
	cmd.Flags().StringVar(&eventType, "type", "", "Filter by event type")
	cmd.Flags().IntVar(&last, "last", 0, "Show only the most recent N events")
	cmd.Flags().StringVar(&sinceRaw, "since", "", "Show events since an RFC3339 timestamp or relative duration")
	cmd.Flags().BoolVar(&follow, "follow", false, "Stream new events over SSE")
	return cmd
}

func newSessionHistoryCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "history <id>",
		Short: "Show session history grouped by turn",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := clientFromDeps(deps)
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
	lastEventID := ""

	err = client.StreamSessionEvents(cmd.Context(), id, query, "", func(event SSEEvent) error {
		if strings.TrimSpace(event.ID) != "" {
			lastEventID = event.ID
			_ = lastEventID
		}

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
		case OutputJSON:
			if err := encoder.Encode(payload); err != nil {
				return err
			}
		case OutputToon:
			if err := writeRawCommandOutput(cmd, renderToonObject("event", []string{
				"sequence", "type", "agent_name", "turn_id", "timestamp", "content",
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
			base := renderHumanSection("Session", []keyValue{
				{Label: "ID", Value: stringOrDash(info.ID)},
				{Label: "Name", Value: stringOrDash(info.Name)},
				{Label: "Agent", Value: stringOrDash(info.AgentName)},
				{Label: "Workspace", Value: stringOrDash(displaySessionWorkspace(info))},
				{Label: "State", Value: stringOrDash(info.State)},
				{Label: "ACP Session", Value: stringOrDash(info.ACPSessionID)},
				{Label: "Created", Value: stringOrDash(formatTime(info.CreatedAt))},
				{Label: "Updated", Value: stringOrDash(formatTime(info.UpdatedAt))},
				{Label: "Age", Value: stringOrDash(formatAge(now, info.CreatedAt))},
			})

			if info.ACPCaps == nil {
				return base, nil
			}
			caps := renderHumanSection("Capabilities", []keyValue{
				{Label: "Supports Load", Value: strconv.FormatBool(info.ACPCaps.SupportsLoadSession)},
				{Label: "Modes", Value: stringOrDash(strings.Join(info.ACPCaps.SupportedModes, ", "))},
				{Label: "Models", Value: stringOrDash(strings.Join(info.ACPCaps.SupportedModels, ", "))},
			})
			return renderHumanBlocks(base, caps), nil
		},
		toon: func() (string, error) {
			return renderToonObject("session", []string{
				"id", "name", "agent_name", "workspace", "state", "acp_session_id", "created_at", "updated_at",
			}, []string{
				info.ID,
				info.Name,
				info.AgentName,
				displaySessionWorkspace(info),
				info.State,
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
		[]string{"ID", "Name", "Agent", "State", "Workspace", "Updated"},
		"sessions",
		[]string{"id", "name", "agent_name", "state", "workspace", "updated_at"},
		func(item SessionRecord) []string {
			return []string{
				stringOrDash(item.ID),
				stringOrDash(item.Name),
				stringOrDash(item.AgentName),
				stringOrDash(item.State),
				stringOrDash(displaySessionWorkspace(item)),
				stringOrDash(formatAge(now, item.UpdatedAt)),
			}
		},
		func(item SessionRecord) []string {
			return []string{
				item.ID,
				item.Name,
				item.AgentName,
				item.State,
				displaySessionWorkspace(item),
				formatTime(item.UpdatedAt),
			}
		},
	)
}

func sessionEventsBundle(events []SessionEventRecord) outputBundle {
	return listBundle(
		events,
		events,
		"Session Events",
		[]string{"Seq", "Type", "Agent", "Turn", "Timestamp", "Content"},
		"events",
		[]string{"sequence", "type", "agent_name", "turn_id", "timestamp", "content"},
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
		[]string{"Turn", "Seq", "Type", "Agent", "Timestamp", "Content"},
		"history",
		[]string{"turn_id", "sequence", "type", "agent_name", "timestamp", "content"},
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
		[]string{"Timestamp", "Type", "Detail", "Stop"},
		"prompt_events",
		[]string{"timestamp", "type", "detail", "stop_reason"},
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
		if strings.TrimSpace(item.State) == string(session.StateStopped) {
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
