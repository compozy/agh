package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

const (
	observeTypeKey = "type"
)

const (
	observeAgentValue   = "Agent"
	observeAgentNameKey = "agent_name"
	observeEventsKey    = "events"
	observeObserveKey   = "observe"
)

func newObserveCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   observeObserveKey,
		Short: "Query observability state",
	}

	cmd.AddCommand(newObserveEventsCommand(deps))
	return cmd
}

func newObserveEventsCommand(deps commandDeps) *cobra.Command {
	var (
		session      string
		agent        string
		typ          string
		since        string
		last         int
		follow       bool
		workspaceRef string
	)

	cmd := &cobra.Command{
		Use:   observeEventsKey,
		Short: "Read cross-session observability events",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			workspace, err := resolveCLIWorkspaceRouteRef(cmd.Context(), deps, client, workspaceRef)
			if err != nil {
				return err
			}

			sinceTime, err := parseSinceFlag(since, deps.now)
			if err != nil {
				return err
			}
			query := ObserveEventQuery{
				WorkspaceRef: workspace,
				SessionID:    session,
				AgentName:    agent,
				Type:         typ,
				Since:        sinceTime,
				Last:         last,
			}

			if follow {
				return streamObserveEvents(cmd, client, query)
			}

			events, err := client.ObserveEvents(cmd.Context(), query)
			if err != nil {
				return err
			}
			return writeObserveEventsOutput(cmd, events)
		},
	}
	cmd.Flags().StringVar(&session, "session", "", "Filter by session id")
	cmd.Flags().StringVar(&agent, "agent", "", "Filter by agent name")
	cmd.Flags().StringVar(&typ, observeTypeKey, "", "Filter by event type")
	cmd.Flags().StringVar(&workspaceRef, "workspace", "", "Workspace root, name, or ID for scoped events")
	cmd.Flags().StringVar(&since, "since", "", "Show events since an RFC3339 timestamp or relative duration")
	cmd.Flags().IntVar(&last, "last", 0, "Show only the most recent N events")
	cmd.Flags().BoolVar(&follow, "follow", false, "Stream new events over SSE")
	return cmd
}

func writeObserveEventsOutput(cmd *cobra.Command, events []ObserveEventRecord) error {
	mode, err := resolveOutputFormat(cmd)
	if err != nil {
		return err
	}
	if mode == OutputJSONL {
		return writeJSONLines(cmd, events)
	}
	return writeCommandOutput(cmd, observeEventsBundle(events))
}

func streamObserveEvents(cmd *cobra.Command, client DaemonClient, query ObserveEventQuery) error {
	mode, err := resolveOutputFormat(cmd)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetEscapeHTML(false)

	return client.StreamObserveEvents(cmd.Context(), query, "", func(event SSEEvent) error {
		var payload ObserveEventRecord
		if len(event.Data) > 0 {
			if err := json.Unmarshal(event.Data, &payload); err != nil {
				return fmt.Errorf("cli: decode observe stream event: %w", err)
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
			if err := writeRawCommandOutput(cmd, renderToonObject("observe_event", []string{
				"id", automationSessionIDKey, observeTypeKey, observeAgentNameKey, "summary", networkTimestampKey,
			}, []string{
				payload.ID,
				payload.SessionID,
				payload.Type,
				payload.AgentName,
				payload.Summary,
				formatTime(payload.Timestamp),
			})); err != nil {
				return err
			}
		default:
			if err := writeRawCommandOutput(cmd, strings.Join([]string{
				stringOrDash(formatTime(payload.Timestamp)),
				stringOrDash(payload.Type),
				stringOrDash(payload.SessionID),
				stringOrDash(payload.AgentName),
				stringOrDash(payload.Summary),
			}, "  ")); err != nil {
				return err
			}
		}
		return nil
	})
}

func observeEventsBundle(events []ObserveEventRecord) outputBundle {
	return listBundle(
		events,
		events,
		"Observability Events",
		[]string{"ID", agentKernelSessionValue, sessionTypeValue, observeAgentValue, "Summary", "Timestamp"},
		"observe_events",
		[]string{"id", automationSessionIDKey, observeTypeKey, observeAgentNameKey, "summary", networkTimestampKey},
		func(event ObserveEventRecord) []string {
			return []string{
				stringOrDash(event.ID),
				stringOrDash(event.SessionID),
				stringOrDash(event.Type),
				stringOrDash(event.AgentName),
				stringOrDash(event.Summary),
				stringOrDash(formatTime(event.Timestamp)),
			}
		},
		func(event ObserveEventRecord) []string {
			return []string{
				event.ID,
				event.SessionID,
				event.Type,
				event.AgentName,
				event.Summary,
				formatTime(event.Timestamp),
			}
		},
	)
}
