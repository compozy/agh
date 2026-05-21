package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/api/contract"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	diagnosticcontract "github.com/pedronauck/agh/internal/diagnosticcontract"
	diagnosticitems "github.com/pedronauck/agh/internal/diagnostics"
	"github.com/pedronauck/agh/internal/network"
	taskpkg "github.com/pedronauck/agh/internal/task"
	"github.com/spf13/cobra"
)

const (
	taskTypeValue = "Type"
)

const (
	taskAttemptValue             = "Attempt"
	taskBridgeValue              = "Bridge"
	taskChannelValue             = "Channel"
	taskClaimedByValue           = "Claimed By"
	taskCoordinationChannelValue = "Coordination Channel"
	taskCreatedValue             = "Created"
	taskCreatedByValue           = "Created By"
	taskDescriptionValue         = "Description"
	taskEndedValue               = "Ended"
	taskErrorValue               = "Error"
	taskIdentifierValue          = "Identifier"
	taskKindValue                = "Kind"
	taskModeValue                = "Mode"
	taskOriginValue              = "Origin"
	taskOutcomeValue             = "Outcome"
	taskOwnerValue               = "Owner"
	taskParentValue              = "Parent"
	taskQueuedValue              = "Queued"
	taskReasonValue              = "Reason"
	taskResultValue              = "Result"
	taskReviewValue              = "Review"
	taskRunValue                 = "Run"
	taskSandboxValue             = "Sandbox"
	taskScopeValue               = "Scope"
	taskSessionValue             = "Session"
	taskStartedValue             = "Started"
	taskStatusValue              = "Status"
	taskSubscriptionValue        = "Subscription"
	taskTaskValue                = "Task"
	taskTaskIDValue              = "Task ID"
	taskTimeValue                = "Time"
	taskTitleValue               = "Title"
	taskUpdatedValue             = "Updated"
	taskWorkspaceValue           = "Workspace"
	taskAttemptKey               = "attempt"
	taskClaimedByKey             = "claimed_by"
	taskCoordinationChannelIDKey = "coordination_channel_id"
	taskCreateKey                = "create"
	taskCreatedAtKey             = "created_at"
	taskDeleteIDValue            = "delete <id>"
	taskDeletedKey               = "deleted"
	taskDescriptionKey           = "description"
	taskEndedAtKey               = "ended_at"
	taskErrorKey                 = "error"
	taskGetIDValue               = "get <id>"
	taskGroupIDKey               = "group_id"
	taskIdentifierKey            = "identifier"
	taskKindKey                  = "kind"
	taskListKey                  = "list"
	taskNetworkChannelKey        = "network_channel"
	taskNextKey                  = "next"
	taskOriginKey                = "origin"
	taskOutcomeKey               = "outcome"
	taskPeerIDKey                = "peer_id"
	taskProfileKey               = "profile"
	taskQueuedAtKey              = "queued_at"
	taskReviewKey                = "review"
	taskRunIDKey                 = "run_id"
	taskScopeKey                 = "scope"
	taskSessionIDKey             = "session_id"
	taskStartedAtKey             = "started_at"
	taskStatusKey                = "status"
	taskTaskKey                  = "task"
	taskTaskIDKey                = "task_id"
	taskTimestampKey             = "timestamp"
	taskTitleKey                 = "title"
	taskUpdateIDValue            = "update <id>"
	taskUpdatedAtKey             = "updated_at"
	taskWorkspaceIDKey           = "workspace_id"
)

type taskCreateInput struct {
	ID           string
	Identifier   string
	ScopeRaw     string
	WorkspaceRef string
	NetworkRaw   string
	Title        string
	Description  string
	PriorityRaw  string
	OwnerKindRaw string
	OwnerRef     string
	MetadataRaw  string
}

type taskUpdateInput struct {
	Title        string
	Description  string
	PriorityRaw  string
	MetadataRaw  string
	NetworkRaw   string
	OwnerKindRaw string
	OwnerRef     string
	ClearOwner   bool
}

type taskExecutionInput struct {
	IdempotencyKey string
	NetworkRaw     string
	MetadataRaw    string
}

type taskReviewSubmitInput struct {
	RunID             string
	OutcomeRaw        string
	Confidence        float64
	Reason            string
	DeliveryID        string
	MissingWork       []string
	MissingWorkRaw    string
	NextRoundGuidance string
	ReviewText        string
}

type taskNotificationSubscribeInput struct {
	SubscriptionID   string
	BridgeInstanceID string
	ScopeRaw         string
	WorkspaceID      string
	PeerID           string
	ThreadID         string
	GroupID          string
	DeliveryModeRaw  string
}

type taskInspectTarget string

const (
	taskInspectTargetTask taskInspectTarget = "task"
	taskInspectTargetRun  taskInspectTarget = "run"
)

func taskInspectTargetForID(id string) taskInspectTarget {
	trimmed := strings.TrimSpace(id)
	switch {
	case strings.HasPrefix(trimmed, "task_"), strings.HasPrefix(trimmed, "task-"):
		return taskInspectTargetTask
	case strings.HasPrefix(trimmed, "run_"), strings.HasPrefix(trimmed, "run-"):
		return taskInspectTargetRun
	default:
		return ""
	}
}

func cliNow(now func() time.Time) time.Time {
	if now == nil {
		return time.Now().UTC()
	}
	return now().UTC()
}

func taskInspectUnknownIDRecord(id string, asOf time.Time) TaskInspectRecord {
	return TaskInspectRecord{
		Target:     providerModelAvailabilityUnknown,
		NextAction: providerModelAvailabilityUnknown,
		AsOf:       asOf,
		Diagnostics: []contract.DiagnosticItem{
			diagnosticitems.NewItem(
				"task.inspect.id_format_unknown",
				diagnosticcontract.CodeIDFormatUnknown,
				diagnosticcontract.CategoryTask,
				"Task inspect id format is unknown",
				"Task inspect accepts ids with task_ / task- or run_ / run- prefixes.",
				diagnosticcontract.SeverityError,
				diagnosticcontract.FreshnessLive,
				diagnosticitems.WithEvidence(map[string]any{"id": id}),
			),
		},
	}
}

func newTaskCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   taskTaskKey,
		Short: "Manage tasks and task runs",
		Example: `  # Create durable task intent without starting execution
  agh task create --scope workspace --workspace checkout-api --title "Audit auth flow"

  # Explicitly enqueue execution for an existing task
  agh task start task-123 --channel coord-run-123

  # Let the current agent session claim work
  agh task next --wait`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newTaskListCommand(deps))
	cmd.AddCommand(newTaskCreateCommand(deps))
	cmd.AddCommand(newTaskGetCommand(deps))
	cmd.AddCommand(newTaskInspectCommand(deps))
	cmd.AddCommand(newTaskUpdateCommand(deps))
	cmd.AddCommand(newTaskDeleteCommand(deps))
	cmd.AddCommand(newTaskProfileCommand(deps))
	cmd.AddCommand(newTaskReviewCommand(deps))
	cmd.AddCommand(newTaskNotificationCommand(deps))
	cmd.AddCommand(newTaskPublishCommand(deps))
	cmd.AddCommand(newTaskStartCommand(deps))
	cmd.AddCommand(newTaskApproveCommand(deps))
	cmd.AddCommand(newTaskRejectCommand(deps))
	cmd.AddCommand(newTaskCancelCommand(deps))
	cmd.AddCommand(newTaskNextCommand(deps))
	cmd.AddCommand(newTaskHeartbeatCommand(deps))
	cmd.AddCommand(newTaskCompleteCommand(deps))
	cmd.AddCommand(newTaskFailCommand(deps))
	cmd.AddCommand(newTaskReleaseCommand(deps))
	cmd.AddCommand(newTaskChildCommand(deps))
	cmd.AddCommand(newTaskDependencyCommand(deps))
	cmd.AddCommand(newTaskRunCommand(deps))
	return cmd
}

func newTaskListCommand(deps commandDeps) *cobra.Command {
	var (
		scopeRaw     string
		workspaceRef string
		statusRaw    string
		ownerKindRaw string
		ownerRef     string
		parentTaskID string
		networkRaw   string
		last         int
	)

	cmd := &cobra.Command{
		Use:   taskListKey,
		Short: "List tasks",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			query, err := parseTaskListFilters(
				scopeRaw,
				workspaceRef,
				statusRaw,
				ownerKindRaw,
				ownerRef,
				parentTaskID,
				networkRaw,
				last,
			)
			if err != nil {
				return err
			}

			tasks, err := client.ListTasks(cmd.Context(), query)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, taskSummaryListBundle(tasks))
		},
	}
	cmd.Flags().StringVar(&scopeRaw, taskScopeKey, "", "Filter by scope: global or workspace")
	cmd.Flags().StringVar(&workspaceRef, "workspace", "", "Filter by workspace path, name, or ID")
	cmd.Flags().StringVar(&statusRaw, taskStatusKey, "", "Filter by task status")
	cmd.Flags().StringVar(&ownerKindRaw, "owner-kind", "", "Filter by owner kind")
	cmd.Flags().StringVar(&ownerRef, "owner-ref", "", "Filter by owner reference")
	cmd.Flags().StringVar(&parentTaskID, "parent", "", "Filter by parent task ID")
	cmd.Flags().StringVar(&networkRaw, "channel", "", "Filter by network channel")
	cmd.Flags().IntVar(&last, "last", 0, "Show only the most recent N tasks")
	return cmd
}

func newTaskCreateCommand(deps commandDeps) *cobra.Command {
	var (
		id           string
		identifier   string
		scopeRaw     string
		workspaceRef string
		networkRaw   string
		title        string
		description  string
		ownerKindRaw string
		ownerRef     string
		metadataRaw  string
		priorityRaw  string
		asAgent      bool
	)

	cmd := &cobra.Command{
		Use:   taskCreateKey,
		Short: "Create a task",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			request, err := buildTaskCreateRequest(cmd, taskCreateInput{
				ID:           id,
				Identifier:   identifier,
				ScopeRaw:     scopeRaw,
				WorkspaceRef: workspaceRef,
				NetworkRaw:   networkRaw,
				Title:        title,
				Description:  description,
				PriorityRaw:  priorityRaw,
				OwnerKindRaw: ownerKindRaw,
				OwnerRef:     ownerRef,
				MetadataRaw:  metadataRaw,
			})
			if err != nil {
				return err
			}

			var created TaskRecord
			if asAgent {
				credentials, identityErr := requireAgentCommandIdentity(
					cmd.Context(),
					deps,
					client,
					agentActionCLI("task.create"),
				)
				if identityErr != nil {
					return identityErr
				}
				created, err = client.CreateTaskAsAgent(cmd.Context(), request, credentials)
			} else {
				created, err = client.CreateTask(cmd.Context(), request)
			}
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, taskBundle(created))
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "Explicit task ID")
	cmd.Flags().StringVar(&identifier, taskIdentifierKey, "", "Human-friendly task identifier")
	cmd.Flags().StringVar(&scopeRaw, taskScopeKey, "", "Task scope: global or workspace")
	cmd.Flags().
		StringVar(&workspaceRef, "workspace", "", "Workspace path, name, or ID (required when --scope=workspace)")
	cmd.Flags().StringVar(&networkRaw, "channel", "", "Optional network channel binding")
	cmd.Flags().StringVar(&title, taskTitleKey, "", "Task title")
	cmd.Flags().StringVar(&description, taskDescriptionKey, "", "Task description")
	cmd.Flags().StringVar(&priorityRaw, "priority", "", "Task priority: low, medium, high, or urgent")
	cmd.Flags().StringVar(&ownerKindRaw, "owner-kind", "", "Optional owner kind")
	cmd.Flags().StringVar(&ownerRef, "owner-ref", "", "Optional owner reference")
	cmd.Flags().StringVar(&metadataRaw, "metadata", "", "Optional metadata JSON")
	cmd.Flags().BoolVar(&asAgent, "as-agent", false, "Create using the current AGH-managed agent session identity")
	mustMarkFlagRequired(cmd, taskScopeKey)
	mustMarkFlagRequired(cmd, taskTitleKey)
	return cmd
}

func newTaskGetCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   taskGetIDValue,
		Short: "Show one task with related detail",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			taskDetail, err := client.GetTask(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, taskDetailBundle(&taskDetail))
		},
	}
}

func newTaskInspectCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "inspect <id>",
		Short: "Inspect a task or run with diagnostics",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			id := strings.TrimSpace(args[0])
			var inspect TaskInspectRecord
			switch taskInspectTargetForID(id) {
			case taskInspectTargetTask:
				inspect, err = client.InspectTask(cmd.Context(), id)
			case taskInspectTargetRun:
				inspect, err = client.InspectRun(cmd.Context(), id)
			default:
				inspect = taskInspectUnknownIDRecord(id, cliNow(deps.now))
			}
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, taskInspectBundle(&inspect))
		},
	}
}

func newTaskUpdateCommand(deps commandDeps) *cobra.Command {
	var (
		title        string
		description  string
		metadataRaw  string
		networkRaw   string
		priorityRaw  string
		ownerKindRaw string
		ownerRef     string
		clearOwner   bool
	)

	cmd := &cobra.Command{
		Use:   taskUpdateIDValue,
		Short: "Update mutable task fields",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			request, err := buildTaskUpdateRequest(cmd, taskUpdateInput{
				Title:        title,
				Description:  description,
				PriorityRaw:  priorityRaw,
				MetadataRaw:  metadataRaw,
				NetworkRaw:   networkRaw,
				OwnerKindRaw: ownerKindRaw,
				OwnerRef:     ownerRef,
				ClearOwner:   clearOwner,
			})
			if err != nil {
				return err
			}
			if !request.HasChanges() {
				return errors.New("cli: task update requires at least one change flag")
			}

			updated, err := client.UpdateTask(cmd.Context(), args[0], request)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, taskBundle(updated))
		},
	}
	cmd.Flags().StringVar(&title, taskTitleKey, "", "Update the task title")
	cmd.Flags().StringVar(&description, taskDescriptionKey, "", "Update the task description")
	cmd.Flags().StringVar(&priorityRaw, "priority", "", "Update the task priority: low, medium, high, or urgent")
	cmd.Flags().StringVar(&metadataRaw, "metadata", "", "Update metadata JSON")
	cmd.Flags().
		StringVar(&networkRaw, "channel", "", "Update the network channel; pass an empty value to clear it")
	cmd.Flags().StringVar(&ownerKindRaw, "owner-kind", "", "Update the owner kind")
	cmd.Flags().StringVar(&ownerRef, "owner-ref", "", "Update the owner reference")
	cmd.Flags().BoolVar(&clearOwner, "clear-owner", false, "Remove the current owner")
	return cmd
}

func buildTaskUpdateRequest(cmd *cobra.Command, input taskUpdateInput) (UpdateTaskRequest, error) {
	request := UpdateTaskRequest{}
	if cmd.Flags().Changed(taskTitleKey) {
		trimmed := strings.TrimSpace(input.Title)
		if trimmed == "" {
			return UpdateTaskRequest{}, errors.New("cli: --title cannot be blank")
		}
		request.Title = new(trimmed)
	}
	if cmd.Flags().Changed(taskDescriptionKey) {
		request.Description = new(strings.TrimSpace(input.Description))
	}
	if cmd.Flags().Changed("priority") {
		priority, err := parseOptionalTaskPriority(input.PriorityRaw)
		if err != nil {
			return UpdateTaskRequest{}, err
		}
		request.Priority = &priority
	}
	if cmd.Flags().Changed("metadata") {
		metadata, err := parseJSONFlag("metadata", input.MetadataRaw)
		if err != nil {
			return UpdateTaskRequest{}, err
		}
		request.Metadata = &metadata
	}
	if cmd.Flags().Changed("channel") {
		if err := validateTaskChannelFlag(input.NetworkRaw); err != nil {
			return UpdateTaskRequest{}, err
		}
		request.NetworkChannel = new(strings.TrimSpace(input.NetworkRaw))
	}

	ownerChanged := cmd.Flags().Changed("owner-kind") || cmd.Flags().Changed("owner-ref")
	if input.ClearOwner && ownerChanged {
		return UpdateTaskRequest{}, errors.New(
			"cli: --clear-owner cannot be combined with --owner-kind or --owner-ref",
		)
	}
	if ownerChanged {
		owner, err := parseRequiredTaskOwnership(input.OwnerKindRaw, input.OwnerRef)
		if err != nil {
			return UpdateTaskRequest{}, err
		}
		request.Owner = owner
	}
	if input.ClearOwner {
		request.ClearOwner = true
	}
	return request, nil
}

func newTaskPublishCommand(deps commandDeps) *cobra.Command {
	return newTaskExecutionCommand(
		deps,
		"publish <id>",
		"Publish a draft task and enqueue its first run",
		func(ctx context.Context, client DaemonClient, id string, request TaskExecutionRequest) (TaskExecutionRecord, error) {
			return client.PublishTask(ctx, id, request)
		},
	)
}

func newTaskStartCommand(deps commandDeps) *cobra.Command {
	return newTaskExecutionCommand(
		deps,
		"start <id>",
		"Enqueue a run for an executable task",
		func(ctx context.Context, client DaemonClient, id string, request TaskExecutionRequest) (TaskExecutionRecord, error) {
			return client.StartTask(ctx, id, request)
		},
	)
}

func newTaskApproveCommand(deps commandDeps) *cobra.Command {
	return newTaskExecutionCommand(
		deps,
		"approve <id>",
		"Approve a task and enqueue its first run",
		func(ctx context.Context, client DaemonClient, id string, request TaskExecutionRequest) (TaskExecutionRecord, error) {
			return client.ApproveTask(ctx, id, request)
		},
	)
}

func newTaskDeleteCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   taskDeleteIDValue,
		Short: "Delete a task",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			if err := client.DeleteTask(cmd.Context(), args[0]); err != nil {
				return err
			}
			return writeCommandOutput(cmd, taskDeleteBundle(args[0]))
		},
	}
	return cmd
}

func newTaskProfileCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   taskProfileKey,
		Short: "Manage task execution profiles",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newTaskProfileInspectCommand(deps))
	cmd.AddCommand(newTaskProfileUpdateCommand(deps))
	cmd.AddCommand(newTaskProfileDeleteCommand(deps))
	return cmd
}

func newTaskProfileInspectCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "inspect <id>",
		Short: "Show one task execution profile",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			profile, err := client.GetTaskExecutionProfile(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, taskExecutionProfileBundle(&profile))
		},
	}
}

func newTaskProfileUpdateCommand(deps commandDeps) *cobra.Command {
	var profileRaw string
	cmd := &cobra.Command{
		Use:   taskUpdateIDValue,
		Short: "Replace one task execution profile",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			request, err := buildTaskExecutionProfileRequest(args[0], profileRaw)
			if err != nil {
				return err
			}
			profile, err := client.SetTaskExecutionProfile(cmd.Context(), args[0], request)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, taskExecutionProfileBundle(&profile))
		},
	}
	cmd.Flags().StringVar(&profileRaw, taskProfileKey, "", "Task execution profile JSON")
	mustMarkFlagRequired(cmd, taskProfileKey)
	return cmd
}

func newTaskProfileDeleteCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   taskDeleteIDValue,
		Short: "Delete one task execution profile",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			if err := client.DeleteTaskExecutionProfile(cmd.Context(), args[0]); err != nil {
				return err
			}
			return writeCommandOutput(cmd, taskExecutionProfileDeleteBundle(args[0]))
		},
	}
}

func buildTaskExecutionProfileRequest(taskID string, raw string) (*TaskExecutionProfileRequest, error) {
	payload, err := parseJSONFlag(taskProfileKey, raw)
	if err != nil {
		return nil, err
	}
	var request TaskExecutionProfileRequest
	if err := json.Unmarshal(payload, &request); err != nil {
		return nil, fmt.Errorf("cli: parse --profile JSON: %w", err)
	}
	trimmedID := strings.TrimSpace(taskID)
	if strings.TrimSpace(request.TaskID) != "" && strings.TrimSpace(request.TaskID) != trimmedID {
		return nil, fmt.Errorf(
			"cli: profile.task_id must match task id %q",
			trimmedID,
		)
	}
	request.TaskID = trimmedID
	return &request, nil
}

func newTaskNotificationCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "notification",
		Short: "Manage task terminal notifications",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newTaskNotificationSubscribeCommand(deps))
	cmd.AddCommand(newTaskNotificationListCommand(deps))
	cmd.AddCommand(newTaskNotificationShowCommand(deps))
	cmd.AddCommand(newTaskNotificationDeleteCommand(deps))
	return cmd
}

func newTaskNotificationSubscribeCommand(deps commandDeps) *cobra.Command {
	input := taskNotificationSubscribeInput{}
	cmd := &cobra.Command{
		Use:   "subscribe <task-id>",
		Short: "Subscribe a bridge target to task terminal notifications",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			request, err := buildTaskBridgeNotificationSubscriptionRequest(input)
			if err != nil {
				return err
			}
			subscription, err := client.CreateTaskBridgeNotificationSubscription(cmd.Context(), args[0], request)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, taskBridgeNotificationSubscriptionBundle(&subscription))
		},
	}
	cmd.Flags().StringVar(&input.SubscriptionID, "subscription-id", "", "Idempotent subscription ID")
	cmd.Flags().StringVar(&input.BridgeInstanceID, "bridge", "", "Bridge instance ID")
	cmd.Flags().
		StringVar(&input.ScopeRaw, taskScopeKey, string(bridgepkg.ScopeGlobal), "Bridge scope: global or workspace")
	cmd.Flags().StringVar(&input.WorkspaceID, "workspace", "", "Workspace ID for workspace bridge scope")
	cmd.Flags().StringVar(&input.PeerID, "peer", "", "Bridge peer ID")
	cmd.Flags().StringVar(&input.ThreadID, "thread", "", "Bridge thread ID")
	cmd.Flags().StringVar(&input.GroupID, "group", "", "Bridge group ID")
	cmd.Flags().StringVar(
		&input.DeliveryModeRaw,
		"mode",
		string(bridgepkg.DeliveryModeReply),
		"Delivery mode: reply or direct-send",
	)
	mustMarkFlagRequired(cmd, "bridge")
	return cmd
}

func newTaskNotificationListCommand(deps commandDeps) *cobra.Command {
	var (
		bridgeInstanceID string
		scopeRaw         string
		workspaceID      string
		last             int
	)
	cmd := &cobra.Command{
		Use:   "list <task-id>",
		Short: "List bridge terminal notification subscriptions for one task",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			query, err := buildTaskBridgeNotificationSubscriptionListQuery(
				bridgeInstanceID,
				scopeRaw,
				workspaceID,
				last,
			)
			if err != nil {
				return err
			}
			subscriptions, err := client.ListTaskBridgeNotificationSubscriptions(cmd.Context(), args[0], query)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, taskBridgeNotificationSubscriptionListBundle(subscriptions))
		},
	}
	cmd.Flags().StringVar(&bridgeInstanceID, "bridge", "", "Filter by bridge instance ID")
	cmd.Flags().StringVar(&scopeRaw, taskScopeKey, "", "Filter by bridge scope: global or workspace")
	cmd.Flags().StringVar(&workspaceID, "workspace", "", "Filter by workspace ID")
	cmd.Flags().IntVar(&last, "last", 0, "Show only the most recent N subscriptions")
	return cmd
}

func newTaskNotificationShowCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "show <task-id> <subscription-id>",
		Short: "Show one bridge terminal notification subscription",
		Args:  exactTwoNonBlankArgs(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			subscription, err := client.GetTaskBridgeNotificationSubscription(cmd.Context(), args[0], args[1])
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, taskBridgeNotificationSubscriptionBundle(&subscription))
		},
	}
}

func newTaskNotificationDeleteCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <task-id> <subscription-id>",
		Short: "Delete one bridge terminal notification subscription",
		Args:  exactTwoNonBlankArgs(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			if err := client.DeleteTaskBridgeNotificationSubscription(cmd.Context(), args[0], args[1]); err != nil {
				return err
			}
			return writeCommandOutput(cmd, taskBridgeNotificationSubscriptionDeleteBundle(args[0], args[1]))
		},
	}
}

func buildTaskBridgeNotificationSubscriptionRequest(
	input taskNotificationSubscribeInput,
) (*TaskBridgeNotificationSubscriptionRequest, error) {
	scope := bridgepkg.Scope(strings.TrimSpace(input.ScopeRaw)).Normalize()
	if err := scope.Validate(); err != nil {
		return nil, fmt.Errorf("cli: invalid notification scope: %w", err)
	}
	mode := bridgepkg.DeliveryMode(input.DeliveryModeRaw).Normalize()
	if err := mode.Validate(); err != nil {
		return nil, fmt.Errorf("cli: invalid delivery mode: %w", err)
	}
	request := TaskBridgeNotificationSubscriptionRequest{
		SubscriptionID:   strings.TrimSpace(input.SubscriptionID),
		BridgeInstanceID: strings.TrimSpace(input.BridgeInstanceID),
		Scope:            scope,
		WorkspaceID:      strings.TrimSpace(input.WorkspaceID),
		PeerID:           strings.TrimSpace(input.PeerID),
		ThreadID:         strings.TrimSpace(input.ThreadID),
		GroupID:          strings.TrimSpace(input.GroupID),
		DeliveryMode:     mode,
	}
	if err := bridgepkg.ValidateScopeWorkspaceID(request.Scope, request.WorkspaceID); err != nil {
		return nil, fmt.Errorf("cli: invalid notification scope: %w", err)
	}
	if strings.TrimSpace(request.PeerID) == "" && strings.TrimSpace(request.GroupID) == "" {
		return nil, errors.New("cli: notification subscription requires --peer or --group")
	}
	return &request, nil
}

func buildTaskBridgeNotificationSubscriptionListQuery(
	bridgeInstanceID string,
	scopeRaw string,
	workspaceID string,
	last int,
) (TaskBridgeNotificationSubscriptionQuery, error) {
	if last < 0 {
		return TaskBridgeNotificationSubscriptionQuery{}, errors.New("cli: --last must be zero or positive")
	}
	query := TaskBridgeNotificationSubscriptionQuery{
		BridgeInstanceID: strings.TrimSpace(bridgeInstanceID),
		WorkspaceID:      strings.TrimSpace(workspaceID),
		Limit:            last,
	}
	if trimmed := strings.TrimSpace(scopeRaw); trimmed != "" {
		scope := bridgepkg.Scope(trimmed).Normalize()
		if err := scope.Validate(); err != nil {
			return TaskBridgeNotificationSubscriptionQuery{}, fmt.Errorf("cli: invalid notification scope: %w", err)
		}
		query.Scope = scope
	}
	return query, nil
}

func newTaskReviewCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   taskReviewKey,
		Short: "Manage task-run reviews",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newTaskReviewRequestCommand(deps))
	cmd.AddCommand(newTaskReviewListCommand(deps))
	cmd.AddCommand(newTaskReviewShowCommand(deps))
	cmd.AddCommand(newTaskReviewSubmitCommand(deps))
	return cmd
}

func newTaskReviewRequestCommand(deps commandDeps) *cobra.Command {
	var (
		reasonRaw string
		policyRaw string
		round     int
		attempt   int
		parentID  string
		asAgent   bool
	)
	cmd := &cobra.Command{
		Use:   "request <run-id>",
		Short: "Request review for a task run",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			request, err := buildTaskRunReviewRequest(args[0], reasonRaw, policyRaw, round, attempt, parentID)
			if err != nil {
				return err
			}
			var review TaskRunReviewRequestRecord
			if asAgent {
				credentials, identityErr := requireAgentCommandIdentity(
					cmd.Context(),
					deps,
					client,
					agentActionCLI("task.review.request"),
				)
				if identityErr != nil {
					return identityErr
				}
				review, err = client.RequestTaskRunReviewAsAgent(cmd.Context(), args[0], request, credentials)
			} else {
				review, err = client.RequestTaskRunReview(cmd.Context(), args[0], request)
			}
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, taskRunReviewRequestBundle(&review))
		},
	}
	cmd.Flags().StringVar(&reasonRaw, "reason", "", "Reason for requesting review")
	cmd.Flags().StringVar(&policyRaw, "policy", "", "Review policy: always, on_success, or on_failure")
	cmd.Flags().IntVar(&round, "round", 0, "Review round number")
	cmd.Flags().IntVar(&attempt, taskAttemptKey, 0, "Review attempt number")
	cmd.Flags().StringVar(&parentID, "parent-review", "", "Parent review ID for continuation rounds")
	cmd.Flags().
		BoolVar(&asAgent, "as-agent", false, "Request review using the current AGH-managed agent session identity")
	return cmd
}

func newTaskReviewListCommand(deps commandDeps) *cobra.Command {
	var (
		taskID            string
		runID             string
		statusRaw         string
		reviewerSessionID string
		last              int
	)
	cmd := &cobra.Command{
		Use:   taskListKey,
		Short: "List task-run reviews",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			query, err := parseTaskRunReviewListFilters(taskID, runID, statusRaw, reviewerSessionID, last)
			if err != nil {
				return err
			}
			reviews, err := client.ListTaskRunReviews(cmd.Context(), query)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, taskRunReviewListBundle(reviews))
		},
	}
	cmd.Flags().StringVar(&taskID, taskTaskKey, "", "Filter by task ID")
	cmd.Flags().StringVar(&runID, "run", "", "Filter by task run ID")
	cmd.Flags().StringVar(&statusRaw, taskStatusKey, "", "Filter by review status")
	cmd.Flags().StringVar(&reviewerSessionID, "reviewer-session", "", "Filter by reviewer session ID")
	cmd.Flags().IntVar(&last, "last", 0, "Show only the most recent N reviews")
	return cmd
}

func newTaskReviewShowCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "show <review-id>",
		Short: "Show one task-run review",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			review, err := client.GetTaskRunReview(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, taskRunReviewBundle(&review))
		},
	}
}

func newTaskReviewSubmitCommand(deps commandDeps) *cobra.Command {
	input := taskReviewSubmitInput{}
	var asAgent bool
	cmd := &cobra.Command{
		Use:   "submit <review-id>",
		Short: "Submit one task-run review verdict",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			request, err := buildTaskRunReviewVerdictRequest(args[0], input)
			if err != nil {
				return err
			}
			var result TaskRunReviewVerdictRecord
			if asAgent {
				credentials, identityErr := requireAgentCommandIdentity(
					cmd.Context(),
					deps,
					client,
					agentActionCLI("task.review.submit"),
				)
				if identityErr != nil {
					return identityErr
				}
				result, err = client.SubmitTaskRunReviewVerdictAsAgent(cmd.Context(), args[0], request, credentials)
			} else {
				result, err = client.SubmitTaskRunReviewVerdict(cmd.Context(), args[0], request)
			}
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, taskRunReviewVerdictBundle(&result))
		},
	}
	cmd.Flags().StringVar(&input.RunID, "run", "", "Task run ID")
	cmd.Flags().StringVar(&input.OutcomeRaw, cliOutcomeKey, "", "Verdict outcome")
	cmd.Flags().Float64Var(&input.Confidence, "confidence", 0, "Verdict confidence from 0 to 1")
	cmd.Flags().StringVar(&input.Reason, "reason", "", "Verdict reason")
	cmd.Flags().StringVar(&input.DeliveryID, "delivery-id", "", "Idempotent delivery ID")
	cmd.Flags().StringArrayVar(&input.MissingWork, "missing-work", nil, "Missing work item (repeatable)")
	cmd.Flags().StringVar(&input.MissingWorkRaw, "missing-work-json", "", "Missing work JSON array")
	cmd.Flags().StringVar(&input.NextRoundGuidance, "next-round-guidance", "", "Guidance for continuation rounds")
	cmd.Flags().StringVar(&input.ReviewText, "review-text", "", "Full review text")
	cmd.Flags().
		BoolVar(&asAgent, "as-agent", false, "Submit review using the current AGH-managed agent session identity")
	mustMarkFlagRequired(cmd, "run")
	mustMarkFlagRequired(cmd, cliOutcomeKey)
	mustMarkFlagRequired(cmd, "confidence")
	mustMarkFlagRequired(cmd, "reason")
	mustMarkFlagRequired(cmd, "delivery-id")
	return cmd
}

func buildTaskRunReviewRequest(
	runID string,
	reasonRaw string,
	policyRaw string,
	round int,
	attempt int,
	parentID string,
) (*TaskRunReviewRequest, error) {
	if round < 0 {
		return nil, errors.New("cli: --round must be zero or positive")
	}
	if attempt < 0 {
		return nil, errors.New("cli: --attempt must be zero or positive")
	}
	policy, err := parseOptionalReviewPolicy(policyRaw)
	if err != nil {
		return nil, err
	}
	request := TaskRunReviewRequest{
		RunID:          strings.TrimSpace(runID),
		ReviewRound:    round,
		Attempt:        attempt,
		Policy:         policy,
		ParentReviewID: strings.TrimSpace(parentID),
		Reason:         strings.TrimSpace(reasonRaw),
	}
	return &request, nil
}

func buildTaskRunReviewVerdictRequest(
	reviewID string,
	input taskReviewSubmitInput,
) (*TaskRunReviewVerdictRequest, error) {
	outcome, err := parseRequiredReviewOutcome(input.OutcomeRaw)
	if err != nil {
		return nil, err
	}
	confidence := input.Confidence
	missingWork, err := missingWorkFromFlags(input.MissingWork, input.MissingWorkRaw)
	if err != nil {
		return nil, err
	}
	request := &TaskRunReviewVerdictRequest{
		RunID: strings.TrimSpace(input.RunID),
		Verdict: taskpkg.RunReviewVerdict{
			Outcome:           outcome,
			Confidence:        &confidence,
			Reason:            strings.TrimSpace(input.Reason),
			DeliveryID:        strings.TrimSpace(input.DeliveryID),
			MissingWork:       missingWork,
			NextRoundGuidance: strings.TrimSpace(input.NextRoundGuidance),
			ReviewText:        strings.TrimSpace(input.ReviewText),
		},
	}
	recordReq := taskpkg.RecordRunReviewRequest{
		ReviewID: strings.TrimSpace(reviewID),
		RunID:    request.RunID,
		Verdict:  request.Verdict,
	}.Normalize()
	if err := recordReq.Validate("task_run_review_verdict"); err != nil {
		return nil, fmt.Errorf("cli: %w", err)
	}
	request.Verdict = recordReq.Verdict
	return request, nil
}

func parseTaskRunReviewListFilters(
	taskID string,
	runID string,
	statusRaw string,
	reviewerSessionID string,
	last int,
) (TaskRunReviewListQuery, error) {
	trimmedTaskID := strings.TrimSpace(taskID)
	trimmedRunID := strings.TrimSpace(runID)
	trimmedReviewerSessionID := strings.TrimSpace(reviewerSessionID)
	if trimmedTaskID != "" && trimmedRunID != "" {
		return TaskRunReviewListQuery{}, errors.New("cli: choose either --task or --run")
	}
	status, err := parseOptionalReviewStatus(statusRaw)
	if err != nil {
		return TaskRunReviewListQuery{}, err
	}
	if trimmedTaskID == "" && trimmedRunID == "" && status == "" && trimmedReviewerSessionID == "" && last == 0 {
		return TaskRunReviewListQuery{}, errors.New("cli: task review list requires at least one filter")
	}
	if err := validateTaskLast(last); err != nil {
		return TaskRunReviewListQuery{}, err
	}
	return TaskRunReviewListQuery{
		TaskID:            trimmedTaskID,
		RunID:             trimmedRunID,
		Status:            status,
		ReviewerSessionID: trimmedReviewerSessionID,
		Limit:             last,
	}, nil
}

func newTaskRejectCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reject <id>",
		Short: "Reject a pending approval task",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			rejected, err := client.RejectTask(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, taskBundle(rejected))
		},
	}
	return cmd
}

func newTaskExecutionCommand(
	deps commandDeps,
	use string,
	short string,
	execute func(context.Context, DaemonClient, string, TaskExecutionRequest) (TaskExecutionRecord, error),
) *cobra.Command {
	var input taskExecutionInput
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			request, err := buildTaskExecutionRequest(cmd, input)
			if err != nil {
				return err
			}
			execution, err := execute(cmd.Context(), client, args[0], request)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, taskExecutionBundle(&execution))
		},
	}
	cmd.Flags().StringVar(&input.IdempotencyKey, "idempotency-key", "", "Optional idempotency key")
	cmd.Flags().StringVar(&input.NetworkRaw, "channel", "", "Optional run channel override")
	cmd.Flags().StringVar(&input.MetadataRaw, "metadata", "", "Optional run metadata JSON")
	return cmd
}

func buildTaskExecutionRequest(cmd *cobra.Command, input taskExecutionInput) (TaskExecutionRequest, error) {
	if err := validateTaskChannelFlag(input.NetworkRaw); err != nil {
		return TaskExecutionRequest{}, err
	}
	request := TaskExecutionRequest{
		IdempotencyKey: strings.TrimSpace(input.IdempotencyKey),
		NetworkChannel: strings.TrimSpace(input.NetworkRaw),
	}
	if cmd.Flags().Changed("metadata") {
		metadata, err := parseJSONFlag("metadata", input.MetadataRaw)
		if err != nil {
			return TaskExecutionRequest{}, err
		}
		request.Metadata = metadata
	}
	return request, nil
}

func newTaskCancelCommand(deps commandDeps) *cobra.Command {
	var (
		reason      string
		metadataRaw string
	)

	cmd := &cobra.Command{
		Use:   "cancel <id>",
		Short: "Cancel a task tree",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			request := CancelTaskRequest{Reason: strings.TrimSpace(reason)}
			if cmd.Flags().Changed("metadata") {
				request.Metadata, err = parseJSONFlag("metadata", metadataRaw)
				if err != nil {
					return err
				}
			}

			canceled, err := client.CancelTask(cmd.Context(), args[0], request)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, taskBundle(canceled))
		},
	}
	cmd.Flags().StringVar(&reason, "reason", "", "Optional cancellation reason")
	cmd.Flags().StringVar(&metadataRaw, "metadata", "", "Optional cancellation metadata JSON")
	return cmd
}

func newTaskNextCommand(deps commandDeps) *cobra.Command {
	var (
		workspaceID          string
		requiredCapabilities []string
		priorityMin          int
		leaseSeconds         int64
		wait                 bool
		idempotencyKey       string
	)

	cmd := &cobra.Command{
		Use:   taskNextKey,
		Short: "Claim the next task run for the current agent session",
		Args:  cobra.NoArgs,
		Example: `  # Claim the next available run for this session
  agh task next

  # Wait until matching work is claimable and request a five-minute lease
  agh task next --wait --lease-seconds 300 -o json

  # Filter by required caller capability
  agh task next --capability go.test --priority-min 10`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := validateAgentTaskLeaseSeconds(leaseSeconds); err != nil {
				return err
			}
			if priorityMin < 0 {
				return fmt.Errorf("cli: --priority-min must be zero or positive: %d", priorityMin)
			}
			request := AgentTaskClaimNextRequest{
				WorkspaceID:          strings.TrimSpace(workspaceID),
				RequiredCapabilities: trimAgentTaskCapabilities(requiredCapabilities),
				PriorityMin:          priorityMin,
				LeaseSeconds:         leaseSeconds,
				Wait:                 wait,
				IdempotencyKey:       strings.TrimSpace(idempotencyKey),
			}

			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			credentials, err := requireAgentCommandIdentity(cmd.Context(), deps, client, agentActionCLI("task.next"))
			if err != nil {
				return err
			}
			record, err := client.AgentTaskClaimNext(cmd.Context(), request, credentials)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, agentTaskNextBundle(record))
		},
	}
	cmd.Flags().StringVar(&workspaceID, "workspace-id", "", "Workspace ID override; defaults to caller workspace")
	cmd.Flags().StringArrayVar(&requiredCapabilities, "capability", nil, "Caller capability filter (repeatable)")
	cmd.Flags().IntVar(&priorityMin, "priority-min", 0, "Minimum task priority")
	cmd.Flags().Int64Var(&leaseSeconds, "lease-seconds", 0, "Lease duration in seconds")
	cmd.Flags().BoolVar(&wait, "wait", false, "Wait until work is claimable")
	cmd.Flags().StringVar(&idempotencyKey, "idempotency-key", "", "Optional idempotency key")
	return cmd
}

func newTaskHeartbeatCommand(deps commandDeps) *cobra.Command {
	var leaseSeconds int64

	cmd := &cobra.Command{
		Use:   "heartbeat <run-id>",
		Short: "Extend a claimed task run lease for the current agent session",
		Args:  cobra.ExactArgs(1),
		Example: `  # Extend the active session-bound lease
  agh task heartbeat run-123

	  # Request a specific lease duration
  agh task heartbeat run-123 --lease-seconds 300`,
		RunE: func(cmd *cobra.Command, args []string) error {
			runID, err := requiredAgentTaskRunID(args[0])
			if err != nil {
				return err
			}
			if err := validateAgentTaskLeaseSeconds(leaseSeconds); err != nil {
				return err
			}
			request := AgentTaskHeartbeatRequest{
				LeaseSeconds: leaseSeconds,
			}

			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			credentials, err := requireAgentCommandIdentity(
				cmd.Context(),
				deps,
				client,
				agentActionCLI("task.heartbeat"),
			)
			if err != nil {
				return err
			}
			record, err := client.AgentTaskHeartbeat(cmd.Context(), runID, request, credentials)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, agentTaskLeaseBundle(record))
		},
	}
	cmd.Flags().Int64Var(&leaseSeconds, "lease-seconds", 0, "Lease duration in seconds")
	return cmd
}

func newTaskCompleteCommand(deps commandDeps) *cobra.Command {
	var resultRaw string

	cmd := &cobra.Command{
		Use:   "complete <run-id>",
		Short: "Complete a claimed task run for the current agent session",
		Args:  cobra.ExactArgs(1),
		Example: `  # Complete a claimed run
  agh task complete run-123

	  # Complete with structured result data
  agh task complete run-123 --result '{"summary":"tests passed"}'`,
		RunE: func(cmd *cobra.Command, args []string) error {
			runID, err := requiredAgentTaskRunID(args[0])
			if err != nil {
				return err
			}
			request := AgentTaskCompleteRequest{}
			if cmd.Flags().Changed("result") {
				request.Result, err = parseAgentTaskJSONFlag("result", resultRaw)
				if err != nil {
					return err
				}
			}

			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			credentials, err := requireAgentCommandIdentity(
				cmd.Context(),
				deps,
				client,
				agentActionCLI("task.complete"),
			)
			if err != nil {
				return err
			}
			record, err := client.AgentTaskComplete(cmd.Context(), runID, request, credentials)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, agentTaskLeaseBundle(record))
		},
	}
	cmd.Flags().StringVar(&resultRaw, "result", "", "Optional result JSON")
	return cmd
}

func newTaskFailCommand(deps commandDeps) *cobra.Command {
	var errorMessage string
	var metadataRaw string

	cmd := &cobra.Command{
		Use:   "fail <run-id>",
		Short: "Fail a claimed task run for the current agent session",
		Args:  cobra.ExactArgs(1),
		Example: `  # Mark a claimed run failed
  agh task fail run-123 --error "tests failed"

	  # Include structured failure metadata
	  agh task fail run-123 \
	    --error "tests failed" \
	    --metadata '{"command":"make test"}'`,
		RunE: func(cmd *cobra.Command, args []string) error {
			runID, err := requiredAgentTaskRunID(args[0])
			if err != nil {
				return err
			}
			request := AgentTaskFailRequest{
				Error: strings.TrimSpace(errorMessage),
			}
			if request.Error == "" {
				return errors.New("cli: --error is required")
			}
			if cmd.Flags().Changed("metadata") {
				request.Metadata, err = parseAgentTaskJSONFlag("metadata", metadataRaw)
				if err != nil {
					return err
				}
			}

			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			credentials, err := requireAgentCommandIdentity(cmd.Context(), deps, client, agentActionCLI("task.fail"))
			if err != nil {
				return err
			}
			record, err := client.AgentTaskFail(cmd.Context(), runID, request, credentials)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, agentTaskLeaseBundle(record))
		},
	}
	cmd.Flags().StringVar(&errorMessage, taskErrorKey, "", "Failure message")
	cmd.Flags().StringVar(&metadataRaw, "metadata", "", "Optional failure metadata JSON")
	mustMarkFlagRequired(cmd, taskErrorKey)
	return cmd
}

func newTaskReleaseCommand(deps commandDeps) *cobra.Command {
	var reason string

	cmd := &cobra.Command{
		Use:   "release <run-id>",
		Short: "Release a claimed task run for the current agent session",
		Args:  cobra.ExactArgs(1),
		Example: `  # Release a claim without completing the run
  agh task release run-123

	  # Include a structured reason for observability
  agh task release run-123 --reason handoff`,
		RunE: func(cmd *cobra.Command, args []string) error {
			runID, err := requiredAgentTaskRunID(args[0])
			if err != nil {
				return err
			}
			request := AgentTaskReleaseRequest{
				Reason: strings.TrimSpace(reason),
			}

			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			credentials, err := requireAgentCommandIdentity(
				cmd.Context(),
				deps,
				client,
				agentActionCLI("task.release"),
			)
			if err != nil {
				return err
			}
			record, err := client.AgentTaskRelease(cmd.Context(), runID, request, credentials)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, agentTaskLeaseBundle(record))
		},
	}
	cmd.Flags().StringVar(&reason, "reason", "", "Optional release reason")
	return cmd
}

func newTaskChildCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "child",
		Short: "Manage child tasks",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newTaskChildCreateCommand(deps))
	return cmd
}

func newTaskChildCreateCommand(deps commandDeps) *cobra.Command {
	var (
		id           string
		identifier   string
		scopeRaw     string
		workspaceRef string
		networkRaw   string
		title        string
		description  string
		priorityRaw  string
		ownerKindRaw string
		ownerRef     string
		metadataRaw  string
	)

	cmd := &cobra.Command{
		Use:   "create <parent-id>",
		Short: "Create a child task beneath a parent",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			baseRequest, err := buildTaskCreateRequest(cmd, taskCreateInput{
				ID:           id,
				Identifier:   identifier,
				ScopeRaw:     scopeRaw,
				WorkspaceRef: workspaceRef,
				NetworkRaw:   networkRaw,
				Title:        title,
				Description:  description,
				PriorityRaw:  priorityRaw,
				OwnerKindRaw: ownerKindRaw,
				OwnerRef:     ownerRef,
				MetadataRaw:  metadataRaw,
			})
			if err != nil {
				return err
			}
			request := CreateTaskChildRequest(baseRequest)

			created, err := client.CreateChildTask(cmd.Context(), args[0], request)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, taskBundle(created))
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "Explicit child task ID")
	cmd.Flags().StringVar(&identifier, taskIdentifierKey, "", "Human-friendly child task identifier")
	cmd.Flags().StringVar(&scopeRaw, taskScopeKey, "", "Child task scope: global or workspace")
	cmd.Flags().
		StringVar(&workspaceRef, "workspace", "", "Workspace path, name, or ID (required when --scope=workspace)")
	cmd.Flags().StringVar(&networkRaw, "channel", "", "Optional network channel binding")
	cmd.Flags().StringVar(&title, taskTitleKey, "", "Child task title")
	cmd.Flags().StringVar(&description, taskDescriptionKey, "", "Child task description")
	cmd.Flags().StringVar(&priorityRaw, "priority", "", "Child task priority: low, medium, high, or urgent")
	cmd.Flags().StringVar(&ownerKindRaw, "owner-kind", "", "Optional child owner kind")
	cmd.Flags().StringVar(&ownerRef, "owner-ref", "", "Optional child owner reference")
	cmd.Flags().StringVar(&metadataRaw, "metadata", "", "Optional child metadata JSON")
	mustMarkFlagRequired(cmd, taskScopeKey)
	mustMarkFlagRequired(cmd, taskTitleKey)
	return cmd
}

func newTaskDependencyCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dependency",
		Short: "Manage task dependencies",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newTaskDependencyAddCommand(deps))
	cmd.AddCommand(newTaskDependencyRemoveCommand(deps))
	return cmd
}

func newTaskDependencyAddCommand(deps commandDeps) *cobra.Command {
	var (
		dependsOnID string
		kindRaw     string
	)

	cmd := &cobra.Command{
		Use:   "add <task-id>",
		Short: "Add a dependency edge to a task",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			request := AddTaskDependencyRequest{DependsOnTaskID: strings.TrimSpace(dependsOnID)}
			if request.DependsOnTaskID == "" {
				return errors.New("cli: --depends-on is required")
			}
			if strings.TrimSpace(kindRaw) != "" {
				kind, err := parseOptionalTaskDependencyKind(kindRaw)
				if err != nil {
					return err
				}
				request.Kind = kind
			}

			updated, err := client.AddTaskDependency(cmd.Context(), args[0], request)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, taskDetailBundle(&updated))
		},
	}
	cmd.Flags().StringVar(&dependsOnID, "depends-on", "", "Dependency task ID")
	cmd.Flags().StringVar(&kindRaw, taskKindKey, "", "Dependency kind")
	mustMarkFlagRequired(cmd, "depends-on")
	return cmd
}

func newTaskDependencyRemoveCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <task-id> <depends-on-id>",
		Short: "Remove a dependency edge from a task",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			updated, err := client.RemoveTaskDependency(cmd.Context(), args[0], args[1])
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, taskDetailBundle(&updated))
		},
	}
}

func newTaskRunCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Manage task runs",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newTaskRunListCommand(deps))
	cmd.AddCommand(newTaskRunEnqueueCommand(deps))
	cmd.AddCommand(newTaskRunClaimCommand(deps))
	cmd.AddCommand(newTaskRunStartCommand(deps))
	cmd.AddCommand(newTaskRunAttachSessionCommand(deps))
	cmd.AddCommand(newTaskRunCompleteCommand(deps))
	cmd.AddCommand(newTaskRunFailCommand(deps))
	cmd.AddCommand(newTaskRunCancelCommand(deps))
	return cmd
}

func newTaskRunListCommand(deps commandDeps) *cobra.Command {
	var (
		statusRaw string
		sessionID string
		last      int
	)

	cmd := &cobra.Command{
		Use:   "list <task-id>",
		Short: "List runs for a task",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			query, err := parseTaskRunListFilters(statusRaw, sessionID, last)
			if err != nil {
				return err
			}

			runs, err := client.ListTaskRuns(cmd.Context(), args[0], query)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, taskRunListBundle(runs))
		},
	}
	cmd.Flags().StringVar(&statusRaw, taskStatusKey, "", "Filter by run status")
	cmd.Flags().StringVar(&sessionID, "session", "", "Filter by attached session ID")
	cmd.Flags().IntVar(&last, "last", 0, "Show only the most recent N runs")
	return cmd
}

func newTaskRunEnqueueCommand(deps commandDeps) *cobra.Command {
	var (
		idempotencyKey string
		networkRaw     string
		metadataRaw    string
	)

	cmd := &cobra.Command{
		Use:   "enqueue <task-id>",
		Short: "Enqueue a task run",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			if err := validateTaskChannelFlag(networkRaw); err != nil {
				return err
			}

			request := EnqueueTaskRunRequest{
				IdempotencyKey: strings.TrimSpace(idempotencyKey),
				NetworkChannel: strings.TrimSpace(networkRaw),
			}
			if cmd.Flags().Changed("metadata") {
				request.Metadata, err = parseJSONFlag("metadata", metadataRaw)
				if err != nil {
					return err
				}
			}

			run, err := client.EnqueueTaskRun(cmd.Context(), args[0], request)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, taskRunBundle(run))
		},
	}
	cmd.Flags().StringVar(&idempotencyKey, "idempotency-key", "", "Optional idempotency key")
	cmd.Flags().StringVar(&networkRaw, "channel", "", "Optional run channel override")
	cmd.Flags().StringVar(&metadataRaw, "metadata", "", "Optional run metadata JSON")
	return cmd
}

func newTaskRunClaimCommand(deps commandDeps) *cobra.Command {
	var idempotencyKey string

	cmd := &cobra.Command{
		Use:   "claim <run-id>",
		Short: "Claim a queued task run",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			run, err := client.ClaimTaskRun(cmd.Context(), args[0], ClaimTaskRunRequest{
				IdempotencyKey: strings.TrimSpace(idempotencyKey),
			})
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, taskRunBundle(run))
		},
	}
	cmd.Flags().StringVar(&idempotencyKey, "idempotency-key", "", "Optional idempotency key")
	return cmd
}

func newTaskRunStartCommand(deps commandDeps) *cobra.Command {
	var idempotencyKey string

	cmd := &cobra.Command{
		Use:   "start <run-id>",
		Short: "Start a claimed task run",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			run, err := client.StartTaskRun(cmd.Context(), args[0], StartTaskRunRequest{
				IdempotencyKey: strings.TrimSpace(idempotencyKey),
			})
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, taskRunBundle(run))
		},
	}
	cmd.Flags().StringVar(&idempotencyKey, "idempotency-key", "", "Optional idempotency key")
	return cmd
}

func newTaskRunAttachSessionCommand(deps commandDeps) *cobra.Command {
	var sessionID string

	cmd := &cobra.Command{
		Use:   "attach-session <run-id>",
		Short: "Attach an existing session to a claimed or starting run",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			if strings.TrimSpace(sessionID) == "" {
				return errors.New("cli: --session is required")
			}

			run, err := client.AttachTaskRunSession(
				cmd.Context(),
				args[0],
				AttachTaskRunSessionRequest{
					SessionID: strings.TrimSpace(sessionID),
				},
			)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, taskRunBundle(run))
		},
	}
	cmd.Flags().StringVar(&sessionID, "session", "", "Existing session ID to attach")
	mustMarkFlagRequired(cmd, "session")
	return cmd
}

func newTaskRunCompleteCommand(deps commandDeps) *cobra.Command {
	var resultRaw string

	cmd := &cobra.Command{
		Use:   "complete <run-id>",
		Short: "Complete a running task run",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			request := CompleteTaskRunRequest{}
			if cmd.Flags().Changed("result") {
				request.Result, err = parseJSONFlag("result", resultRaw)
				if err != nil {
					return err
				}
			}

			run, err := client.CompleteTaskRun(cmd.Context(), args[0], request)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, taskRunBundle(run))
		},
	}
	cmd.Flags().StringVar(&resultRaw, "result", "", "Optional result JSON")
	return cmd
}

func newTaskRunFailCommand(deps commandDeps) *cobra.Command {
	var (
		errorMessage string
		metadataRaw  string
	)

	cmd := &cobra.Command{
		Use:   "fail <run-id>",
		Short: "Fail a task run",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			request := FailTaskRunRequest{Error: strings.TrimSpace(errorMessage)}
			if request.Error == "" {
				return errors.New("cli: --error is required")
			}
			if cmd.Flags().Changed("metadata") {
				request.Metadata, err = parseJSONFlag("metadata", metadataRaw)
				if err != nil {
					return err
				}
			}

			run, err := client.FailTaskRun(cmd.Context(), args[0], request)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, taskRunBundle(run))
		},
	}
	cmd.Flags().StringVar(&errorMessage, taskErrorKey, "", "Failure message")
	cmd.Flags().StringVar(&metadataRaw, "metadata", "", "Optional failure metadata JSON")
	mustMarkFlagRequired(cmd, taskErrorKey)
	return cmd
}

func newTaskRunCancelCommand(deps commandDeps) *cobra.Command {
	var (
		reason      string
		metadataRaw string
	)

	cmd := &cobra.Command{
		Use:   "cancel <run-id>",
		Short: "Cancel a task run",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			request := CancelTaskRunRequest{Reason: strings.TrimSpace(reason)}
			if cmd.Flags().Changed("metadata") {
				request.Metadata, err = parseJSONFlag("metadata", metadataRaw)
				if err != nil {
					return err
				}
			}

			run, err := client.CancelTaskRun(cmd.Context(), args[0], request)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, taskRunBundle(run))
		},
	}
	cmd.Flags().StringVar(&reason, "reason", "", "Optional cancellation reason")
	cmd.Flags().StringVar(&metadataRaw, "metadata", "", "Optional cancellation metadata JSON")
	return cmd
}

func parseTaskListFilters(
	scopeRaw string,
	workspaceRef string,
	statusRaw string,
	ownerKindRaw string,
	ownerRef string,
	parentTaskID string,
	channelRaw string,
	last int,
) (TaskListQuery, error) {
	scope, workspace, err := resolveTaskScopeWorkspace(scopeRaw, workspaceRef, false)
	if err != nil {
		return TaskListQuery{}, err
	}
	status, err := parseOptionalTaskStatus(statusRaw)
	if err != nil {
		return TaskListQuery{}, err
	}
	ownerKind, err := parseOptionalTaskOwnerKind(ownerKindRaw)
	if err != nil {
		return TaskListQuery{}, err
	}
	trimmedOwnerRef := strings.TrimSpace(ownerRef)
	if (ownerKind != "" && trimmedOwnerRef == "") || (ownerKind == "" && trimmedOwnerRef != "") {
		return TaskListQuery{}, errors.New(
			"cli: --owner-kind and --owner-ref must be provided together",
		)
	}
	if err := validateTaskChannelFlag(channelRaw); err != nil {
		return TaskListQuery{}, err
	}
	if err := validateTaskLast(last); err != nil {
		return TaskListQuery{}, err
	}

	return TaskListQuery{
		Scope:          scope,
		Workspace:      workspace,
		Status:         status,
		OwnerKind:      ownerKind,
		OwnerRef:       trimmedOwnerRef,
		ParentTaskID:   strings.TrimSpace(parentTaskID),
		NetworkChannel: strings.TrimSpace(channelRaw),
		Limit:          last,
	}, nil
}

func parseTaskRunListFilters(
	statusRaw string,
	sessionID string,
	last int,
) (TaskRunListQuery, error) {
	status, err := parseOptionalTaskRunStatus(statusRaw)
	if err != nil {
		return TaskRunListQuery{}, err
	}
	if err := validateTaskLast(last); err != nil {
		return TaskRunListQuery{}, err
	}
	return TaskRunListQuery{
		Status:    status,
		SessionID: strings.TrimSpace(sessionID),
		Limit:     last,
	}, nil
}

func parseOptionalReviewPolicy(raw string) (taskpkg.ReviewPolicy, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}
	policy := taskpkg.ReviewPolicy(trimmed).Normalize()
	if err := policy.Validate("review_policy"); err != nil {
		return "", fmt.Errorf("cli: %w", err)
	}
	return policy, nil
}

func parseOptionalReviewStatus(raw string) (taskpkg.RunReviewStatus, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}
	status := taskpkg.RunReviewStatus(trimmed).Normalize()
	if err := status.Validate("review_status"); err != nil {
		return "", fmt.Errorf("cli: %w", err)
	}
	return status, nil
}

func parseRequiredReviewOutcome(raw string) (taskpkg.RunReviewOutcome, error) {
	outcome := taskpkg.RunReviewOutcome(strings.TrimSpace(raw)).Normalize()
	if outcome == "" {
		return "", errors.New("cli: --outcome is required")
	}
	if err := outcome.Validate("review_outcome"); err != nil {
		return "", fmt.Errorf("cli: %w", err)
	}
	return outcome, nil
}

func missingWorkFromFlags(items []string, raw string) (json.RawMessage, error) {
	hasRaw := strings.TrimSpace(raw) != ""
	if hasRaw && len(items) > 0 {
		return nil, errors.New("cli: --missing-work-json cannot be combined with --missing-work")
	}
	if hasRaw {
		payload, err := parseJSONFlag("missing-work-json", raw)
		if err != nil {
			return nil, err
		}
		var items []json.RawMessage
		if err := json.Unmarshal(payload, &items); err != nil {
			return nil, errors.New("cli: --missing-work-json must be a JSON array")
		}
		return payload, nil
	}

	normalized := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}
	if len(normalized) == 0 {
		return nil, nil
	}
	payload, err := json.Marshal(normalized)
	if err != nil {
		return nil, fmt.Errorf("cli: encode --missing-work: %w", err)
	}
	return payload, nil
}

func resolveTaskScopeWorkspace(
	rawScope string,
	workspaceRef string,
	scopeRequired bool,
) (taskpkg.Scope, string, error) {
	scope, err := parseOptionalTaskScope(rawScope)
	if err != nil {
		return "", "", err
	}
	if scopeRequired && scope == "" {
		return "", "", errors.New("cli: --scope is required")
	}

	workspace := strings.TrimSpace(workspaceRef)
	switch scope.Normalize() {
	case taskpkg.ScopeGlobal:
		if workspace != "" {
			return "", "", errors.New("cli: --workspace must be empty when --scope is global")
		}
	case taskpkg.ScopeWorkspace:
		if workspace == "" {
			return "", "", errors.New("cli: --workspace is required when --scope is workspace")
		}
	}
	return scope, workspace, nil
}

func parseOptionalTaskScope(raw string) (taskpkg.Scope, error) {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	if trimmed == "" {
		return "", nil
	}
	scope := taskpkg.Scope(trimmed)
	if err := scope.Validate(taskScopeKey); err != nil {
		return "", fmt.Errorf("cli: %w", err)
	}
	return scope, nil
}

func parseOptionalTaskStatus(raw string) (taskpkg.Status, error) {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	if trimmed == "" {
		return "", nil
	}
	status := taskpkg.Status(trimmed)
	if err := status.Validate(taskStatusKey); err != nil {
		return "", fmt.Errorf("cli: %w", err)
	}
	return status, nil
}

func parseOptionalTaskRunStatus(raw string) (taskpkg.RunStatus, error) {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	if trimmed == "" {
		return "", nil
	}
	status := taskpkg.RunStatus(trimmed)
	if err := status.Validate(taskStatusKey); err != nil {
		return "", fmt.Errorf("cli: %w", err)
	}
	return status, nil
}

func parseOptionalTaskOwnerKind(raw string) (taskpkg.OwnerKind, error) {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	if trimmed == "" {
		return "", nil
	}
	kind := taskpkg.OwnerKind(trimmed)
	if err := kind.Validate("owner_kind"); err != nil {
		return "", fmt.Errorf("cli: %w", err)
	}
	return kind, nil
}

func parseOptionalTaskDependencyKind(raw string) (taskpkg.DependencyKind, error) {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	if trimmed == "" {
		return "", nil
	}
	kind := taskpkg.DependencyKind(trimmed)
	if err := kind.Validate(taskKindKey); err != nil {
		return "", fmt.Errorf("cli: %w", err)
	}
	return kind, nil
}

func parseRequiredTaskOwnership(kindRaw string, refRaw string) (*taskpkg.Ownership, error) {
	kindText := strings.TrimSpace(kindRaw)
	ref := strings.TrimSpace(refRaw)
	if kindText == "" || ref == "" {
		return nil, errors.New("cli: --owner-kind and --owner-ref must be provided together")
	}
	kind, err := parseOptionalTaskOwnerKind(kindText)
	if err != nil {
		return nil, err
	}
	owner := &taskpkg.Ownership{Kind: kind, Ref: ref}
	if err := owner.Validate(taskOwnerKey); err != nil {
		return nil, fmt.Errorf("cli: %w", err)
	}
	return owner, nil
}

func parseJSONFlag(flagName string, raw string) (json.RawMessage, error) {
	payload, err := parseRequiredJSONRawMessage(raw)
	if errors.Is(err, errEmptyJSONFlag) {
		return nil, fmt.Errorf("cli: --%s requires valid JSON", flagName)
	}
	if err != nil {
		return nil, fmt.Errorf("cli: invalid --%s JSON: %w", flagName, err)
	}
	return payload, nil
}

func parseAgentTaskJSONFlag(flagName string, raw string) (json.RawMessage, error) {
	payload, err := parseJSONFlag(flagName, raw)
	if err != nil {
		return nil, err
	}
	if err := contract.ValidateNoRawClaimTokenField(payload); err != nil {
		return nil, fmt.Errorf("cli: --%s must not contain raw lease credential fields: %w", flagName, err)
	}
	return payload, nil
}

func requiredAgentTaskRunID(rawRunID string) (string, error) {
	runID := strings.TrimSpace(rawRunID)
	if runID == "" {
		return "", errors.New("cli: run id is required")
	}
	return runID, nil
}

func validateAgentTaskLeaseSeconds(seconds int64) error {
	if seconds < 0 {
		return fmt.Errorf("cli: --lease-seconds must be zero or positive: %d", seconds)
	}
	maxSeconds := int64(taskpkg.MaxRunLeaseDuration.Seconds())
	if seconds > maxSeconds {
		return fmt.Errorf("cli: --lease-seconds must be less than or equal to %d", maxSeconds)
	}
	return nil
}

func trimAgentTaskCapabilities(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	trimmed := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		trimmed = append(trimmed, value)
	}
	if len(trimmed) == 0 {
		return nil
	}
	return trimmed
}

func buildTaskCreateRequest(cmd *cobra.Command, input taskCreateInput) (CreateTaskRequest, error) {
	scope, workspace, err := resolveTaskScopeWorkspace(input.ScopeRaw, input.WorkspaceRef, true)
	if err != nil {
		return CreateTaskRequest{}, err
	}
	if err := validateTaskChannelFlag(input.NetworkRaw); err != nil {
		return CreateTaskRequest{}, err
	}

	owner, err := parseOptionalTaskOwnership(cmd, input.OwnerKindRaw, input.OwnerRef)
	if err != nil {
		return CreateTaskRequest{}, err
	}
	metadata, err := parseOptionalTaskMetadata(cmd, input.MetadataRaw)
	if err != nil {
		return CreateTaskRequest{}, err
	}
	priority, err := parseOptionalChangedTaskPriority(cmd, input.PriorityRaw)
	if err != nil {
		return CreateTaskRequest{}, err
	}

	request := CreateTaskRequest{
		ID:             strings.TrimSpace(input.ID),
		Identifier:     strings.TrimSpace(input.Identifier),
		Scope:          scope,
		Workspace:      workspace,
		NetworkChannel: strings.TrimSpace(input.NetworkRaw),
		Title:          strings.TrimSpace(input.Title),
		Description:    strings.TrimSpace(input.Description),
		Priority:       priority,
		Owner:          owner,
		Metadata:       metadata,
	}
	if request.Title == "" {
		return CreateTaskRequest{}, errors.New("cli: --title is required")
	}
	return request, nil
}

func parseOptionalTaskOwnership(
	cmd *cobra.Command,
	ownerKindRaw string,
	ownerRef string,
) (*taskpkg.Ownership, error) {
	if !cmd.Flags().Changed("owner-kind") && !cmd.Flags().Changed("owner-ref") {
		return nil, nil
	}
	return parseRequiredTaskOwnership(ownerKindRaw, ownerRef)
}

func parseOptionalTaskMetadata(cmd *cobra.Command, metadataRaw string) (json.RawMessage, error) {
	if !cmd.Flags().Changed("metadata") {
		return nil, nil
	}
	return parseJSONFlag("metadata", metadataRaw)
}

func parseOptionalChangedTaskPriority(cmd *cobra.Command, raw string) (taskpkg.Priority, error) {
	if !cmd.Flags().Changed("priority") {
		return "", nil
	}
	return parseOptionalTaskPriority(raw)
}

func parseOptionalTaskPriority(raw string) (taskpkg.Priority, error) {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	if trimmed == "" {
		return "", errors.New("cli: --priority cannot be blank")
	}
	priority := taskpkg.Priority(trimmed)
	if err := priority.Validate("priority"); err != nil {
		return "", fmt.Errorf("cli: %w", err)
	}
	return priority, nil
}

func validateTaskChannelFlag(channel string) error {
	trimmed := strings.TrimSpace(channel)
	if trimmed == "" {
		return nil
	}
	if err := network.ValidateChannel(trimmed); err != nil {
		return fmt.Errorf("cli: invalid --channel value %q: %w", trimmed, err)
	}
	return nil
}

func validateTaskLast(last int) error {
	if last < 0 {
		return fmt.Errorf("cli: --last must be zero or positive: %d", last)
	}
	return nil
}

func agentTaskNextBundle(record AgentTaskNextRecord) outputBundle {
	return outputBundle{
		jsonValue: record,
		human: func() (string, error) {
			return renderJSONPreview(record)
		},
		toon: func() (string, error) {
			return renderJSONPreview(record)
		},
	}
}

func agentTaskLeaseBundle(record AgentTaskLeaseRecord) outputBundle {
	return outputBundle{
		jsonValue: record,
		human: func() (string, error) {
			return renderJSONPreview(record)
		},
		toon: func() (string, error) {
			return renderJSONPreview(record)
		},
	}
}

func taskBundle(item TaskRecord) outputBundle {
	return outputBundle{
		jsonValue: item,
		human: func() (string, error) {
			return renderHumanSection(taskTaskValue, []keyValue{
				{Label: "ID", Value: stringOrDash(item.ID)},
				{Label: taskIdentifierValue, Value: stringOrDash(item.Identifier)},
				{Label: taskScopeValue, Value: stringOrDash(string(item.Scope))},
				{Label: taskWorkspaceValue, Value: stringOrDash(item.WorkspaceID)},
				{Label: taskParentValue, Value: stringOrDash(item.ParentTaskID)},
				{Label: taskTitleValue, Value: stringOrDash(item.Title)},
				{Label: taskDescriptionValue, Value: stringOrDash(item.Description)},
				{Label: taskStatusValue, Value: stringOrDash(string(item.Status))},
				{Label: taskOwnerValue, Value: stringOrDash(formatTaskOwnership(item.Owner))},
				{Label: taskCreatedByValue, Value: stringOrDash(formatTaskActor(item.CreatedBy))},
				{Label: taskOriginValue, Value: stringOrDash(formatTaskOrigin(item.Origin))},
				{Label: taskChannelValue, Value: stringOrDash(item.NetworkChannel)},
				{Label: taskCreatedValue, Value: stringOrDash(formatTime(item.CreatedAt))},
				{Label: taskUpdatedValue, Value: stringOrDash(formatTime(item.UpdatedAt))},
				{Label: "Closed", Value: stringOrDash(formatTimePtr(item.ClosedAt))},
				{Label: "Metadata", Value: stringOrDash(compactJSON(item.Metadata))},
			}), nil
		},
		toon: func() (string, error) {
			return renderToonObject(taskTaskKey, []string{
				"id",
				taskIdentifierKey,
				taskScopeKey,
				taskWorkspaceIDKey,
				"parent_task_id",
				taskTitleKey,
				taskDescriptionKey,
				taskStatusKey,
				taskOwnerKey,
				"created_by",
				taskOriginKey,
				taskNetworkChannelKey,
				taskCreatedAtKey,
				taskUpdatedAtKey,
				"closed_at",
				"metadata",
			}, []string{
				item.ID,
				item.Identifier,
				string(item.Scope),
				item.WorkspaceID,
				item.ParentTaskID,
				item.Title,
				item.Description,
				string(item.Status),
				formatTaskOwnership(item.Owner),
				formatTaskActor(item.CreatedBy),
				formatTaskOrigin(item.Origin),
				item.NetworkChannel,
				formatTime(item.CreatedAt),
				formatTime(item.UpdatedAt),
				formatTimePtr(item.ClosedAt),
				compactJSON(item.Metadata),
			}), nil
		},
	}
}

func taskExecutionProfileBundle(profile *TaskExecutionProfileRecord) outputBundle {
	return outputBundle{
		jsonValue: *profile,
		human: func() (string, error) {
			return renderHumanSection("Task Execution Profile", []keyValue{
				{Label: taskTaskIDValue, Value: stringOrDash(profile.TaskID)},
				{Label: "Coordinator", Value: stringOrDash(string(profile.Coordinator.Mode))},
				{Label: "Worker", Value: stringOrDash(string(profile.Worker.Mode))},
				{Label: "Worker Agent", Value: stringOrDash(profile.Worker.AgentName)},
				{Label: "Worker Provider", Value: stringOrDash(profile.Worker.Provider)},
				{Label: "Worker Model", Value: stringOrDash(profile.Worker.Model)},
				{Label: "Review Agent", Value: stringOrDash(profile.Review.AgentName)},
				{Label: taskSandboxValue, Value: stringOrDash(string(profile.Sandbox.Mode))},
				{Label: "Sandbox Ref", Value: stringOrDash(profile.Sandbox.SandboxRef)},
				{Label: taskUpdatedValue, Value: stringOrDash(formatTime(profile.UpdatedAt))},
			}), nil
		},
		toon: func() (string, error) {
			return renderJSONPreview(profile)
		},
	}
}

func taskBridgeNotificationSubscriptionBundle(
	subscription *TaskBridgeNotificationSubscriptionRecord,
) outputBundle {
	return outputBundle{
		jsonValue: *subscription,
		human: func() (string, error) {
			return renderHumanSection(
				"Task Bridge Notification Subscription",
				taskBridgeNotificationRows(subscription),
			), nil
		},
		toon: func() (string, error) {
			return renderJSONPreview(subscription)
		},
	}
}

func taskBridgeNotificationSubscriptionListBundle(
	items []TaskBridgeNotificationSubscriptionRecord,
) outputBundle {
	return listBundle(
		items,
		items,
		"Task Bridge Notification Subscriptions",
		[]string{
			taskSubscriptionValue,
			taskTaskValue,
			taskBridgeValue,
			taskScopeValue,
			taskPeerValue,
			taskGroupValue,
			taskModeValue,
			"Cursor Seq",
			"Cursor Error",
			"Cursor Updated",
			taskUpdatedValue,
		},
		"task_bridge_notification_subscriptions",
		[]string{
			"subscription_id",
			taskTaskIDKey,
			taskBridgeInstanceIDKey,
			taskScopeKey,
			taskPeerIDKey,
			taskGroupIDKey,
			"delivery_mode",
			"cursor_last_sequence",
			"cursor_last_error",
			"cursor_updated_at",
			taskUpdatedAtKey,
		},
		func(item TaskBridgeNotificationSubscriptionRecord) []string {
			return []string{
				stringOrDash(item.SubscriptionID),
				stringOrDash(item.TaskID),
				stringOrDash(item.BridgeInstanceID),
				stringOrDash(string(item.Scope)),
				stringOrDash(item.PeerID),
				stringOrDash(item.GroupID),
				stringOrDash(string(item.DeliveryMode)),
				int64OrDash(item.Cursor.LastSequence),
				stringOrDash(item.Cursor.LastError),
				stringOrDash(formatTimePtr(item.Cursor.UpdatedAt)),
				stringOrDash(formatTime(item.UpdatedAt)),
			}
		},
		func(item TaskBridgeNotificationSubscriptionRecord) []string {
			return []string{
				item.SubscriptionID,
				item.TaskID,
				item.BridgeInstanceID,
				string(item.Scope),
				item.PeerID,
				item.GroupID,
				string(item.DeliveryMode),
				strconv.FormatInt(item.Cursor.LastSequence, 10),
				item.Cursor.LastError,
				formatTimePtr(item.Cursor.UpdatedAt),
				formatTime(item.UpdatedAt),
			}
		},
	)
}

func taskBridgeNotificationRows(subscription *TaskBridgeNotificationSubscriptionRecord) []keyValue {
	return []keyValue{
		{Label: taskSubscriptionValue, Value: stringOrDash(subscription.SubscriptionID)},
		{Label: taskTaskValue, Value: stringOrDash(subscription.TaskID)},
		{Label: taskBridgeValue, Value: stringOrDash(subscription.BridgeInstanceID)},
		{Label: taskScopeValue, Value: stringOrDash(string(subscription.Scope))},
		{Label: taskWorkspaceValue, Value: stringOrDash(subscription.WorkspaceID)},
		{Label: taskPeerValue, Value: stringOrDash(subscription.PeerID)},
		{Label: taskThreadValue, Value: stringOrDash(subscription.ThreadID)},
		{Label: taskGroupValue, Value: stringOrDash(subscription.GroupID)},
		{Label: taskModeValue, Value: stringOrDash(string(subscription.DeliveryMode))},
		{Label: "Cursor Consumer", Value: stringOrDash(subscription.Cursor.ConsumerID)},
		{Label: "Cursor Stream", Value: stringOrDash(subscription.Cursor.StreamName)},
		{Label: "Cursor Subject", Value: stringOrDash(subscription.Cursor.SubjectID)},
		{Label: "Cursor Last Sequence", Value: int64OrDash(subscription.Cursor.LastSequence)},
		{Label: "Cursor Last Delivery", Value: stringOrDash(subscription.Cursor.LastDeliveryID)},
		{Label: "Cursor Last Delivered", Value: stringOrDash(formatTimePtr(subscription.Cursor.LastDeliveredAt))},
		{Label: "Cursor Last Error", Value: stringOrDash(subscription.Cursor.LastError)},
		{Label: "Cursor Updated", Value: stringOrDash(formatTimePtr(subscription.Cursor.UpdatedAt))},
		{Label: taskCreatedByValue, Value: stringOrDash(formatTaskActor(subscription.CreatedBy))},
		{Label: taskUpdatedValue, Value: stringOrDash(formatTime(subscription.UpdatedAt))},
	}
}

func taskBridgeNotificationSubscriptionDeleteBundle(taskID string, subscriptionID string) outputBundle {
	item := struct {
		TaskID         string `json:"task_id"`
		SubscriptionID string `json:"subscription_id"`
		Status         string `json:"status"`
	}{
		TaskID:         strings.TrimSpace(taskID),
		SubscriptionID: strings.TrimSpace(subscriptionID),
		Status:         taskDeletedKey,
	}
	return outputBundle{
		jsonValue: item,
		human: func() (string, error) {
			return renderHumanSection("Task Bridge Notification Subscription", []keyValue{
				{Label: taskTaskIDValue, Value: stringOrDash(item.TaskID)},
				{Label: taskSubscriptionValue, Value: stringOrDash(item.SubscriptionID)},
				{Label: taskStatusValue, Value: item.Status},
			}), nil
		},
		toon: func() (string, error) {
			return renderToonObject(
				"task_bridge_notification_subscription",
				[]string{taskTaskIDKey, "subscription_id", taskStatusKey},
				[]string{item.TaskID, item.SubscriptionID, item.Status},
			), nil
		},
	}
}

func taskRunReviewRequestBundle(record *TaskRunReviewRequestRecord) outputBundle {
	return outputBundle{
		jsonValue: *record,
		human: func() (string, error) {
			return renderHumanSection("Task Run Review Request", []keyValue{
				{Label: taskReviewValue, Value: stringOrDash(record.Review.ReviewID)},
				{Label: taskRunValue, Value: stringOrDash(record.Review.RunID)},
				{Label: taskTaskValue, Value: stringOrDash(record.Review.TaskID)},
				{Label: taskStatusValue, Value: stringOrDash(string(record.Review.Status))},
				{Label: "Policy", Value: stringOrDash(string(record.Review.Policy))},
				{Label: taskCreatedValue, Value: strconv.FormatBool(record.Created)},
			}), nil
		},
		toon: func() (string, error) {
			return renderJSONPreview(record)
		},
	}
}

func taskRunReviewBundle(review *TaskRunReviewRecord) outputBundle {
	return outputBundle{
		jsonValue: *review,
		human: func() (string, error) {
			return renderHumanSection("Task Run Review", taskRunReviewRows(review)), nil
		},
		toon: func() (string, error) {
			return renderJSONPreview(review)
		},
	}
}

func taskRunReviewVerdictBundle(record *TaskRunReviewVerdictRecord) outputBundle {
	return outputBundle{
		jsonValue: *record,
		human: func() (string, error) {
			rows := taskRunReviewRows(&record.Review)
			if record.ContinuationRun != nil {
				rows = append(rows, keyValue{Label: "Continuation Run", Value: stringOrDash(record.ContinuationRun.ID)})
			}
			rows = append(rows, keyValue{Label: "Circuit Opened", Value: strconv.FormatBool(record.CircuitOpened)})
			return renderHumanSection("Task Run Review Verdict", rows), nil
		},
		toon: func() (string, error) {
			return renderJSONPreview(record)
		},
	}
}

func taskRunReviewRows(review *TaskRunReviewRecord) []keyValue {
	return []keyValue{
		{Label: taskReviewValue, Value: stringOrDash(review.ReviewID)},
		{Label: taskTaskValue, Value: stringOrDash(review.TaskID)},
		{Label: taskRunValue, Value: stringOrDash(review.RunID)},
		{Label: taskStatusValue, Value: stringOrDash(string(review.Status))},
		{Label: taskOutcomeValue, Value: stringOrDash(string(review.Outcome))},
		{Label: taskReasonValue, Value: stringOrDash(review.Reason)},
		{Label: "Delivery", Value: stringOrDash(review.DeliveryID)},
		{Label: "Missing Work", Value: stringOrDash(compactJSON(review.MissingWork))},
		{Label: "Next Guidance", Value: stringOrDash(review.NextRoundGuidance)},
		{Label: "Reviewer Session", Value: stringOrDash(review.ReviewerSessionID)},
		{Label: "Reviewed By", Value: stringOrDash(formatTaskActorPtr(review.ReviewedBy))},
		{Label: "Requested", Value: stringOrDash(formatTime(review.RequestedAt))},
		{Label: "Reviewed", Value: stringOrDash(formatTime(review.ReviewedAt))},
		{Label: taskUpdatedValue, Value: stringOrDash(formatTime(review.UpdatedAt))},
	}
}

func taskRunReviewListBundle(items []TaskRunReviewRecord) outputBundle {
	return listBundle(
		items,
		items,
		"Task Run Reviews",
		[]string{
			taskReviewValue,
			taskTaskValue,
			taskRunValue,
			taskStatusValue,
			taskOutcomeValue,
			"Reviewer Session",
			taskUpdatedValue,
		},
		"task_run_reviews",
		[]string{
			"review_id",
			taskTaskIDKey,
			taskRunIDKey,
			taskStatusKey,
			cliOutcomeKey,
			"reviewer_session_id",
			taskUpdatedAtKey,
		},
		func(item TaskRunReviewRecord) []string {
			return []string{
				stringOrDash(item.ReviewID),
				stringOrDash(item.TaskID),
				stringOrDash(item.RunID),
				stringOrDash(string(item.Status)),
				stringOrDash(string(item.Outcome)),
				stringOrDash(item.ReviewerSessionID),
				stringOrDash(formatTime(item.UpdatedAt)),
			}
		},
		func(item TaskRunReviewRecord) []string {
			return []string{
				item.ReviewID,
				item.TaskID,
				item.RunID,
				string(item.Status),
				string(item.Outcome),
				item.ReviewerSessionID,
				formatTime(item.UpdatedAt),
			}
		},
	)
}

func taskExecutionBundle(item *TaskExecutionRecord) outputBundle {
	return outputBundle{
		jsonValue: *item,
		human: func() (string, error) {
			taskBlock, err := taskBundle(item.Task).human()
			if err != nil {
				return "", err
			}
			runBlock, err := taskRunBundle(item.Run).human()
			if err != nil {
				return "", err
			}
			return renderHumanBlocks(taskBlock, runBlock), nil
		},
		toon: func() (string, error) {
			taskBlock, err := taskBundle(item.Task).toon()
			if err != nil {
				return "", err
			}
			runBlock, err := taskRunBundle(item.Run).toon()
			if err != nil {
				return "", err
			}
			return renderHumanBlocks(taskBlock, runBlock), nil
		},
	}
}

func taskDeleteBundle(id string) outputBundle {
	item := struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}{
		ID:     strings.TrimSpace(id),
		Status: taskDeletedKey,
	}
	return outputBundle{
		jsonValue: item,
		human: func() (string, error) {
			return renderHumanSection(taskTaskValue, []keyValue{
				{Label: "ID", Value: stringOrDash(item.ID)},
				{Label: taskStatusValue, Value: item.Status},
			}), nil
		},
		toon: func() (string, error) {
			return renderToonObject(taskTaskKey, []string{"id", taskStatusKey}, []string{item.ID, item.Status}), nil
		},
	}
}

func taskExecutionProfileDeleteBundle(id string) outputBundle {
	item := struct {
		TaskID string `json:"task_id"`
		Status string `json:"status"`
	}{
		TaskID: strings.TrimSpace(id),
		Status: taskDeletedKey,
	}
	return outputBundle{
		jsonValue: item,
		human: func() (string, error) {
			return renderHumanSection("Task Execution Profile", []keyValue{
				{Label: taskTaskIDValue, Value: stringOrDash(item.TaskID)},
				{Label: taskStatusValue, Value: item.Status},
			}), nil
		},
		toon: func() (string, error) {
			return renderToonObject(
				"task_execution_profile",
				[]string{taskTaskIDKey, taskStatusKey},
				[]string{item.TaskID, item.Status},
			), nil
		},
	}
}

func taskSummaryListBundle(items []TaskSummaryRecord) outputBundle {
	return listBundle(
		items,
		items,
		"Tasks",
		[]string{
			"ID",
			taskIdentifierValue,
			taskScopeValue,
			taskWorkspaceValue,
			taskParentValue,
			taskStatusValue,
			taskOwnerValue,
			taskChannelValue,
			taskTitleValue,
		},
		"tasks",
		[]string{
			"id",
			taskIdentifierKey,
			taskScopeKey,
			taskWorkspaceIDKey,
			"parent_task_id",
			taskStatusKey,
			taskOwnerKey,
			taskNetworkChannelKey,
			taskTitleKey,
		},
		func(item TaskSummaryRecord) []string {
			return []string{
				stringOrDash(item.ID),
				stringOrDash(item.Identifier),
				stringOrDash(string(item.Scope)),
				stringOrDash(item.WorkspaceID),
				stringOrDash(item.ParentTaskID),
				stringOrDash(string(item.Status)),
				stringOrDash(formatTaskOwnership(item.Owner)),
				stringOrDash(item.NetworkChannel),
				stringOrDash(item.Title),
			}
		},
		func(item TaskSummaryRecord) []string {
			return []string{
				item.ID,
				item.Identifier,
				string(item.Scope),
				item.WorkspaceID,
				item.ParentTaskID,
				string(item.Status),
				formatTaskOwnership(item.Owner),
				item.NetworkChannel,
				item.Title,
			}
		},
	)
}

func taskDetailBundle(detail *TaskDetailRecord) outputBundle {
	return outputBundle{
		jsonValue: detail,
		human:     func() (string, error) { return renderTaskDetailHuman(detail) },
		toon:      func() (string, error) { return renderTaskDetailToon(detail) },
	}
}

func taskInspectBundle(record *TaskInspectRecord) outputBundle {
	return outputBundle{
		jsonValue: record,
		human:     func() (string, error) { return renderTaskInspectHuman(record) },
		toon:      func() (string, error) { return renderTaskInspectToon(record) },
	}
}

func renderTaskInspectHuman(record *TaskInspectRecord) (string, error) {
	blocks := []string{
		renderHumanSection("Task Inspect", []keyValue{
			{Label: "Target", Value: stringOrDash(record.Target)},
			{Label: taskTaskValue, Value: stringOrDash(record.Task.ID)},
			{Label: taskTitleValue, Value: stringOrDash(record.Task.Title)},
			{Label: taskStatusValue, Value: stringOrDash(string(record.Task.Status))},
			{Label: "Current Run", Value: stringOrDash(taskInspectCurrentRunID(record.CurrentRun))},
			{Label: cliNextActionValue, Value: stringOrDash(record.NextAction)},
			{Label: "As Of", Value: stringOrDash(formatTime(record.AsOf))},
		}),
	}
	if record.CurrentRun != nil {
		blocks = append(blocks, renderHumanSection("Current Run", taskInspectRunSectionRows(record.CurrentRun)))
	}
	if record.BoundSession != nil {
		blocks = append(blocks, renderHumanSection("Bound Session", taskInspectSessionRows(record.BoundSession)))
	}
	blocks = append(
		blocks,
		renderHumanTable(
			"Diagnostics",
			[]string{cliCodeValue, cliSeverityValue, "Message", cliCommandValue},
			taskInspectDiagnosticRows(record.Diagnostics),
		),
		renderHumanTable(
			"Recent Runs",
			[]string{
				taskRunValue,
				taskStatusValue,
				taskAttemptValue,
				taskSessionValue,
				cliHashValue,
				"Heartbeat Age",
				taskErrorValue,
			},
			taskInspectRunRows(record.RecentRuns),
		),
		renderHumanTable(
			"Recent Events",
			[]string{"ID", taskTypeValue, taskRunValue, taskOutcomeValue, authoredContextSummaryValue, taskTimeValue},
			taskInspectEventRows(record.RecentEvents),
		),
	)
	return renderHumanBlocks(blocks...), nil
}

func renderTaskInspectToon(record *TaskInspectRecord) (string, error) {
	blocks := []string{
		renderToonObject("task_inspect", []string{
			"target",
			taskTaskIDKey,
			taskTitleKey,
			taskStatusKey,
			"current_run_id",
			"next_action",
			"as_of",
		}, []string{
			record.Target,
			record.Task.ID,
			record.Task.Title,
			string(record.Task.Status),
			taskInspectCurrentRunID(record.CurrentRun),
			record.NextAction,
			formatTime(record.AsOf),
		}),
		renderToonArray(
			"diagnostics",
			[]string{"code", "severity", "message", "command"},
			taskInspectDiagnosticToonRows(record.Diagnostics),
		),
		renderToonArray(
			"recent_runs",
			[]string{
				taskRunIDKey,
				taskStatusKey,
				taskAttemptKey,
				taskSessionIDKey,
				"hash",
				"heartbeat_age_seconds",
				taskErrorKey,
			},
			taskInspectRunToonRows(record.RecentRuns),
		),
		renderToonArray(
			"recent_events",
			[]string{"id", extensionTypeKey, taskRunIDKey, taskOutcomeKey, memorySummaryKey, taskTimestampKey},
			taskInspectEventToonRows(record.RecentEvents),
		),
	}
	return renderHumanBlocks(blocks...), nil
}

func taskInspectRunSectionRows(run *contract.TaskInspectRunPayload) []keyValue {
	return []keyValue{
		{Label: taskRunValue, Value: stringOrDash(run.RunID)},
		{Label: taskTaskValue, Value: stringOrDash(run.TaskID)},
		{Label: taskStatusValue, Value: stringOrDash(string(run.Status))},
		{Label: taskAttemptValue, Value: intOrDash(run.Attempt)},
		{Label: taskSessionValue, Value: stringOrDash(run.BoundSessionID)},
		{Label: "Claim Hash", Value: stringOrDash(run.ClaimTokenHashTruncated)},
		{Label: "Lease Until", Value: stringOrDash(formatTimePtr(run.LeaseUntil))},
		{Label: "Heartbeat Age", Value: stringOrDash(taskInspectHeartbeatAge(run.HeartbeatAgeSeconds))},
		{Label: taskErrorValue, Value: stringOrDash(run.LastErrorSummary)},
	}
}

func taskInspectSessionRows(session *contract.TaskInspectSessionPayload) []keyValue {
	return []keyValue{
		{Label: taskSessionValue, Value: stringOrDash(session.SessionID)},
		{Label: taskStatusValue, Value: stringOrDash(session.State)},
		{Label: "Agent", Value: stringOrDash(session.AgentName)},
		{Label: installProviderValue, Value: stringOrDash(session.ProviderName)},
		{Label: taskWorkspaceValue, Value: stringOrDash(session.WorkspaceID)},
		{Label: taskStartedValue, Value: stringOrDash(formatTimePtr(session.StartedAt))},
		{Label: "Last Activity", Value: stringOrDash(formatTimePtr(session.LastActivityAt))},
		{Label: taskReasonValue, Value: stringOrDash(session.StopReason)},
		{Label: "Failure", Value: stringOrDash(session.FailureKind)},
	}
}

func taskInspectDiagnosticRows(items []contract.DiagnosticItem) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			stringOrDash(item.Code),
			stringOrDash(item.Severity),
			stringOrDash(item.Message),
			stringOrDash(item.SuggestedCommand),
		})
	}
	return rows
}

func taskInspectDiagnosticToonRows(items []contract.DiagnosticItem) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{item.Code, item.Severity, item.Message, item.SuggestedCommand})
	}
	return rows
}

func taskInspectRunRows(items []contract.TaskInspectRunPayload) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			stringOrDash(item.RunID),
			stringOrDash(string(item.Status)),
			intOrDash(item.Attempt),
			stringOrDash(item.BoundSessionID),
			stringOrDash(item.ClaimTokenHashTruncated),
			stringOrDash(taskInspectHeartbeatAge(item.HeartbeatAgeSeconds)),
			stringOrDash(item.LastErrorSummary),
		})
	}
	return rows
}

func taskInspectRunToonRows(items []contract.TaskInspectRunPayload) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.RunID,
			string(item.Status),
			strconv.Itoa(item.Attempt),
			item.BoundSessionID,
			item.ClaimTokenHashTruncated,
			taskInspectHeartbeatAge(item.HeartbeatAgeSeconds),
			item.LastErrorSummary,
		})
	}
	return rows
}

func taskInspectEventRows(items []contract.TaskInspectEventPayload) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			stringOrDash(item.ID),
			stringOrDash(item.Type),
			stringOrDash(item.RunID),
			stringOrDash(item.Outcome),
			stringOrDash(item.Summary),
			stringOrDash(formatTime(item.Timestamp)),
		})
	}
	return rows
}

func taskInspectEventToonRows(items []contract.TaskInspectEventPayload) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.ID,
			item.Type,
			item.RunID,
			item.Outcome,
			item.Summary,
			formatTime(item.Timestamp),
		})
	}
	return rows
}

func taskInspectCurrentRunID(run *contract.TaskInspectRunPayload) string {
	if run == nil {
		return ""
	}
	return run.RunID
}

func taskInspectHeartbeatAge(value *int64) string {
	if value == nil {
		return ""
	}
	return int64OrDash(*value)
}

func renderTaskDetailHuman(detail *TaskDetailRecord) (string, error) {
	taskBlock, err := taskBundle(detail.Task).human()
	if err != nil {
		return "", err
	}
	return renderHumanBlocks(
		taskBlock,
		renderHumanTable(
			"Child Tasks",
			[]string{
				"ID",
				taskIdentifierValue,
				taskScopeValue,
				taskWorkspaceValue,
				taskStatusValue,
				taskOwnerValue,
				taskTitleValue,
			},
			taskChildRows(detail.Children),
		),
		renderHumanTable(
			"Dependencies",
			[]string{taskTaskValue, "Depends On", taskKindValue, taskCreatedValue},
			taskDependencyRows(detail.Dependencies),
		),
		renderHumanTable(
			"Task Runs",
			[]string{
				"ID",
				taskStatusValue,
				taskAttemptValue,
				taskSessionValue,
				taskClaimedByValue,
				taskChannelValue,
				taskCoordinationChannelValue,
				taskQueuedValue,
				taskStartedValue,
				taskEndedValue,
				taskErrorValue,
			},
			taskRunRows(detail.Runs),
		),
		renderHumanTable(
			"Task Events",
			[]string{"ID", taskTypeValue, taskRunValue, "Actor", taskOriginValue, taskTimeValue},
			taskEventRows(detail.Events),
		),
	), nil
}

func renderTaskDetailToon(detail *TaskDetailRecord) (string, error) {
	taskBlock, err := taskBundle(detail.Task).toon()
	if err != nil {
		return "", err
	}
	return renderHumanBlocks(
		taskBlock,
		renderToonArray(
			"task_children",
			[]string{
				"id",
				taskIdentifierKey,
				taskScopeKey,
				taskWorkspaceIDKey,
				taskStatusKey,
				taskOwnerKey,
				taskTitleKey,
			},
			taskChildToonRows(detail.Children),
		),
		renderToonArray(
			"task_dependencies",
			[]string{taskTaskIDKey, "depends_on_task_id", taskKindKey, taskCreatedAtKey},
			taskDependencyToonRows(detail.Dependencies),
		),
		renderToonArray(
			"task_runs",
			[]string{
				"id",
				taskStatusKey,
				taskAttemptKey,
				taskSessionIDKey,
				taskClaimedByKey,
				taskNetworkChannelKey,
				taskCoordinationChannelIDKey,
				taskQueuedAtKey,
				taskStartedAtKey,
				taskEndedAtKey,
				taskErrorKey,
			},
			taskRunToonRows(detail.Runs),
		),
		renderToonArray(
			"task_events",
			[]string{"id", "event_type", taskRunIDKey, "actor", taskOriginKey, taskTimestampKey},
			taskEventToonRows(detail.Events),
		),
	), nil
}

func taskRunBundle(item TaskRunRecord) outputBundle {
	return outputBundle{
		jsonValue: item,
		human: func() (string, error) {
			return renderHumanSection("Task Run", []keyValue{
				{Label: "ID", Value: stringOrDash(item.ID)},
				{Label: taskTaskValue, Value: stringOrDash(item.TaskID)},
				{Label: taskStatusValue, Value: stringOrDash(string(item.Status))},
				{Label: taskAttemptValue, Value: intOrDash(item.Attempt)},
				{Label: taskClaimedByValue, Value: stringOrDash(formatTaskActorPtr(item.ClaimedBy))},
				{Label: taskSessionValue, Value: stringOrDash(item.SessionID)},
				{Label: taskOriginValue, Value: stringOrDash(formatTaskOrigin(item.Origin))},
				{Label: "Idempotency Key", Value: stringOrDash(item.IdempotencyKey)},
				{Label: taskChannelValue, Value: stringOrDash(item.NetworkChannel)},
				{Label: taskCoordinationChannelValue, Value: stringOrDash(item.CoordinationChannelID)},
				{Label: taskQueuedValue, Value: stringOrDash(formatTime(item.QueuedAt))},
				{Label: "Claimed", Value: stringOrDash(formatTimePtr(item.ClaimedAt))},
				{Label: taskStartedValue, Value: stringOrDash(formatTimePtr(item.StartedAt))},
				{Label: taskEndedValue, Value: stringOrDash(formatTimePtr(item.EndedAt))},
				{Label: taskErrorValue, Value: stringOrDash(item.Error)},
				{Label: taskResultValue, Value: stringOrDash(compactJSON(item.Result))},
			}), nil
		},
		toon: func() (string, error) {
			return renderToonObject("task_run", []string{
				"id",
				taskTaskIDKey,
				taskStatusKey,
				taskAttemptKey,
				taskClaimedByKey,
				taskSessionIDKey,
				taskOriginKey,
				"idempotency_key",
				taskNetworkChannelKey,
				taskCoordinationChannelIDKey,
				taskQueuedAtKey,
				"claimed_at",
				taskStartedAtKey,
				taskEndedAtKey,
				taskErrorKey,
				"result",
			}, []string{
				item.ID,
				item.TaskID,
				string(item.Status),
				strconv.Itoa(item.Attempt),
				formatTaskActorPtr(item.ClaimedBy),
				item.SessionID,
				formatTaskOrigin(item.Origin),
				item.IdempotencyKey,
				item.NetworkChannel,
				item.CoordinationChannelID,
				formatTime(item.QueuedAt),
				formatTimePtr(item.ClaimedAt),
				formatTimePtr(item.StartedAt),
				formatTimePtr(item.EndedAt),
				item.Error,
				compactJSON(item.Result),
			}), nil
		},
	}
}

func taskRunListBundle(items []TaskRunRecord) outputBundle {
	return listBundle(
		items,
		items,
		"Task Runs",
		[]string{
			"ID",
			taskStatusValue,
			taskAttemptValue,
			taskSessionValue,
			taskClaimedByValue,
			taskChannelValue,
			taskCoordinationChannelValue,
			taskQueuedValue,
			taskStartedValue,
			taskEndedValue,
			taskErrorValue,
		},
		"task_runs",
		[]string{
			"id",
			taskStatusKey,
			taskAttemptKey,
			taskSessionIDKey,
			taskClaimedByKey,
			taskNetworkChannelKey,
			taskCoordinationChannelIDKey,
			taskQueuedAtKey,
			taskStartedAtKey,
			taskEndedAtKey,
			taskErrorKey,
		},
		func(item TaskRunRecord) []string {
			return []string{
				stringOrDash(item.ID),
				stringOrDash(string(item.Status)),
				intOrDash(item.Attempt),
				stringOrDash(item.SessionID),
				stringOrDash(formatTaskActorPtr(item.ClaimedBy)),
				stringOrDash(item.NetworkChannel),
				stringOrDash(item.CoordinationChannelID),
				stringOrDash(formatTime(item.QueuedAt)),
				stringOrDash(formatTimePtr(item.StartedAt)),
				stringOrDash(formatTimePtr(item.EndedAt)),
				stringOrDash(item.Error),
			}
		},
		func(item TaskRunRecord) []string {
			return []string{
				item.ID,
				string(item.Status),
				strconv.Itoa(item.Attempt),
				item.SessionID,
				formatTaskActorPtr(item.ClaimedBy),
				item.NetworkChannel,
				item.CoordinationChannelID,
				formatTime(item.QueuedAt),
				formatTimePtr(item.StartedAt),
				formatTimePtr(item.EndedAt),
				item.Error,
			}
		},
	)
}

func taskChildRows(items []TaskSummaryRecord) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			stringOrDash(item.ID),
			stringOrDash(item.Identifier),
			stringOrDash(string(item.Scope)),
			stringOrDash(item.WorkspaceID),
			stringOrDash(string(item.Status)),
			stringOrDash(formatTaskOwnership(item.Owner)),
			stringOrDash(item.Title),
		})
	}
	return rows
}

func taskChildToonRows(items []TaskSummaryRecord) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.ID,
			item.Identifier,
			string(item.Scope),
			item.WorkspaceID,
			string(item.Status),
			formatTaskOwnership(item.Owner),
			item.Title,
		})
	}
	return rows
}

func taskDependencyRows(items []TaskDependencyRecord) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			stringOrDash(item.TaskID),
			stringOrDash(item.DependsOnTaskID),
			stringOrDash(string(item.Kind)),
			stringOrDash(formatTime(item.CreatedAt)),
		})
	}
	return rows
}

func taskDependencyToonRows(items []TaskDependencyRecord) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.TaskID,
			item.DependsOnTaskID,
			string(item.Kind),
			formatTime(item.CreatedAt),
		})
	}
	return rows
}

func taskRunRows(items []TaskRunRecord) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			stringOrDash(item.ID),
			stringOrDash(string(item.Status)),
			intOrDash(item.Attempt),
			stringOrDash(item.SessionID),
			stringOrDash(formatTaskActorPtr(item.ClaimedBy)),
			stringOrDash(item.NetworkChannel),
			stringOrDash(item.CoordinationChannelID),
			stringOrDash(formatTime(item.QueuedAt)),
			stringOrDash(formatTimePtr(item.StartedAt)),
			stringOrDash(formatTimePtr(item.EndedAt)),
			stringOrDash(item.Error),
		})
	}
	return rows
}

func taskRunToonRows(items []TaskRunRecord) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.ID,
			string(item.Status),
			strconv.Itoa(item.Attempt),
			item.SessionID,
			formatTaskActorPtr(item.ClaimedBy),
			item.NetworkChannel,
			item.CoordinationChannelID,
			formatTime(item.QueuedAt),
			formatTimePtr(item.StartedAt),
			formatTimePtr(item.EndedAt),
			item.Error,
		})
	}
	return rows
}

func taskEventRows(items []TaskEventRecord) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			stringOrDash(item.ID),
			stringOrDash(item.EventType),
			stringOrDash(item.RunID),
			stringOrDash(formatTaskActor(item.Actor)),
			stringOrDash(formatTaskOrigin(item.Origin)),
			stringOrDash(formatTime(item.Timestamp)),
		})
	}
	return rows
}

func taskEventToonRows(items []TaskEventRecord) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.ID,
			item.EventType,
			item.RunID,
			formatTaskActor(item.Actor),
			formatTaskOrigin(item.Origin),
			formatTime(item.Timestamp),
		})
	}
	return rows
}

func formatTaskOwnership(owner *taskpkg.Ownership) string {
	if owner == nil {
		return ""
	}
	return firstNonEmpty(
		string(owner.Kind)+":"+strings.TrimSpace(owner.Ref),
		strings.TrimSpace(owner.Ref),
	)
}

func formatTaskActor(actor taskpkg.ActorIdentity) string {
	return firstNonEmpty(
		string(actor.Kind)+":"+strings.TrimSpace(actor.Ref),
		strings.TrimSpace(actor.Ref),
	)
}

func formatTaskActorPtr(actor *taskpkg.ActorIdentity) string {
	if actor == nil {
		return ""
	}
	return formatTaskActor(*actor)
}

func formatTaskOrigin(origin taskpkg.Origin) string {
	return firstNonEmpty(
		string(origin.Kind)+":"+strings.TrimSpace(origin.Ref),
		strings.TrimSpace(origin.Ref),
	)
}
