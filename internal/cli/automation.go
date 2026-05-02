package cli

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	automationpkg "github.com/pedronauck/agh/internal/automation"
	"github.com/spf13/cobra"
)

type automationTriggerCommandInput struct {
	Name               string
	ScopeRaw           string
	EventRaw           string
	WorkspaceRef       string
	AgentName          string
	Prompt             string
	RetryRaw           string
	FilterFlags        []string
	Enabled            bool
	WebhookID          string
	EndpointSlug       string
	WebhookSecretRef   string
	WebhookSecretValue string
}

type automationJobUpdateInput struct {
	Name         string
	AgentName    string
	WorkspaceRef string
	Prompt       string
	ScheduleRaw  string
	RetryRaw     string
	Enabled      bool
}

func newAutomationCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "automation",
		Short: "Manage automation jobs, triggers, and runs",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newAutomationJobsCommand(deps))
	cmd.AddCommand(newAutomationTriggersCommand(deps))
	cmd.AddCommand(newAutomationRunsCommand(deps))
	return cmd
}

func newAutomationJobsCommand(deps commandDeps) *cobra.Command {
	var (
		scopeRaw     string
		workspaceRef string
		sourceRaw    string
		last         int
	)

	cmd := &cobra.Command{
		Use:   "jobs",
		Short: "Manage automation jobs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			query, err := parseAutomationJobListQuery(
				cmd.Context(),
				client,
				scopeRaw,
				workspaceRef,
				sourceRaw,
				last,
			)
			if err != nil {
				return err
			}

			jobs, err := client.ListAutomationJobs(cmd.Context(), query)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, automationJobListBundle(jobs))
		},
	}
	cmd.Flags().StringVar(&scopeRaw, "scope", "", "Filter by scope: global or workspace")
	cmd.Flags().StringVar(&workspaceRef, "workspace", "", "Filter by workspace path, name, or ID")
	cmd.Flags().
		StringVar(&sourceRaw, "source", "", "Filter by definition source: config or dynamic")
	cmd.Flags().IntVar(&last, "last", 0, "Show only the most recent N jobs")

	cmd.AddCommand(newAutomationJobsCreateCommand(deps))
	cmd.AddCommand(newAutomationJobsGetCommand(deps))
	cmd.AddCommand(newAutomationJobsUpdateCommand(deps))
	cmd.AddCommand(newAutomationJobsDeleteCommand(deps))
	cmd.AddCommand(newAutomationJobsTriggerCommand(deps))
	cmd.AddCommand(newAutomationJobsHistoryCommand(deps))
	return cmd
}

func newAutomationJobsCreateCommand(deps commandDeps) *cobra.Command {
	var (
		name         string
		scopeRaw     string
		scheduleRaw  string
		agentName    string
		workspaceRef string
		prompt       string
		retryRaw     string
		enabled      bool
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an automation job",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			scope, workspaceID, err := resolveAutomationScopeWorkspace(
				cmd.Context(),
				client,
				scopeRaw,
				workspaceRef,
			)
			if err != nil {
				return err
			}
			schedule, err := parseAutomationScheduleFlag(scheduleRaw)
			if err != nil {
				return err
			}
			retry, err := parseAutomationRetryFlag(retryRaw)
			if err != nil {
				return err
			}

			request := AutomationJobCreateRequest{
				Scope:       scope,
				Name:        strings.TrimSpace(name),
				AgentName:   strings.TrimSpace(agentName),
				WorkspaceID: workspaceID,
				Prompt:      strings.TrimSpace(prompt),
				Schedule:    schedule,
			}
			if cmd.Flags().Changed("enabled") {
				request.Enabled = boolPointer(enabled)
			}
			if retry != nil {
				request.Retry = retry
			}

			created, err := client.CreateAutomationJob(cmd.Context(), request)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, automationJobBundle(created))
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Job name")
	cmd.Flags().StringVar(&scopeRaw, "scope", "", "Job scope: global or workspace")
	cmd.Flags().
		StringVar(&scheduleRaw, "schedule", "", "Schedule spec: <cron-expr>, every:<duration>, or at:<timestamp>")
	cmd.Flags().StringVar(&agentName, "agent", "", "Agent definition name")
	cmd.Flags().
		StringVar(&workspaceRef, "workspace", "", "Workspace path, name, or ID (required when --scope=workspace)")
	cmd.Flags().StringVar(&prompt, "prompt", "", "Prompt body to dispatch")
	cmd.Flags().
		StringVar(&retryRaw, "retry", "", `Retry policy: "none", "backoff", or "backoff:<max_retries>:<base_delay>"`)
	cmd.Flags().BoolVar(&enabled, "enabled", false, "Create the job enabled or disabled")
	mustMarkFlagRequired(cmd, "name")
	mustMarkFlagRequired(cmd, "scope")
	mustMarkFlagRequired(cmd, "schedule")
	mustMarkFlagRequired(cmd, "agent")
	mustMarkFlagRequired(cmd, "prompt")
	return cmd
}

func newAutomationJobsGetCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Show one automation job",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			job, err := client.GetAutomationJob(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, automationJobBundle(job))
		},
	}
}

func newAutomationJobsUpdateCommand(deps commandDeps) *cobra.Command {
	var (
		name         string
		agentName    string
		workspaceRef string
		prompt       string
		scheduleRaw  string
		retryRaw     string
		enabled      bool
	)

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update an automation job",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			request, err := buildAutomationJobUpdateRequest(cmd, client, automationJobUpdateInput{
				Name:         name,
				AgentName:    agentName,
				WorkspaceRef: workspaceRef,
				Prompt:       prompt,
				ScheduleRaw:  scheduleRaw,
				RetryRaw:     retryRaw,
				Enabled:      enabled,
			})
			if err != nil {
				return err
			}
			if !request.HasChanges() {
				return errors.New("cli: automation job update requires at least one change flag")
			}

			updated, err := client.UpdateAutomationJob(cmd.Context(), args[0], request)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, automationJobBundle(updated))
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Update the job name")
	cmd.Flags().StringVar(&agentName, "agent", "", "Update the agent definition")
	cmd.Flags().StringVar(&workspaceRef, "workspace", "", "Update the workspace path, name, or ID")
	cmd.Flags().StringVar(&prompt, "prompt", "", "Update the prompt body")
	cmd.Flags().StringVar(&scheduleRaw, "schedule", "", "Update the schedule spec")
	cmd.Flags().
		StringVar(&retryRaw, "retry", "", `Update retry policy: "none", "backoff", or "backoff:<max_retries>:<base_delay>"`)
	cmd.Flags().BoolVar(&enabled, "enabled", false, "Update the enabled state")
	return cmd
}

func buildAutomationJobUpdateRequest(
	cmd *cobra.Command,
	client DaemonClient,
	input automationJobUpdateInput,
) (AutomationJobUpdateRequest, error) {
	request := AutomationJobUpdateRequest{}
	if cmd.Flags().Changed("name") {
		request.Name = stringPointer(strings.TrimSpace(input.Name))
	}
	if cmd.Flags().Changed("agent") {
		request.AgentName = stringPointer(strings.TrimSpace(input.AgentName))
	}
	if cmd.Flags().Changed("workspace") {
		workspaceID, err := resolveAutomationWorkspaceID(cmd.Context(), client, input.WorkspaceRef)
		if err != nil {
			return AutomationJobUpdateRequest{}, err
		}
		request.WorkspaceID = stringPointer(workspaceID)
	}
	if cmd.Flags().Changed("prompt") {
		request.Prompt = stringPointer(strings.TrimSpace(input.Prompt))
	}
	if cmd.Flags().Changed("schedule") {
		schedule, err := parseAutomationScheduleFlag(input.ScheduleRaw)
		if err != nil {
			return AutomationJobUpdateRequest{}, err
		}
		request.Schedule = &schedule
	}
	if cmd.Flags().Changed("retry") {
		retry, err := parseAutomationRetryFlag(input.RetryRaw)
		if err != nil {
			return AutomationJobUpdateRequest{}, err
		}
		if retry == nil {
			return AutomationJobUpdateRequest{}, errors.New(
				`cli: --retry requires "none", "backoff", or "backoff:<max_retries>:<base_delay>"`,
			)
		}
		request.Retry = retry
	}
	if cmd.Flags().Changed("enabled") {
		request.Enabled = boolPointer(input.Enabled)
	}
	return request, nil
}

func newAutomationJobsDeleteCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete an automation job",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			current, err := client.GetAutomationJob(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if err := client.DeleteAutomationJob(cmd.Context(), args[0]); err != nil {
				return err
			}
			return writeCommandOutput(cmd, automationJobBundle(current))
		},
	}
}

func newAutomationJobsTriggerCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "trigger <id>",
		Short: "Force an immediate automation job run",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			run, err := client.TriggerAutomationJob(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, automationRunBundle(run))
		},
	}
}

func newAutomationJobsHistoryCommand(deps commandDeps) *cobra.Command {
	var (
		statusRaw string
		sinceRaw  string
		untilRaw  string
		last      int
	)

	cmd := &cobra.Command{
		Use:   "history <id>",
		Short: "Show run history for one automation job",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			query, err := parseAutomationRunListQuery(statusRaw, sinceRaw, untilRaw, last, deps.now)
			if err != nil {
				return err
			}

			runs, err := client.AutomationJobRuns(cmd.Context(), args[0], query)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, automationRunListBundle(runs))
		},
	}
	cmd.Flags().StringVar(&statusRaw, "status", "", "Filter by run status")
	cmd.Flags().
		StringVar(&sinceRaw, "since", "", "Show runs since an RFC3339 timestamp or relative duration")
	cmd.Flags().
		StringVar(&untilRaw, "until", "", "Show runs until an RFC3339 timestamp or relative duration")
	cmd.Flags().IntVar(&last, "last", 0, "Show only the most recent N runs")
	return cmd
}

func newAutomationTriggersCommand(deps commandDeps) *cobra.Command {
	var (
		scopeRaw     string
		workspaceRef string
		eventRaw     string
		sourceRaw    string
		last         int
	)

	cmd := &cobra.Command{
		Use:   "triggers",
		Short: "Manage automation triggers",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			query, err := parseAutomationTriggerListQuery(
				cmd.Context(),
				client,
				scopeRaw,
				workspaceRef,
				eventRaw,
				sourceRaw,
				last,
			)
			if err != nil {
				return err
			}

			triggers, err := client.ListAutomationTriggers(cmd.Context(), query)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, automationTriggerListBundle(triggers))
		},
	}
	cmd.Flags().StringVar(&scopeRaw, "scope", "", "Filter by scope: global or workspace")
	cmd.Flags().StringVar(&workspaceRef, "workspace", "", "Filter by workspace path, name, or ID")
	cmd.Flags().StringVar(&eventRaw, "event", "", "Filter by activation event")
	cmd.Flags().
		StringVar(&sourceRaw, "source", "", "Filter by definition source: config or dynamic")
	cmd.Flags().IntVar(&last, "last", 0, "Show only the most recent N triggers")

	cmd.AddCommand(newAutomationTriggersCreateCommand(deps))
	cmd.AddCommand(newAutomationTriggersGetCommand(deps))
	cmd.AddCommand(newAutomationTriggersUpdateCommand(deps))
	cmd.AddCommand(newAutomationTriggersDeleteCommand(deps))
	cmd.AddCommand(newAutomationTriggersHistoryCommand(deps))
	return cmd
}

func newAutomationTriggersCreateCommand(deps commandDeps) *cobra.Command {
	var (
		name               string
		scopeRaw           string
		eventRaw           string
		workspaceRef       string
		agentName          string
		prompt             string
		retryRaw           string
		filterFlags        []string
		enabled            bool
		webhookID          string
		endpointSlug       string
		webhookSecretRef   string
		webhookSecretValue string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an automation trigger",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			request, err := buildAutomationTriggerCreateRequest(
				cmd,
				client,
				automationTriggerCommandInput{
					Name:               name,
					ScopeRaw:           scopeRaw,
					EventRaw:           eventRaw,
					WorkspaceRef:       workspaceRef,
					AgentName:          agentName,
					Prompt:             prompt,
					RetryRaw:           retryRaw,
					FilterFlags:        filterFlags,
					Enabled:            enabled,
					WebhookID:          webhookID,
					EndpointSlug:       endpointSlug,
					WebhookSecretRef:   webhookSecretRef,
					WebhookSecretValue: webhookSecretValue,
				},
			)
			if err != nil {
				return err
			}

			created, err := client.CreateAutomationTrigger(cmd.Context(), request)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, automationTriggerBundle(created))
		},
	}
	bindAutomationTriggerCreateFlags(
		cmd,
		&name,
		&scopeRaw,
		&eventRaw,
		&workspaceRef,
		&agentName,
		&prompt,
		&retryRaw,
		&filterFlags,
		&enabled,
		&webhookID,
		&endpointSlug,
		&webhookSecretRef,
		&webhookSecretValue,
	)
	return cmd
}

func bindAutomationTriggerCreateFlags(
	cmd *cobra.Command,
	name *string,
	scopeRaw *string,
	eventRaw *string,
	workspaceRef *string,
	agentName *string,
	prompt *string,
	retryRaw *string,
	filterFlags *[]string,
	enabled *bool,
	webhookID *string,
	endpointSlug *string,
	webhookSecretRef *string,
	webhookSecretValue *string,
) {
	cmd.Flags().StringVar(name, "name", "", "Trigger name")
	cmd.Flags().StringVar(scopeRaw, "scope", "", "Trigger scope: global or workspace")
	cmd.Flags().StringVar(eventRaw, "event", "", "Trigger event name")
	cmd.Flags().
		StringVar(workspaceRef, "workspace", "", "Workspace path, name, or ID (required when --scope=workspace)")
	cmd.Flags().StringVar(agentName, "agent", "", "Agent definition name")
	cmd.Flags().StringVar(prompt, "prompt", "", "Prompt template body")
	cmd.Flags().
		StringArrayVar(filterFlags, "filter", nil, "Exact-match filter(s): key=value or comma-separated key=value pairs")
	cmd.Flags().
		StringVar(retryRaw, "retry", "", `Retry policy: "none", "backoff", or "backoff:<max_retries>:<base_delay>"`)
	cmd.Flags().BoolVar(enabled, "enabled", false, "Create the trigger enabled or disabled")
	cmd.Flags().StringVar(webhookID, "webhook-id", "", "Stable webhook identifier override for webhook events")
	cmd.Flags().StringVar(endpointSlug, "endpoint-slug", "", "Public endpoint slug for webhook events")
	cmd.Flags().StringVar(webhookSecretRef, "webhook-secret-ref", "", "Secret ref for webhook events")
	cmd.Flags().StringVar(webhookSecretValue, "webhook-secret-value", "", "Write-only webhook secret value")
	mustMarkFlagRequired(cmd, "name")
	mustMarkFlagRequired(cmd, "scope")
	mustMarkFlagRequired(cmd, "event")
	mustMarkFlagRequired(cmd, "agent")
	mustMarkFlagRequired(cmd, "prompt")
}

func newAutomationTriggersGetCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Show one automation trigger",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			trigger, err := client.GetAutomationTrigger(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, automationTriggerBundle(trigger))
		},
	}
}

func newAutomationTriggersUpdateCommand(deps commandDeps) *cobra.Command {
	var (
		name               string
		agentName          string
		workspaceRef       string
		prompt             string
		eventRaw           string
		filterFlags        []string
		retryRaw           string
		enabled            bool
		webhookID          string
		endpointSlug       string
		webhookSecretRef   string
		webhookSecretValue string
	)

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update an automation trigger",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			request, err := buildAutomationTriggerUpdateRequest(
				cmd,
				client,
				automationTriggerCommandInput{
					Name:               name,
					EventRaw:           eventRaw,
					WorkspaceRef:       workspaceRef,
					AgentName:          agentName,
					Prompt:             prompt,
					RetryRaw:           retryRaw,
					FilterFlags:        filterFlags,
					Enabled:            enabled,
					WebhookID:          webhookID,
					EndpointSlug:       endpointSlug,
					WebhookSecretRef:   webhookSecretRef,
					WebhookSecretValue: webhookSecretValue,
				},
			)
			if err != nil {
				return err
			}

			updated, err := client.UpdateAutomationTrigger(cmd.Context(), args[0], request)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, automationTriggerBundle(updated))
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Update the trigger name")
	cmd.Flags().StringVar(&agentName, "agent", "", "Update the agent definition")
	cmd.Flags().StringVar(&workspaceRef, "workspace", "", "Update the workspace path, name, or ID")
	cmd.Flags().StringVar(&prompt, "prompt", "", "Update the prompt template body")
	cmd.Flags().StringVar(&eventRaw, "event", "", "Update the trigger event")
	cmd.Flags().
		StringArrayVar(&filterFlags, "filter", nil, "Replace filters with key=value entries")
	cmd.Flags().
		StringVar(&retryRaw, "retry", "", `Update retry policy: "none", "backoff", or "backoff:<max_retries>:<base_delay>"`)
	cmd.Flags().BoolVar(&enabled, "enabled", false, "Update the enabled state")
	cmd.Flags().StringVar(&webhookID, "webhook-id", "", "Update the stable webhook identifier")
	cmd.Flags().StringVar(&endpointSlug, "endpoint-slug", "", "Update the webhook endpoint slug")
	cmd.Flags().
		StringVar(&webhookSecretRef, "webhook-secret-ref", "", "Update the webhook secret ref")
	cmd.Flags().
		StringVar(&webhookSecretValue, "webhook-secret-value", "", "Write-only webhook secret value")
	return cmd
}

func newAutomationTriggersDeleteCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete an automation trigger",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			current, err := client.GetAutomationTrigger(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if err := client.DeleteAutomationTrigger(cmd.Context(), args[0]); err != nil {
				return err
			}
			return writeCommandOutput(cmd, automationTriggerBundle(current))
		},
	}
}

func newAutomationTriggersHistoryCommand(deps commandDeps) *cobra.Command {
	var (
		statusRaw string
		sinceRaw  string
		untilRaw  string
		last      int
	)

	cmd := &cobra.Command{
		Use:   "history <id>",
		Short: "Show run history for one automation trigger",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			query, err := parseAutomationRunListQuery(statusRaw, sinceRaw, untilRaw, last, deps.now)
			if err != nil {
				return err
			}

			runs, err := client.AutomationTriggerRuns(cmd.Context(), args[0], query)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, automationRunListBundle(runs))
		},
	}
	cmd.Flags().StringVar(&statusRaw, "status", "", "Filter by run status")
	cmd.Flags().
		StringVar(&sinceRaw, "since", "", "Show runs since an RFC3339 timestamp or relative duration")
	cmd.Flags().
		StringVar(&untilRaw, "until", "", "Show runs until an RFC3339 timestamp or relative duration")
	cmd.Flags().IntVar(&last, "last", 0, "Show only the most recent N runs")
	return cmd
}

func newAutomationRunsCommand(deps commandDeps) *cobra.Command {
	var (
		jobID     string
		triggerID string
		statusRaw string
		sinceRaw  string
		untilRaw  string
		last      int
	)

	cmd := &cobra.Command{
		Use:   "runs",
		Short: "Inspect automation run history",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			query, err := parseAutomationRunListQuery(statusRaw, sinceRaw, untilRaw, last, deps.now)
			if err != nil {
				return err
			}
			query.JobID = strings.TrimSpace(jobID)
			query.TriggerID = strings.TrimSpace(triggerID)

			runs, err := client.ListAutomationRuns(cmd.Context(), query)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, automationRunListBundle(runs))
		},
	}
	cmd.Flags().StringVar(&jobID, "job-id", "", "Filter by automation job ID")
	cmd.Flags().StringVar(&triggerID, "trigger-id", "", "Filter by automation trigger ID")
	cmd.Flags().StringVar(&statusRaw, "status", "", "Filter by run status")
	cmd.Flags().
		StringVar(&sinceRaw, "since", "", "Show runs since an RFC3339 timestamp or relative duration")
	cmd.Flags().
		StringVar(&untilRaw, "until", "", "Show runs until an RFC3339 timestamp or relative duration")
	cmd.Flags().IntVar(&last, "last", 0, "Show only the most recent N runs")
	cmd.AddCommand(newAutomationRunsGetCommand(deps))
	return cmd
}

func newAutomationRunsGetCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Show one automation run",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			run, err := client.GetAutomationRun(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, automationRunBundle(run))
		},
	}
}

func parseAutomationJobListQuery(
	ctx context.Context,
	client DaemonClient,
	scopeRaw string,
	workspaceRef string,
	sourceRaw string,
	last int,
) (AutomationJobQuery, error) {
	query := AutomationJobQuery{}
	if err := validateAutomationLast(last); err != nil {
		return AutomationJobQuery{}, err
	}
	query.Limit = last

	scope, err := parseOptionalAutomationScope(scopeRaw)
	if err != nil {
		return AutomationJobQuery{}, err
	}
	query.Scope = scope

	if trimmed := strings.TrimSpace(workspaceRef); trimmed != "" {
		workspaceID, err := resolveAutomationWorkspaceID(ctx, client, trimmed)
		if err != nil {
			return AutomationJobQuery{}, err
		}
		query.WorkspaceID = workspaceID
	}

	source, err := parseOptionalAutomationSource(sourceRaw)
	if err != nil {
		return AutomationJobQuery{}, err
	}
	query.Source = source
	return query, nil
}

func parseAutomationTriggerListQuery(
	ctx context.Context,
	client DaemonClient,
	scopeRaw string,
	workspaceRef string,
	eventRaw string,
	sourceRaw string,
	last int,
) (AutomationTriggerQuery, error) {
	query := AutomationTriggerQuery{
		Event: strings.TrimSpace(eventRaw),
	}
	if err := validateAutomationLast(last); err != nil {
		return AutomationTriggerQuery{}, err
	}
	query.Limit = last

	scope, err := parseOptionalAutomationScope(scopeRaw)
	if err != nil {
		return AutomationTriggerQuery{}, err
	}
	query.Scope = scope

	if trimmed := strings.TrimSpace(workspaceRef); trimmed != "" {
		workspaceID, err := resolveAutomationWorkspaceID(ctx, client, trimmed)
		if err != nil {
			return AutomationTriggerQuery{}, err
		}
		query.WorkspaceID = workspaceID
	}

	source, err := parseOptionalAutomationSource(sourceRaw)
	if err != nil {
		return AutomationTriggerQuery{}, err
	}
	query.Source = source
	return query, nil
}

func parseAutomationRunListQuery(
	statusRaw string,
	sinceRaw string,
	untilRaw string,
	last int,
	now func() time.Time,
) (AutomationRunQuery, error) {
	query := AutomationRunQuery{}
	if err := validateAutomationLast(last); err != nil {
		return AutomationRunQuery{}, err
	}
	query.Limit = last

	status, err := parseOptionalAutomationRunStatus(statusRaw)
	if err != nil {
		return AutomationRunQuery{}, err
	}
	query.Status = status

	since, err := parseAutomationOptionalTimeFlag(sinceRaw, "since", now)
	if err != nil {
		return AutomationRunQuery{}, err
	}
	query.Since = since

	until, err := parseAutomationOptionalTimeFlag(untilRaw, "until", now)
	if err != nil {
		return AutomationRunQuery{}, err
	}
	query.Until = until
	return query, nil
}

func resolveAutomationScopeWorkspace(
	ctx context.Context,
	client DaemonClient,
	rawScope string,
	workspaceRef string,
) (automationpkg.Scope, string, error) {
	scope, err := parseRequiredAutomationScope(rawScope)
	if err != nil {
		return "", "", err
	}

	trimmedWorkspace := strings.TrimSpace(workspaceRef)
	switch scope {
	case automationpkg.AutomationScopeGlobal:
		if trimmedWorkspace != "" {
			return "", "", errors.New("cli: --workspace must be empty when --scope is global")
		}
		return scope, "", nil
	case automationpkg.AutomationScopeWorkspace:
		if trimmedWorkspace == "" {
			return "", "", errors.New("cli: --workspace is required when --scope is workspace")
		}
		workspaceID, err := resolveAutomationWorkspaceID(ctx, client, trimmedWorkspace)
		if err != nil {
			return "", "", err
		}
		return scope, workspaceID, nil
	default:
		return "", "", fmt.Errorf("cli: unsupported automation scope %q", scope)
	}
}

func resolveAutomationWorkspaceID(
	ctx context.Context,
	client DaemonClient,
	ref string,
) (string, error) {
	trimmed := strings.TrimSpace(ref)
	if trimmed == "" {
		return "", nil
	}
	detail, err := client.GetWorkspace(ctx, trimmed)
	if err != nil {
		return "", fmt.Errorf("cli: resolve workspace %q: %w", trimmed, err)
	}
	id := strings.TrimSpace(detail.Workspace.ID)
	if id == "" {
		return "", fmt.Errorf("cli: resolve workspace %q: missing workspace id", trimmed)
	}
	return id, nil
}

func parseRequiredAutomationScope(raw string) (automationpkg.Scope, error) {
	if strings.TrimSpace(raw) == "" {
		return "", errors.New("cli: --scope is required")
	}
	return parseOptionalAutomationScope(raw)
}

func parseOptionalAutomationScope(raw string) (automationpkg.Scope, error) {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	if trimmed == "" {
		return "", nil
	}
	scope := automationpkg.Scope(trimmed)
	if err := scope.Validate("scope"); err != nil {
		return "", fmt.Errorf("cli: %w", err)
	}
	return scope, nil
}

func parseOptionalAutomationSource(raw string) (automationpkg.JobSource, error) {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	if trimmed == "" {
		return "", nil
	}
	source := automationpkg.JobSource(trimmed)
	if err := source.Validate("source"); err != nil {
		return "", fmt.Errorf("cli: %w", err)
	}
	return source, nil
}

func parseOptionalAutomationRunStatus(raw string) (automationpkg.RunStatus, error) {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	if trimmed == "" {
		return "", nil
	}
	status := automationpkg.RunStatus(trimmed)
	if err := status.Validate("status"); err != nil {
		return "", fmt.Errorf("cli: %w", err)
	}
	return status, nil
}

func parseAutomationScheduleFlag(raw string) (automationpkg.ScheduleSpec, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return automationpkg.ScheduleSpec{}, errors.New("cli: --schedule is required")
	}

	lower := strings.ToLower(value)
	spec := automationpkg.ScheduleSpec{}
	switch {
	case strings.HasPrefix(lower, "every:"):
		spec.Mode = automationpkg.ScheduleModeEvery
		spec.Interval = strings.TrimSpace(value[len("every:"):])
	case strings.HasPrefix(lower, "at:"):
		timestamp, err := normalizeAutomationAtTime(strings.TrimSpace(value[len("at:"):]))
		if err != nil {
			return automationpkg.ScheduleSpec{}, err
		}
		spec.Mode = automationpkg.ScheduleModeAt
		spec.Time = timestamp
	default:
		spec.Mode = automationpkg.ScheduleModeCron
		spec.Expr = value
	}

	if err := spec.Validate("schedule"); err != nil {
		return automationpkg.ScheduleSpec{}, fmt.Errorf("cli: %w", err)
	}
	return spec, nil
}

func normalizeAutomationAtTime(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", errors.New("cli: at-schedule timestamp is required")
	}

	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
	} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed.UTC().Format(time.RFC3339), nil
		}
	}

	return "", fmt.Errorf(
		"cli: invalid at-schedule timestamp %q: use RFC3339 or YYYY-MM-DDTHH:MM",
		value,
	)
}

func parseAutomationRetryFlag(raw string) (*automationpkg.RetryConfig, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}

	lower := strings.ToLower(trimmed)
	var cfg automationpkg.RetryConfig
	switch {
	case lower == string(automationpkg.RetryStrategyNone):
		cfg = automationpkg.DefaultRetryConfig()
	case lower == string(automationpkg.RetryStrategyBackoff):
		cfg = automationpkg.DefaultBackoffRetryConfig()
	case strings.HasPrefix(lower, "backoff:"):
		payload := strings.TrimSpace(trimmed[len("backoff:"):])
		parts := strings.Split(payload, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf(
				`cli: invalid retry value %q: use "none", "backoff", or "backoff:<max_retries>:<base_delay>"`,
				trimmed,
			)
		}
		maxRetries, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return nil, fmt.Errorf("cli: invalid retry max_retries %q: %w", parts[0], err)
		}
		cfg = automationpkg.RetryConfig{
			Strategy:   automationpkg.RetryStrategyBackoff,
			MaxRetries: maxRetries,
			BaseDelay:  strings.TrimSpace(parts[1]),
		}
	default:
		return nil, fmt.Errorf(
			`cli: invalid retry value %q: use "none", "backoff", or "backoff:<max_retries>:<base_delay>"`,
			trimmed,
		)
	}

	if err := cfg.Validate("retry"); err != nil {
		return nil, fmt.Errorf("cli: %w", err)
	}
	return &cfg, nil
}

func parseAutomationFilterFlags(flags []string) (map[string]string, error) {
	if len(flags) == 0 {
		return nil, nil
	}

	filter := make(map[string]string)
	for _, rawFlag := range flags {
		for entry := range strings.SplitSeq(rawFlag, ",") {
			trimmedEntry := strings.TrimSpace(entry)
			if trimmedEntry == "" {
				continue
			}
			key, value, ok := strings.Cut(trimmedEntry, "=")
			if !ok {
				return nil, fmt.Errorf("cli: invalid filter %q: use key=value", trimmedEntry)
			}
			key = strings.TrimSpace(key)
			value = strings.TrimSpace(value)
			if key == "" {
				return nil, fmt.Errorf("cli: invalid filter %q: key is required", trimmedEntry)
			}
			if value == "" {
				return nil, fmt.Errorf("cli: invalid filter %q: value is required", trimmedEntry)
			}
			filter[key] = value
		}
	}
	if len(filter) == 0 {
		return nil, nil
	}
	if err := automationpkg.ValidateTriggerFilter(filter, "filter"); err != nil {
		return nil, fmt.Errorf("cli: %w", err)
	}
	return filter, nil
}

func parseAutomationOptionalTimeFlag(
	raw string,
	flagName string,
	now func() time.Time,
) (time.Time, error) {
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
		return time.Time{}, fmt.Errorf("cli: invalid --%s value %q", flagName, value)
	}
	if duration < 0 {
		return time.Time{}, fmt.Errorf("cli: relative --%s must be positive: %q", flagName, value)
	}

	if now != nil {
		return now().UTC().Add(-duration), nil
	}
	return time.Now().UTC().Add(-duration), nil
}

func validateAutomationLast(last int) error {
	if last < 0 {
		return fmt.Errorf("cli: --last must be zero or positive: %d", last)
	}
	return nil
}

func automationJobBundle(item JobRecord) outputBundle {
	return outputBundle{
		jsonValue: item,
		human: func() (string, error) {
			return renderHumanBlocks(
				renderHumanSection("Automation Job", []keyValue{
					{Label: "ID", Value: stringOrDash(item.ID)},
					{Label: "Name", Value: stringOrDash(item.Name)},
					{Label: "Scope", Value: stringOrDash(string(item.Scope))},
					{Label: "Workspace", Value: stringOrDash(item.WorkspaceID)},
					{Label: "Agent", Value: stringOrDash(item.AgentName)},
					{Label: "Enabled", Value: strconv.FormatBool(item.Enabled)},
					{Label: "Source", Value: stringOrDash(string(item.Source))},
					{
						Label: "Schedule",
						Value: stringOrDash(formatAutomationSchedule(item.Schedule)),
					},
					{Label: "Retry", Value: stringOrDash(formatAutomationRetry(item.Retry))},
					{
						Label: "Fire Limit",
						Value: stringOrDash(formatAutomationFireLimit(item.FireLimit)),
					},
					{Label: "Next Run", Value: stringOrDash(formatOptionalTime(item.NextRun))},
					{
						Label: "Last Scheduled",
						Value: stringOrDash(formatOptionalTime(automationJobLastScheduledAt(item))),
					},
					{Label: "Last Fire ID", Value: stringOrDash(automationJobLastFireID(item))},
					{Label: "Catch-up Policy", Value: stringOrDash(automationJobCatchUpPolicy(item))},
					{Label: "Misfires", Value: strconv.Itoa(automationJobMisfireCount(item))},
					{Label: "Created", Value: stringOrDash(formatTime(item.CreatedAt))},
					{Label: "Updated", Value: stringOrDash(formatTime(item.UpdatedAt))},
				}),
				renderHumanSection(
					"Prompt",
					[]keyValue{{Label: "Body", Value: stringOrDash(item.Prompt)}},
				),
			), nil
		},
		toon: func() (string, error) {
			return renderToonObject("automation_job", []string{
				"id",
				"name",
				"scope",
				"workspace_id",
				"agent_name",
				"enabled",
				"source",
				"schedule",
				"retry",
				"fire_limit",
				"next_run",
				"last_scheduled_at",
				"last_fire_id",
				"catch_up_policy",
				"misfire_count",
				"created_at",
				"updated_at",
				"prompt",
			}, []string{
				item.ID,
				item.Name,
				string(item.Scope),
				item.WorkspaceID,
				item.AgentName,
				strconv.FormatBool(item.Enabled),
				string(item.Source),
				formatAutomationSchedule(item.Schedule),
				formatAutomationRetry(item.Retry),
				formatAutomationFireLimit(item.FireLimit),
				formatOptionalTime(item.NextRun),
				formatOptionalTime(automationJobLastScheduledAt(item)),
				automationJobLastFireID(item),
				automationJobCatchUpPolicy(item),
				strconv.Itoa(automationJobMisfireCount(item)),
				formatTime(item.CreatedAt),
				formatTime(item.UpdatedAt),
				item.Prompt,
			}), nil
		},
	}
}

func automationJobListBundle(items []JobRecord) outputBundle {
	return listBundle(
		items,
		items,
		"Automation Jobs",
		[]string{
			"ID",
			"Name",
			"Scope",
			"Workspace",
			"Schedule",
			"Agent",
			"Enabled",
			"Source",
			"Next Run",
		},
		"automation_jobs",
		[]string{
			"id",
			"name",
			"scope",
			"workspace_id",
			"schedule",
			"agent_name",
			"enabled",
			"source",
			"next_run",
		},
		func(item JobRecord) []string {
			return []string{
				stringOrDash(item.ID),
				stringOrDash(item.Name),
				stringOrDash(string(item.Scope)),
				stringOrDash(item.WorkspaceID),
				stringOrDash(formatAutomationSchedule(item.Schedule)),
				stringOrDash(item.AgentName),
				strconv.FormatBool(item.Enabled),
				stringOrDash(string(item.Source)),
				stringOrDash(formatOptionalTime(item.NextRun)),
			}
		},
		func(item JobRecord) []string {
			return []string{
				item.ID,
				item.Name,
				string(item.Scope),
				item.WorkspaceID,
				formatAutomationSchedule(item.Schedule),
				item.AgentName,
				strconv.FormatBool(item.Enabled),
				string(item.Source),
				formatOptionalTime(item.NextRun),
			}
		},
	)
}

func automationTriggerBundle(item TriggerRecord) outputBundle {
	return outputBundle{
		jsonValue: item,
		human: func() (string, error) {
			return renderHumanBlocks(
				renderHumanSection("Automation Trigger", []keyValue{
					{Label: "ID", Value: stringOrDash(item.ID)},
					{Label: "Name", Value: stringOrDash(item.Name)},
					{Label: "Scope", Value: stringOrDash(string(item.Scope))},
					{Label: "Workspace", Value: stringOrDash(item.WorkspaceID)},
					{Label: "Agent", Value: stringOrDash(item.AgentName)},
					{Label: "Event", Value: stringOrDash(item.Event)},
					{Label: "Enabled", Value: strconv.FormatBool(item.Enabled)},
					{Label: "Source", Value: stringOrDash(string(item.Source))},
					{Label: "Retry", Value: stringOrDash(formatAutomationRetry(item.Retry))},
					{
						Label: "Fire Limit",
						Value: stringOrDash(formatAutomationFireLimit(item.FireLimit)),
					},
					{Label: "Webhook ID", Value: stringOrDash(item.WebhookID)},
					{Label: "Endpoint Slug", Value: stringOrDash(item.EndpointSlug)},
					{Label: "Webhook Path", Value: stringOrDash(displayTriggerEndpoint(item))},
					{Label: "Created", Value: stringOrDash(formatTime(item.CreatedAt))},
					{Label: "Updated", Value: stringOrDash(formatTime(item.UpdatedAt))},
				}),
				renderHumanSection(
					"Prompt",
					[]keyValue{{Label: "Body", Value: stringOrDash(item.Prompt)}},
				),
				renderHumanTable(
					"Filters",
					[]string{"Path", "Value"},
					automationFilterRows(item.Filter),
				),
			), nil
		},
		toon: func() (string, error) {
			return renderHumanBlocks(
				renderToonObject("automation_trigger", []string{
					"id",
					"name",
					"scope",
					"workspace_id",
					"agent_name",
					"event",
					"enabled",
					"source",
					"retry",
					"fire_limit",
					"webhook_id",
					"endpoint_slug",
					"webhook_path",
					"created_at",
					"updated_at",
					"prompt",
				}, []string{
					item.ID,
					item.Name,
					string(item.Scope),
					item.WorkspaceID,
					item.AgentName,
					item.Event,
					strconv.FormatBool(item.Enabled),
					string(item.Source),
					formatAutomationRetry(item.Retry),
					formatAutomationFireLimit(item.FireLimit),
					item.WebhookID,
					item.EndpointSlug,
					displayTriggerEndpoint(item),
					formatTime(item.CreatedAt),
					formatTime(item.UpdatedAt),
					item.Prompt,
				}),
				renderToonArray(
					"filters",
					[]string{"path", "value"},
					automationFilterRows(item.Filter),
				),
			), nil
		},
	}
}

func automationTriggerListBundle(items []TriggerRecord) outputBundle {
	return listBundle(
		items,
		items,
		"Automation Triggers",
		[]string{"ID", "Name", "Event", "Scope", "Workspace", "Agent", "Enabled", "Source"},
		"automation_triggers",
		[]string{"id", "name", "event", "scope", "workspace_id", "agent_name", "enabled", "source"},
		func(item TriggerRecord) []string {
			return []string{
				stringOrDash(item.ID),
				stringOrDash(item.Name),
				stringOrDash(item.Event),
				stringOrDash(string(item.Scope)),
				stringOrDash(item.WorkspaceID),
				stringOrDash(item.AgentName),
				strconv.FormatBool(item.Enabled),
				stringOrDash(string(item.Source)),
			}
		},
		func(item TriggerRecord) []string {
			return []string{
				item.ID,
				item.Name,
				item.Event,
				string(item.Scope),
				item.WorkspaceID,
				item.AgentName,
				strconv.FormatBool(item.Enabled),
				string(item.Source),
			}
		},
	)
}

func automationRunBundle(item RunRecord) outputBundle {
	return outputBundle{
		jsonValue: item,
		human: func() (string, error) {
			return renderHumanSection("Automation Run", []keyValue{
				{Label: "ID", Value: stringOrDash(item.ID)},
				{Label: "Target", Value: stringOrDash(displayRunTarget(item))},
				{Label: "Job ID", Value: stringOrDash(item.JobID)},
				{Label: "Trigger ID", Value: stringOrDash(item.TriggerID)},
				{Label: "Session ID", Value: stringOrDash(item.SessionID)},
				{Label: "Fire ID", Value: stringOrDash(item.FireID)},
				{Label: "Status", Value: stringOrDash(string(item.Status))},
				{Label: "Attempt", Value: strconv.Itoa(item.Attempt)},
				{Label: "Scheduled", Value: stringOrDash(formatOptionalTime(item.ScheduledAt))},
				{Label: "Started", Value: stringOrDash(formatOptionalTime(item.StartedAt))},
				{Label: "Ended", Value: stringOrDash(formatOptionalTime(item.EndedAt))},
				{Label: "Error", Value: stringOrDash(item.Error)},
				{Label: "Delivery Error", Value: stringOrDash(item.DeliveryError)},
			}), nil
		},
		toon: func() (string, error) {
			return renderToonObject("automation_run", []string{
				"id",
				"target",
				"job_id",
				"trigger_id",
				"session_id",
				"fire_id",
				"status",
				"attempt",
				"scheduled_at",
				"started_at",
				"ended_at",
				"error",
				"delivery_error",
			}, []string{
				item.ID,
				displayRunTarget(item),
				item.JobID,
				item.TriggerID,
				item.SessionID,
				item.FireID,
				string(item.Status),
				strconv.Itoa(item.Attempt),
				formatOptionalTime(item.ScheduledAt),
				formatOptionalTime(item.StartedAt),
				formatOptionalTime(item.EndedAt),
				item.Error,
				item.DeliveryError,
			}), nil
		},
	}
}

func automationRunListBundle(items []RunRecord) outputBundle {
	return listBundle(
		items,
		items,
		"Automation Runs",
		[]string{
			"ID",
			"Target",
			"Status",
			"Attempt",
			"Session",
			"Scheduled",
			"Started",
			"Ended",
			"Error",
			"Delivery Error",
		},
		"automation_runs",
		[]string{
			"id",
			"target",
			"status",
			"attempt",
			"session_id",
			"scheduled_at",
			"started_at",
			"ended_at",
			"error",
			"delivery_error",
		},
		func(item RunRecord) []string {
			return []string{
				stringOrDash(item.ID),
				stringOrDash(displayRunTarget(item)),
				stringOrDash(string(item.Status)),
				strconv.Itoa(item.Attempt),
				stringOrDash(item.SessionID),
				stringOrDash(formatOptionalTime(item.ScheduledAt)),
				stringOrDash(formatOptionalTime(item.StartedAt)),
				stringOrDash(formatOptionalTime(item.EndedAt)),
				stringOrDash(item.Error),
				stringOrDash(item.DeliveryError),
			}
		},
		func(item RunRecord) []string {
			return []string{
				item.ID,
				displayRunTarget(item),
				string(item.Status),
				strconv.Itoa(item.Attempt),
				item.SessionID,
				formatOptionalTime(item.ScheduledAt),
				formatOptionalTime(item.StartedAt),
				formatOptionalTime(item.EndedAt),
				item.Error,
				item.DeliveryError,
			}
		},
	)
}

func automationJobLastScheduledAt(item JobRecord) *time.Time {
	if item.Scheduler == nil {
		return nil
	}
	return item.Scheduler.LastScheduledAt
}

func automationJobLastFireID(item JobRecord) string {
	if item.Scheduler == nil {
		return ""
	}
	return item.Scheduler.LastFireID
}

func automationJobCatchUpPolicy(item JobRecord) string {
	if item.Scheduler == nil {
		return ""
	}
	return string(item.Scheduler.CatchUpPolicy)
}

func automationJobMisfireCount(item JobRecord) int {
	if item.Scheduler == nil {
		return 0
	}
	return item.Scheduler.MisfireCount
}

func automationFilterRows(filter map[string]string) [][]string {
	if len(filter) == 0 {
		return nil
	}
	keys := make([]string, 0, len(filter))
	for key := range filter {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	rows := make([][]string, 0, len(keys))
	for _, key := range keys {
		rows = append(rows, []string{key, filter[key]})
	}
	return rows
}

func buildAutomationTriggerCreateRequest(
	cmd *cobra.Command,
	client DaemonClient,
	input automationTriggerCommandInput,
) (AutomationTriggerCreateRequest, error) {
	scope, workspaceID, err := resolveAutomationScopeWorkspace(
		cmd.Context(),
		client,
		input.ScopeRaw,
		input.WorkspaceRef,
	)
	if err != nil {
		return AutomationTriggerCreateRequest{}, err
	}
	retry, err := parseAutomationRetryFlag(input.RetryRaw)
	if err != nil {
		return AutomationTriggerCreateRequest{}, err
	}
	filter, err := parseAutomationFilterFlags(input.FilterFlags)
	if err != nil {
		return AutomationTriggerCreateRequest{}, err
	}

	request := AutomationTriggerCreateRequest{
		Scope:              scope,
		Name:               strings.TrimSpace(input.Name),
		AgentName:          strings.TrimSpace(input.AgentName),
		WorkspaceID:        workspaceID,
		Prompt:             strings.TrimSpace(input.Prompt),
		Event:              strings.TrimSpace(input.EventRaw),
		Filter:             filter,
		WebhookID:          strings.TrimSpace(input.WebhookID),
		EndpointSlug:       strings.TrimSpace(input.EndpointSlug),
		WebhookSecretRef:   strings.TrimSpace(input.WebhookSecretRef),
		WebhookSecretValue: strings.TrimSpace(input.WebhookSecretValue),
	}
	if cmd.Flags().Changed("enabled") {
		request.Enabled = boolPointer(input.Enabled)
	}
	if retry != nil {
		request.Retry = retry
	}
	return request, nil
}

func buildAutomationTriggerUpdateRequest(
	cmd *cobra.Command,
	client DaemonClient,
	input automationTriggerCommandInput,
) (AutomationTriggerUpdateRequest, error) {
	request := AutomationTriggerUpdateRequest{}
	if cmd.Flags().Changed("name") {
		request.Name = stringPointer(strings.TrimSpace(input.Name))
	}
	if cmd.Flags().Changed("agent") {
		request.AgentName = stringPointer(strings.TrimSpace(input.AgentName))
	}
	if cmd.Flags().Changed("workspace") {
		workspaceID, err := resolveAutomationWorkspaceID(cmd.Context(), client, input.WorkspaceRef)
		if err != nil {
			return AutomationTriggerUpdateRequest{}, err
		}
		request.WorkspaceID = stringPointer(workspaceID)
	}
	if cmd.Flags().Changed("prompt") {
		request.Prompt = stringPointer(strings.TrimSpace(input.Prompt))
	}
	if cmd.Flags().Changed("event") {
		request.Event = stringPointer(strings.TrimSpace(input.EventRaw))
	}
	if cmd.Flags().Changed("filter") {
		filter, err := parseAutomationFilterFlags(input.FilterFlags)
		if err != nil {
			return AutomationTriggerUpdateRequest{}, err
		}
		request.Filter = filter
	}
	if cmd.Flags().Changed("retry") {
		retry, err := parseAutomationRetryFlag(input.RetryRaw)
		if err != nil {
			return AutomationTriggerUpdateRequest{}, err
		}
		if retry == nil {
			return AutomationTriggerUpdateRequest{}, errors.New(
				`cli: --retry requires "none", "backoff", or "backoff:<max_retries>:<base_delay>"`,
			)
		}
		request.Retry = retry
	}
	if cmd.Flags().Changed("enabled") {
		request.Enabled = boolPointer(input.Enabled)
	}
	if cmd.Flags().Changed("webhook-id") {
		request.WebhookID = stringPointer(strings.TrimSpace(input.WebhookID))
	}
	if cmd.Flags().Changed("endpoint-slug") {
		request.EndpointSlug = stringPointer(strings.TrimSpace(input.EndpointSlug))
	}
	if cmd.Flags().Changed("webhook-secret-ref") {
		request.WebhookSecretRef = stringPointer(strings.TrimSpace(input.WebhookSecretRef))
	}
	if cmd.Flags().Changed("webhook-secret-value") {
		request.WebhookSecretValue = stringPointer(strings.TrimSpace(input.WebhookSecretValue))
	}
	if !request.HasChanges() {
		return AutomationTriggerUpdateRequest{}, errors.New(
			"cli: automation trigger update requires at least one change flag",
		)
	}
	return request, nil
}

func formatAutomationSchedule(spec *automationpkg.ScheduleSpec) string {
	if spec == nil {
		return ""
	}
	switch spec.Mode {
	case automationpkg.ScheduleModeEvery:
		return "every:" + strings.TrimSpace(spec.Interval)
	case automationpkg.ScheduleModeAt:
		return "at:" + strings.TrimSpace(spec.Time)
	default:
		return "cron:" + strings.TrimSpace(spec.Expr)
	}
}

func formatAutomationRetry(cfg automationpkg.RetryConfig) string {
	switch cfg.Strategy {
	case automationpkg.RetryStrategyBackoff:
		return fmt.Sprintf("backoff:%d:%s", cfg.MaxRetries, strings.TrimSpace(cfg.BaseDelay))
	default:
		return "none"
	}
}

func formatAutomationFireLimit(cfg automationpkg.FireLimitConfig) string {
	return fmt.Sprintf("%d/%s", cfg.Max, strings.TrimSpace(cfg.Window))
}

func displayTriggerEndpoint(item TriggerRecord) string {
	if !strings.EqualFold(strings.TrimSpace(item.Event), "webhook") {
		return ""
	}
	endpoint, err := automationpkg.FormatWebhookEndpoint(item.EndpointSlug, item.WebhookID)
	if err != nil {
		return ""
	}
	if item.Scope == automationpkg.AutomationScopeWorkspace &&
		strings.TrimSpace(item.WorkspaceID) != "" {
		return "/api/webhooks/workspaces/" + strings.TrimSpace(item.WorkspaceID) + "/" + endpoint
	}
	return "/api/webhooks/global/" + endpoint
}

func displayRunTarget(item RunRecord) string {
	switch {
	case strings.TrimSpace(item.JobID) != "":
		return "job:" + strings.TrimSpace(item.JobID)
	case strings.TrimSpace(item.TriggerID) != "":
		return "trigger:" + strings.TrimSpace(item.TriggerID)
	default:
		return ""
	}
}

func formatOptionalTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return formatTime(*value)
}

func boolPointer(value bool) *bool {
	return &value
}

func stringPointer(value string) *string {
	return &value
}
