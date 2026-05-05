package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/network"
	taskpkg "github.com/pedronauck/agh/internal/task"
	"github.com/spf13/cobra"
)

type taskCreateInput struct {
	ID           string
	Identifier   string
	ScopeRaw     string
	WorkspaceRef string
	NetworkRaw   string
	Title        string
	Description  string
	OwnerKindRaw string
	OwnerRef     string
	MetadataRaw  string
}

type taskUpdateInput struct {
	Title        string
	Description  string
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

func newTaskCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
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
	cmd.AddCommand(newTaskUpdateCommand(deps))
	cmd.AddCommand(newTaskDeleteCommand(deps))
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
		Use:   "list",
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
	cmd.Flags().StringVar(&scopeRaw, "scope", "", "Filter by scope: global or workspace")
	cmd.Flags().StringVar(&workspaceRef, "workspace", "", "Filter by workspace path, name, or ID")
	cmd.Flags().StringVar(&statusRaw, "status", "", "Filter by task status")
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
	)

	cmd := &cobra.Command{
		Use:   "create",
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
				OwnerKindRaw: ownerKindRaw,
				OwnerRef:     ownerRef,
				MetadataRaw:  metadataRaw,
			})
			if err != nil {
				return err
			}

			created, err := client.CreateTask(cmd.Context(), request)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, taskBundle(created))
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "Explicit task ID")
	cmd.Flags().StringVar(&identifier, "identifier", "", "Human-friendly task identifier")
	cmd.Flags().StringVar(&scopeRaw, "scope", "", "Task scope: global or workspace")
	cmd.Flags().
		StringVar(&workspaceRef, "workspace", "", "Workspace path, name, or ID (required when --scope=workspace)")
	cmd.Flags().StringVar(&networkRaw, "channel", "", "Optional network channel binding")
	cmd.Flags().StringVar(&title, "title", "", "Task title")
	cmd.Flags().StringVar(&description, "description", "", "Task description")
	cmd.Flags().StringVar(&ownerKindRaw, "owner-kind", "", "Optional owner kind")
	cmd.Flags().StringVar(&ownerRef, "owner-ref", "", "Optional owner reference")
	cmd.Flags().StringVar(&metadataRaw, "metadata", "", "Optional metadata JSON")
	mustMarkFlagRequired(cmd, "scope")
	mustMarkFlagRequired(cmd, "title")
	return cmd
}

func newTaskGetCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
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

func newTaskUpdateCommand(deps commandDeps) *cobra.Command {
	var (
		title        string
		description  string
		metadataRaw  string
		networkRaw   string
		ownerKindRaw string
		ownerRef     string
		clearOwner   bool
	)

	cmd := &cobra.Command{
		Use:   "update <id>",
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
	cmd.Flags().StringVar(&title, "title", "", "Update the task title")
	cmd.Flags().StringVar(&description, "description", "", "Update the task description")
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
	if cmd.Flags().Changed("title") {
		trimmed := strings.TrimSpace(input.Title)
		if trimmed == "" {
			return UpdateTaskRequest{}, errors.New("cli: --title cannot be blank")
		}
		request.Title = stringPointer(trimmed)
	}
	if cmd.Flags().Changed("description") {
		request.Description = stringPointer(strings.TrimSpace(input.Description))
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
		request.NetworkChannel = stringPointer(strings.TrimSpace(input.NetworkRaw))
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
		Use:   "delete <id>",
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
		Use:   "next",
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
	cmd.Flags().StringVar(&errorMessage, "error", "", "Failure message")
	cmd.Flags().StringVar(&metadataRaw, "metadata", "", "Optional failure metadata JSON")
	mustMarkFlagRequired(cmd, "error")
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
	cmd.Flags().StringVar(&identifier, "identifier", "", "Human-friendly child task identifier")
	cmd.Flags().StringVar(&scopeRaw, "scope", "", "Child task scope: global or workspace")
	cmd.Flags().
		StringVar(&workspaceRef, "workspace", "", "Workspace path, name, or ID (required when --scope=workspace)")
	cmd.Flags().StringVar(&networkRaw, "channel", "", "Optional network channel binding")
	cmd.Flags().StringVar(&title, "title", "", "Child task title")
	cmd.Flags().StringVar(&description, "description", "", "Child task description")
	cmd.Flags().StringVar(&ownerKindRaw, "owner-kind", "", "Optional child owner kind")
	cmd.Flags().StringVar(&ownerRef, "owner-ref", "", "Optional child owner reference")
	cmd.Flags().StringVar(&metadataRaw, "metadata", "", "Optional child metadata JSON")
	mustMarkFlagRequired(cmd, "scope")
	mustMarkFlagRequired(cmd, "title")
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
	cmd.Flags().StringVar(&kindRaw, "kind", "", "Dependency kind")
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
	cmd.Flags().StringVar(&statusRaw, "status", "", "Filter by run status")
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
	cmd.Flags().StringVar(&errorMessage, "error", "", "Failure message")
	cmd.Flags().StringVar(&metadataRaw, "metadata", "", "Optional failure metadata JSON")
	mustMarkFlagRequired(cmd, "error")
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
	if err := scope.Validate("scope"); err != nil {
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
	if err := status.Validate("status"); err != nil {
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
	if err := status.Validate("status"); err != nil {
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
	if err := kind.Validate("kind"); err != nil {
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
	if err := owner.Validate("owner"); err != nil {
		return nil, fmt.Errorf("cli: %w", err)
	}
	return owner, nil
}

func parseJSONFlag(flagName string, raw string) (json.RawMessage, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, fmt.Errorf("cli: --%s requires valid JSON", flagName)
	}
	var decoded any
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		return nil, fmt.Errorf("cli: invalid --%s JSON: %w", flagName, err)
	}
	return json.RawMessage(trimmed), nil
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

	request := CreateTaskRequest{
		ID:             strings.TrimSpace(input.ID),
		Identifier:     strings.TrimSpace(input.Identifier),
		Scope:          scope,
		Workspace:      workspace,
		NetworkChannel: strings.TrimSpace(input.NetworkRaw),
		Title:          strings.TrimSpace(input.Title),
		Description:    strings.TrimSpace(input.Description),
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
			return renderHumanSection("Task", []keyValue{
				{Label: "ID", Value: stringOrDash(item.ID)},
				{Label: "Identifier", Value: stringOrDash(item.Identifier)},
				{Label: "Scope", Value: stringOrDash(string(item.Scope))},
				{Label: "Workspace", Value: stringOrDash(item.WorkspaceID)},
				{Label: "Parent", Value: stringOrDash(item.ParentTaskID)},
				{Label: "Title", Value: stringOrDash(item.Title)},
				{Label: "Description", Value: stringOrDash(item.Description)},
				{Label: "Status", Value: stringOrDash(string(item.Status))},
				{Label: "Owner", Value: stringOrDash(formatTaskOwnership(item.Owner))},
				{Label: "Created By", Value: stringOrDash(formatTaskActor(item.CreatedBy))},
				{Label: "Origin", Value: stringOrDash(formatTaskOrigin(item.Origin))},
				{Label: "Channel", Value: stringOrDash(item.NetworkChannel)},
				{Label: "Created", Value: stringOrDash(formatTime(item.CreatedAt))},
				{Label: "Updated", Value: stringOrDash(formatTime(item.UpdatedAt))},
				{Label: "Closed", Value: stringOrDash(formatTimePtr(item.ClosedAt))},
				{Label: "Metadata", Value: stringOrDash(compactJSON(item.Metadata))},
			}), nil
		},
		toon: func() (string, error) {
			return renderToonObject("task", []string{
				"id",
				"identifier",
				"scope",
				"workspace_id",
				"parent_task_id",
				"title",
				"description",
				"status",
				"owner",
				"created_by",
				"origin",
				"network_channel",
				"created_at",
				"updated_at",
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
		Status: "deleted",
	}
	return outputBundle{
		jsonValue: item,
		human: func() (string, error) {
			return renderHumanSection("Task", []keyValue{
				{Label: "ID", Value: stringOrDash(item.ID)},
				{Label: "Status", Value: item.Status},
			}), nil
		},
		toon: func() (string, error) {
			return renderToonObject("task", []string{"id", "status"}, []string{item.ID, item.Status}), nil
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
			"Identifier",
			"Scope",
			"Workspace",
			"Parent",
			"Status",
			"Owner",
			"Channel",
			"Title",
		},
		"tasks",
		[]string{
			"id",
			"identifier",
			"scope",
			"workspace_id",
			"parent_task_id",
			"status",
			"owner",
			"network_channel",
			"title",
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

func renderTaskDetailHuman(detail *TaskDetailRecord) (string, error) {
	taskBlock, err := taskBundle(detail.Task).human()
	if err != nil {
		return "", err
	}
	return renderHumanBlocks(
		taskBlock,
		renderHumanTable(
			"Child Tasks",
			[]string{"ID", "Identifier", "Scope", "Workspace", "Status", "Owner", "Title"},
			taskChildRows(detail.Children),
		),
		renderHumanTable(
			"Dependencies",
			[]string{"Task", "Depends On", "Kind", "Created"},
			taskDependencyRows(detail.Dependencies),
		),
		renderHumanTable(
			"Task Runs",
			[]string{
				"ID",
				"Status",
				"Attempt",
				"Session",
				"Claimed By",
				"Channel",
				"Coordination Channel",
				"Queued",
				"Started",
				"Ended",
				"Error",
			},
			taskRunRows(detail.Runs),
		),
		renderHumanTable(
			"Task Events",
			[]string{"ID", "Type", "Run", "Actor", "Origin", "Time"},
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
			[]string{"id", "identifier", "scope", "workspace_id", "status", "owner", "title"},
			taskChildToonRows(detail.Children),
		),
		renderToonArray(
			"task_dependencies",
			[]string{"task_id", "depends_on_task_id", "kind", "created_at"},
			taskDependencyToonRows(detail.Dependencies),
		),
		renderToonArray(
			"task_runs",
			[]string{
				"id",
				"status",
				"attempt",
				"session_id",
				"claimed_by",
				"network_channel",
				"coordination_channel_id",
				"queued_at",
				"started_at",
				"ended_at",
				"error",
			},
			taskRunToonRows(detail.Runs),
		),
		renderToonArray(
			"task_events",
			[]string{"id", "event_type", "run_id", "actor", "origin", "timestamp"},
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
				{Label: "Task", Value: stringOrDash(item.TaskID)},
				{Label: "Status", Value: stringOrDash(string(item.Status))},
				{Label: "Attempt", Value: intOrDash(item.Attempt)},
				{Label: "Claimed By", Value: stringOrDash(formatTaskActorPtr(item.ClaimedBy))},
				{Label: "Session", Value: stringOrDash(item.SessionID)},
				{Label: "Origin", Value: stringOrDash(formatTaskOrigin(item.Origin))},
				{Label: "Idempotency Key", Value: stringOrDash(item.IdempotencyKey)},
				{Label: "Channel", Value: stringOrDash(item.NetworkChannel)},
				{Label: "Coordination Channel", Value: stringOrDash(item.CoordinationChannelID)},
				{Label: "Queued", Value: stringOrDash(formatTime(item.QueuedAt))},
				{Label: "Claimed", Value: stringOrDash(formatTimePtr(item.ClaimedAt))},
				{Label: "Started", Value: stringOrDash(formatTimePtr(item.StartedAt))},
				{Label: "Ended", Value: stringOrDash(formatTimePtr(item.EndedAt))},
				{Label: "Error", Value: stringOrDash(item.Error)},
				{Label: "Result", Value: stringOrDash(compactJSON(item.Result))},
			}), nil
		},
		toon: func() (string, error) {
			return renderToonObject("task_run", []string{
				"id",
				"task_id",
				"status",
				"attempt",
				"claimed_by",
				"session_id",
				"origin",
				"idempotency_key",
				"network_channel",
				"coordination_channel_id",
				"queued_at",
				"claimed_at",
				"started_at",
				"ended_at",
				"error",
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
			"Status",
			"Attempt",
			"Session",
			"Claimed By",
			"Channel",
			"Coordination Channel",
			"Queued",
			"Started",
			"Ended",
			"Error",
		},
		"task_runs",
		[]string{
			"id",
			"status",
			"attempt",
			"session_id",
			"claimed_by",
			"network_channel",
			"coordination_channel_id",
			"queued_at",
			"started_at",
			"ended_at",
			"error",
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
