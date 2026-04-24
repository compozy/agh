package cli

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func newObserveCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "observe",
		Short: "Query global observability state",
	}

	cmd.AddCommand(newObserveEventsCommand(deps))
	cmd.AddCommand(newObserveHealthCommand(deps))
	return cmd
}

func newObserveEventsCommand(deps commandDeps) *cobra.Command {
	var (
		session string
		agent   string
		typ     string
		since   string
		last    int
		follow  bool
	)

	cmd := &cobra.Command{
		Use:   "events",
		Short: "Read cross-session observability events",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			sinceTime, err := parseSinceFlag(since, deps.now)
			if err != nil {
				return err
			}
			query := ObserveEventQuery{
				SessionID: session,
				AgentName: agent,
				Type:      typ,
				Since:     sinceTime,
				Last:      last,
			}

			if follow {
				return streamObserveEvents(cmd, client, query)
			}

			events, err := client.ObserveEvents(cmd.Context(), query)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, observeEventsBundle(events))
		},
	}
	cmd.Flags().StringVar(&session, "session", "", "Filter by session id")
	cmd.Flags().StringVar(&agent, "agent", "", "Filter by agent name")
	cmd.Flags().StringVar(&typ, "type", "", "Filter by event type")
	cmd.Flags().StringVar(&since, "since", "", "Show events since an RFC3339 timestamp or relative duration")
	cmd.Flags().IntVar(&last, "last", 0, "Show only the most recent N events")
	cmd.Flags().BoolVar(&follow, "follow", false, "Stream new events over SSE")
	return cmd
}

func newObserveHealthCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "Show observability health",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			health, err := client.ObserveHealth(cmd.Context())
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, observeHealthBundle(health))
		},
	}
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
		case OutputJSON:
			if err := encoder.Encode(payload); err != nil {
				return err
			}
		case OutputToon:
			if err := writeRawCommandOutput(cmd, renderToonObject("observe_event", []string{
				"id", "session_id", "type", "agent_name", "summary", "timestamp",
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
		[]string{"ID", "Session", "Type", "Agent", "Summary", "Timestamp"},
		"observe_events",
		[]string{"id", "session_id", "type", "agent_name", "summary", "timestamp"},
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

func observeHealthBundle(health HealthStatus) outputBundle {
	return outputBundle{
		jsonValue: health,
		human: func() (string, error) {
			return renderHumanSection("Observe Health", []keyValue{
				{Label: "Status", Value: stringOrDash(health.Status)},
				{Label: "Uptime Seconds", Value: int64OrDash(health.UptimeSeconds)},
				{Label: "Active Sessions", Value: strconv.Itoa(health.ActiveSessions)},
				{Label: "Active Agents", Value: strconv.Itoa(health.ActiveAgents)},
				{Label: "Global DB Bytes", Value: int64OrDash(health.GlobalDBSizeBytes)},
				{Label: "Session DB Bytes", Value: int64OrDash(health.SessionDBSizeBytes)},
				{Label: "Persistence", Value: stringOrDash(health.Persistence.Status)},
				{Label: "Retention", Value: stringOrDash(observeRetentionSummary(health))},
				{Label: "Retention Last Sweep", Value: stringOrDash(formatTimePtr(health.Retention.LastSweepAt))},
				{Label: "Version", Value: stringOrDash(health.Version)},
			}), nil
		},
		toon: func() (string, error) {
			return renderToonObject("observe_health", []string{
				"status",
				"uptime_seconds",
				"active_sessions",
				"active_agents",
				"global_db_size_bytes",
				"session_db_size_bytes",
				"persistence",
				"retention",
				"version",
			}, []string{
				health.Status,
				strconv.FormatInt(health.UptimeSeconds, 10),
				strconv.Itoa(health.ActiveSessions),
				strconv.Itoa(health.ActiveAgents),
				strconv.FormatInt(health.GlobalDBSizeBytes, 10),
				strconv.FormatInt(health.SessionDBSizeBytes, 10),
				health.Persistence.Status,
				observeRetentionSummary(health),
				health.Version,
			}), nil
		},
	}
}

func observeRetentionSummary(health HealthStatus) string {
	retention := health.Retention
	if !retention.Enabled {
		return "disabled"
	}
	return fmt.Sprintf(
		"%s (%d days, deleted %d rows)",
		stringOrDash(retention.LastSweepStatus),
		retention.RetentionDays,
		retention.DeletedEventSummaries+retention.DeletedTokenStats+retention.DeletedPermissionLogRows,
	)
}
