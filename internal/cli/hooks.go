package cli

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/spf13/cobra"
)

func newHooksCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hooks",
		Short: "Inspect configured and executed hooks",
	}

	cmd.AddCommand(newHooksListCommand(deps))
	cmd.AddCommand(newHooksInfoCommand(deps))
	cmd.AddCommand(newHooksEventsCommand(deps))
	cmd.AddCommand(newHooksRunsCommand(deps))

	return cmd
}

func newHooksListCommand(deps commandDeps) *cobra.Command {
	var query HookCatalogQuery

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List resolved hooks in pipeline order",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			hooks, err := client.HookCatalog(cmd.Context(), query)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, hookListBundle(hooks))
		},
	}

	cmd.Flags().StringVar(&query.Workspace, "workspace", "", "Filter by workspace name or ID")
	cmd.Flags().StringVar(&query.Agent, "agent", "", "Filter by agent name")
	cmd.Flags().StringVar(&query.Event, "event", "", "Filter by hook event")
	cmd.Flags().StringVar(&query.Source, "source", "", "Filter by hook source")
	cmd.Flags().StringVar(&query.Mode, "mode", "", "Filter by hook mode")
	return cmd
}

func newHooksInfoCommand(deps commandDeps) *cobra.Command {
	var workspace string

	cmd := &cobra.Command{
		Use:   "info <name>",
		Short: "Show detailed information for one or more hooks by name",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			hooks, err := client.HookCatalog(cmd.Context(), HookCatalogQuery{Workspace: workspace})
			if err != nil {
				return err
			}

			name := strings.TrimSpace(args[0])
			matches := make([]HookCatalogRecord, 0)
			for _, hook := range hooks {
				if strings.TrimSpace(hook.Name) == name {
					matches = append(matches, hook)
				}
			}
			if len(matches) == 0 {
				return fmt.Errorf("cli: no hooks named %q found", name)
			}

			return writeCommandOutput(cmd, hookInfoBundle(matches))
		},
	}

	cmd.Flags().StringVar(&workspace, "workspace", "", "Resolve hooks in one workspace context")
	return cmd
}

func newHooksEventsCommand(deps commandDeps) *cobra.Command {
	var query HookEventsQuery

	cmd := &cobra.Command{
		Use:   "events",
		Short: "List supported hook events",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			events, err := client.HookEvents(cmd.Context(), query)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, hookEventsBundle(events))
		},
	}

	cmd.Flags().StringVar(&query.Family, "family", "", "Filter by hook event family")
	cmd.Flags().BoolVar(&query.SyncOnly, "sync-only", false, "Show only sync-eligible events")
	return cmd
}

func newHooksRunsCommand(deps commandDeps) *cobra.Command {
	var (
		query        HookRunsQuery
		sinceRaw     string
		workspaceRef string
	)

	cmd := &cobra.Command{
		Use:   "runs",
		Short: "Show persisted hook execution history",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			if strings.TrimSpace(query.Session) == "" {
				return errors.New("cli: --session is required")
			}

			since, err := parseSinceFlag(sinceRaw, deps.now)
			if err != nil {
				return err
			}
			if !since.IsZero() {
				query.Since = since.UTC().Format(time.RFC3339Nano)
			}

			workspace, err := resolveCLIWorkspaceRouteRef(cmd.Context(), deps, client, workspaceRef)
			if err != nil {
				return err
			}
			runs, err := client.HookRuns(cmd.Context(), workspace, query)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, hookRunsBundle(runs))
		},
	}

	cmd.Flags().StringVar(&query.Session, "session", "", "Session ID")
	cmd.Flags().StringVar(&query.Event, "event", "", "Filter by hook event")
	cmd.Flags().StringVar(&query.Outcome, "outcome", "", "Filter by hook outcome")
	cmd.Flags().StringVar(&sinceRaw, "since", "", "Show runs since an RFC3339 timestamp or relative duration")
	cmd.Flags().IntVar(&query.Last, "last", 0, "Show only the most recent N runs")
	cmd.Flags().StringVar(&workspaceRef, "workspace", "", "Workspace name, ID, or path")
	return cmd
}

func hookListBundle(hooks []HookCatalogRecord) outputBundle {
	return listBundle(
		hooks,
		hooks,
		"Hooks",
		[]string{"Order", "Name", "Event", "Source", "Mode", "Priority"},
		"hooks",
		[]string{"order", "name", "event", "source", "skill_source", "mode", "required", "priority"},
		func(item HookCatalogRecord) []string {
			return []string{
				strconv.Itoa(item.Order),
				stringOrDash(item.Name),
				stringOrDash(item.Event),
				stringOrDash(item.Source),
				stringOrDash(item.Mode),
				strconv.Itoa(item.Priority),
			}
		},
		func(item HookCatalogRecord) []string {
			return []string{
				strconv.Itoa(item.Order),
				item.Name,
				item.Event,
				item.Source,
				item.SkillSource,
				item.Mode,
				strconv.FormatBool(item.Required),
				strconv.Itoa(item.Priority),
			}
		},
	)
}

func hookInfoBundle(hooks []HookCatalogRecord) outputBundle {
	return outputBundle{
		jsonValue: hooks,
		human: func() (string, error) {
			blocks := make([]string, 0, len(hooks))
			for _, item := range hooks {
				blocks = append(blocks, renderHumanBlocks(
					renderHumanSection("Hook", []keyValue{
						{Label: "Name", Value: stringOrDash(item.Name)},
						{Label: "Order", Value: strconv.Itoa(item.Order)},
						{Label: "Event", Value: stringOrDash(item.Event)},
						{Label: "Source", Value: stringOrDash(item.Source)},
						{Label: "Skill Source", Value: stringOrDash(item.SkillSource)},
						{Label: "Mode", Value: stringOrDash(item.Mode)},
						{Label: "Required", Value: strconv.FormatBool(item.Required)},
						{Label: "Priority", Value: strconv.Itoa(item.Priority)},
						{Label: "Timeout (ms)", Value: int64OrDash(item.TimeoutMS)},
						{Label: "Executor Kind", Value: stringOrDash(item.ExecutorKind)},
					}),
					renderHumanTable("Matcher", []string{"Field", "Value"}, hookMatcherRows(item.Matcher)),
					renderHumanTable("Metadata", []string{"Key", "Value"}, hookMetadataRows(item.Metadata)),
				))
			}
			return renderHumanBlocks(blocks...), nil
		},
		toon: func() (string, error) {
			rows := make([][]string, 0, len(hooks))
			for _, item := range hooks {
				rows = append(rows, []string{
					item.Name,
					strconv.Itoa(item.Order),
					item.Event,
					item.Source,
					item.SkillSource,
					item.Mode,
					strconv.FormatBool(item.Required),
					strconv.Itoa(item.Priority),
					strconv.FormatInt(item.TimeoutMS, 10),
					item.ExecutorKind,
				})
			}

			blocks := []string{
				renderToonArray(
					"hooks",
					[]string{
						"name",
						"order",
						"event",
						"source",
						"skill_source",
						"mode",
						"required",
						"priority",
						"timeout_ms",
						"executor_kind",
					},
					rows,
				),
			}
			for _, item := range hooks {
				blocks = append(blocks, renderHumanBlocks(
					renderToonObject("hook", []string{"name", "order", "event"}, []string{
						item.Name,
						strconv.Itoa(item.Order),
						item.Event,
					}),
					renderToonArray("matcher", []string{"field", "value"}, hookMatcherRows(item.Matcher)),
					renderToonArray("metadata", []string{"key", "value"}, hookMetadataRows(item.Metadata)),
				))
			}
			return renderHumanBlocks(blocks...), nil
		},
	}
}

func hookEventsBundle(events []HookEventRecord) outputBundle {
	return listBundle(
		events,
		events,
		"Hook Events",
		[]string{"Event", "Family", "Sync", "Payload", "Patch"},
		"events",
		[]string{"event", "family", "sync_eligible", "payload_schema", "patch_schema"},
		func(item HookEventRecord) []string {
			return []string{
				stringOrDash(item.Event),
				stringOrDash(item.Family),
				strconv.FormatBool(item.SyncEligible),
				stringOrDash(item.PayloadSchema),
				stringOrDash(item.PatchSchema),
			}
		},
		func(item HookEventRecord) []string {
			return []string{
				item.Event,
				item.Family,
				strconv.FormatBool(item.SyncEligible),
				item.PayloadSchema,
				item.PatchSchema,
			}
		},
	)
}

func hookRunsBundle(runs []HookRunRecord) outputBundle {
	return listBundle(
		runs,
		runs,
		"Hook Runs",
		[]string{"Hook", "Event", "Outcome", "Duration", "Error"},
		"runs",
		[]string{"hook_name", "event", "outcome", "duration_ms", "error", "recorded_at"},
		func(item HookRunRecord) []string {
			return []string{
				stringOrDash(item.HookName),
				stringOrDash(item.Event),
				stringOrDash(item.Outcome),
				stringOrDash(formatHookDuration(item.DurationMS)),
				stringOrDash(item.Error),
			}
		},
		func(item HookRunRecord) []string {
			return []string{
				item.HookName,
				item.Event,
				item.Outcome,
				strconv.FormatInt(item.DurationMS, 10),
				item.Error,
				formatTime(item.RecordedAt),
			}
		},
	)
}

func hookMatcherRows(matcher hookspkg.HookMatcher) [][]string {
	rows := make([][]string, 0, 16)
	appendRow := func(label string, value string) {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			rows = append(rows, []string{label, trimmed})
		}
	}

	appendRow("agent_name", matcher.AgentName)
	appendRow("agent_type", matcher.AgentType)
	appendRow("workspace_id", matcher.WorkspaceID)
	appendRow("workspace_root", matcher.WorkspaceRoot)
	appendRow("session_type", matcher.SessionType)
	appendRow("input_class", matcher.InputClass)
	appendRow("acp_event_type", matcher.ACPEventType)
	appendRow("turn_id", matcher.TurnID)
	appendRow("tool_id", matcher.ToolID)
	appendRow("tool_name", matcher.ToolName)
	if matcher.ToolReadOnly != nil {
		rows = append(rows, []string{"tool_read_only", strconv.FormatBool(*matcher.ToolReadOnly)})
	}
	appendRow("decision_class", matcher.DecisionClass)
	appendRow("message_role", matcher.MessageRole)
	appendRow("message_delta_type", matcher.MessageDeltaType)
	if matcher.NetworkMatcher != nil {
		appendRow("channel", matcher.Channel)
		appendRow("surface", matcher.Surface)
		appendRow("kind", matcher.Kind)
		appendRow("direction", matcher.Direction)
		appendRow("work_state", matcher.WorkState)
	}
	if matcher.CompactionMatcher != nil {
		appendRow("compaction_reason", matcher.Reason)
		appendRow("compaction_strategy", matcher.Strategy)
	}
	return rows
}

func hookMetadataRows(metadata map[string]string) [][]string {
	if len(metadata) == 0 {
		return nil
	}

	keys := make([]string, 0, len(metadata))
	for key := range metadata {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	rows := make([][]string, 0, len(keys))
	for _, key := range keys {
		rows = append(rows, []string{key, metadata[key]})
	}
	return rows
}

func formatHookDuration(durationMS int64) string {
	if durationMS <= 0 {
		return ""
	}
	return fmt.Sprintf("%dms", durationMS)
}
