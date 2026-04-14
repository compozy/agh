package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/pedronauck/agh/internal/network"
	taskpkg "github.com/pedronauck/agh/internal/task"
	"github.com/spf13/cobra"
)

func newTaskCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Manage tasks and task runs",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newTaskListCommand(deps))
	cmd.AddCommand(newTaskCreateCommand(deps))
	cmd.AddCommand(newTaskGetCommand(deps))
	cmd.AddCommand(newTaskUpdateCommand(deps))
	cmd.AddCommand(newTaskCancelCommand(deps))
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
			client, _, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			query, err := parseTaskListFilters(scopeRaw, workspaceRef, statusRaw, ownerKindRaw, ownerRef, parentTaskID, networkRaw, last)
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
			client, _, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			scope, workspace, err := resolveTaskScopeWorkspace(scopeRaw, workspaceRef, true)
			if err != nil {
				return err
			}
			if err := validateTaskChannelFlag("channel", networkRaw); err != nil {
				return err
			}

			var owner *taskpkg.Ownership
			if cmd.Flags().Changed("owner-kind") || cmd.Flags().Changed("owner-ref") {
				owner, err = parseRequiredTaskOwnership(ownerKindRaw, ownerRef)
				if err != nil {
					return err
				}
			}

			var metadata json.RawMessage
			if cmd.Flags().Changed("metadata") {
				metadata, err = parseJSONFlag("metadata", metadataRaw)
				if err != nil {
					return err
				}
			}

			request := CreateTaskRequest{
				ID:             strings.TrimSpace(id),
				Identifier:     strings.TrimSpace(identifier),
				Scope:          scope,
				Workspace:      workspace,
				NetworkChannel: strings.TrimSpace(networkRaw),
				Title:          strings.TrimSpace(title),
				Description:    strings.TrimSpace(description),
				Owner:          owner,
				Metadata:       metadata,
			}
			if request.Title == "" {
				return errors.New("cli: --title is required")
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
	cmd.Flags().StringVar(&workspaceRef, "workspace", "", "Workspace path, name, or ID (required when --scope=workspace)")
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
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			taskDetail, err := client.GetTask(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, taskDetailBundle(taskDetail))
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
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			request := UpdateTaskRequest{}
			if cmd.Flags().Changed("title") {
				trimmed := strings.TrimSpace(title)
				if trimmed == "" {
					return errors.New("cli: --title cannot be blank")
				}
				request.Title = stringPointer(trimmed)
			}
			if cmd.Flags().Changed("description") {
				request.Description = stringPointer(strings.TrimSpace(description))
			}
			if cmd.Flags().Changed("metadata") {
				metadata, err := parseJSONFlag("metadata", metadataRaw)
				if err != nil {
					return err
				}
				request.Metadata = &metadata
			}
			if cmd.Flags().Changed("channel") {
				if err := validateTaskChannelFlag("channel", networkRaw); err != nil {
					return err
				}
				request.NetworkChannel = stringPointer(strings.TrimSpace(networkRaw))
			}

			ownerChanged := cmd.Flags().Changed("owner-kind") || cmd.Flags().Changed("owner-ref")
			if clearOwner && ownerChanged {
				return errors.New("cli: --clear-owner cannot be combined with --owner-kind or --owner-ref")
			}
			if ownerChanged {
				owner, err := parseRequiredTaskOwnership(ownerKindRaw, ownerRef)
				if err != nil {
					return err
				}
				request.Owner = owner
			}
			if clearOwner {
				request.ClearOwner = true
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
	cmd.Flags().StringVar(&networkRaw, "channel", "", "Update the network channel; pass an empty value to clear it")
	cmd.Flags().StringVar(&ownerKindRaw, "owner-kind", "", "Update the owner kind")
	cmd.Flags().StringVar(&ownerRef, "owner-ref", "", "Update the owner reference")
	cmd.Flags().BoolVar(&clearOwner, "clear-owner", false, "Remove the current owner")
	return cmd
}

func newTaskCancelCommand(deps commandDeps) *cobra.Command {
	var (
		reason      string
		metadataRaw string
	)

	cmd := &cobra.Command{
		Use:   "cancel <id>",
		Short: "Cancel a task tree",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := clientFromDeps(deps)
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

			cancelled, err := client.CancelTask(cmd.Context(), args[0], request)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, taskBundle(cancelled))
		},
	}
	cmd.Flags().StringVar(&reason, "reason", "", "Optional cancellation reason")
	cmd.Flags().StringVar(&metadataRaw, "metadata", "", "Optional cancellation metadata JSON")
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
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			scope, workspace, err := resolveTaskScopeWorkspace(scopeRaw, workspaceRef, true)
			if err != nil {
				return err
			}
			if err := validateTaskChannelFlag("channel", networkRaw); err != nil {
				return err
			}

			var owner *taskpkg.Ownership
			if cmd.Flags().Changed("owner-kind") || cmd.Flags().Changed("owner-ref") {
				owner, err = parseRequiredTaskOwnership(ownerKindRaw, ownerRef)
				if err != nil {
					return err
				}
			}

			var metadata json.RawMessage
			if cmd.Flags().Changed("metadata") {
				metadata, err = parseJSONFlag("metadata", metadataRaw)
				if err != nil {
					return err
				}
			}

			request := CreateTaskChildRequest{
				ID:             strings.TrimSpace(id),
				Identifier:     strings.TrimSpace(identifier),
				Scope:          scope,
				Workspace:      workspace,
				NetworkChannel: strings.TrimSpace(networkRaw),
				Title:          strings.TrimSpace(title),
				Description:    strings.TrimSpace(description),
				Owner:          owner,
				Metadata:       metadata,
			}
			if request.Title == "" {
				return errors.New("cli: --title is required")
			}

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
	cmd.Flags().StringVar(&workspaceRef, "workspace", "", "Workspace path, name, or ID (required when --scope=workspace)")
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
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := clientFromDeps(deps)
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
			return writeCommandOutput(cmd, taskDetailBundle(updated))
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
			client, _, err := clientFromDeps(deps)
			if err != nil {
				return err
			}

			updated, err := client.RemoveTaskDependency(cmd.Context(), args[0], args[1])
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, taskDetailBundle(updated))
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
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := clientFromDeps(deps)
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
	)

	cmd := &cobra.Command{
		Use:   "enqueue <task-id>",
		Short: "Enqueue a task run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			if err := validateTaskChannelFlag("channel", networkRaw); err != nil {
				return err
			}

			run, err := client.EnqueueTaskRun(cmd.Context(), args[0], EnqueueTaskRunRequest{
				IdempotencyKey: strings.TrimSpace(idempotencyKey),
				NetworkChannel: strings.TrimSpace(networkRaw),
			})
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, taskRunBundle(run))
		},
	}
	cmd.Flags().StringVar(&idempotencyKey, "idempotency-key", "", "Optional idempotency key")
	cmd.Flags().StringVar(&networkRaw, "channel", "", "Optional run channel override")
	return cmd
}

func newTaskRunClaimCommand(deps commandDeps) *cobra.Command {
	var idempotencyKey string

	cmd := &cobra.Command{
		Use:   "claim <run-id>",
		Short: "Claim a queued task run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := clientFromDeps(deps)
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
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := clientFromDeps(deps)
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
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			if strings.TrimSpace(sessionID) == "" {
				return errors.New("cli: --session is required")
			}

			run, err := client.AttachTaskRunSession(cmd.Context(), args[0], AttachTaskRunSessionRequest{
				SessionID: strings.TrimSpace(sessionID),
			})
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
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := clientFromDeps(deps)
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
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := clientFromDeps(deps)
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
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := clientFromDeps(deps)
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

func parseTaskListFilters(scopeRaw string, workspaceRef string, statusRaw string, ownerKindRaw string, ownerRef string, parentTaskID string, channelRaw string, last int) (TaskListQuery, error) {
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
		return TaskListQuery{}, errors.New("cli: --owner-kind and --owner-ref must be provided together")
	}
	if err := validateTaskChannelFlag("channel", channelRaw); err != nil {
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

func parseTaskRunListFilters(statusRaw string, sessionID string, last int) (TaskRunListQuery, error) {
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

func resolveTaskScopeWorkspace(rawScope string, workspaceRef string, scopeRequired bool) (taskpkg.Scope, string, error) {
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

func parseOptionalTaskStatus(raw string) (taskpkg.TaskStatus, error) {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	if trimmed == "" {
		return "", nil
	}
	status := taskpkg.TaskStatus(trimmed)
	if err := status.Validate("status"); err != nil {
		return "", fmt.Errorf("cli: %w", err)
	}
	return status, nil
}

func parseOptionalTaskRunStatus(raw string) (taskpkg.TaskRunStatus, error) {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	if trimmed == "" {
		return "", nil
	}
	status := taskpkg.TaskRunStatus(trimmed)
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

func validateTaskChannelFlag(flagName string, channel string) error {
	trimmed := strings.TrimSpace(channel)
	if trimmed == "" {
		return nil
	}
	if err := network.ValidateChannel(trimmed); err != nil {
		return fmt.Errorf("cli: invalid --%s value %q: %w", flagName, trimmed, err)
	}
	return nil
}

func validateTaskLast(last int) error {
	if last < 0 {
		return fmt.Errorf("cli: --last must be zero or positive: %d", last)
	}
	return nil
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
				"id", "identifier", "scope", "workspace_id", "parent_task_id", "title", "description", "status", "owner", "created_by", "origin", "network_channel", "created_at", "updated_at", "closed_at", "metadata",
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

func taskSummaryListBundle(items []TaskSummaryRecord) outputBundle {
	return listBundle(
		items,
		items,
		"Tasks",
		[]string{"ID", "Identifier", "Scope", "Workspace", "Parent", "Status", "Owner", "Channel", "Title"},
		"tasks",
		[]string{"id", "identifier", "scope", "workspace_id", "parent_task_id", "status", "owner", "network_channel", "title"},
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

func taskDetailBundle(detail TaskDetailRecord) outputBundle {
	return outputBundle{
		jsonValue: detail,
		human: func() (string, error) {
			taskBlock, err := taskBundle(detail.Task).human()
			if err != nil {
				return "", err
			}
			return renderHumanBlocks(
				taskBlock,
				renderHumanTable("Child Tasks", []string{"ID", "Identifier", "Scope", "Workspace", "Status", "Owner", "Title"}, taskChildRows(detail.Children)),
				renderHumanTable("Dependencies", []string{"Task", "Depends On", "Kind", "Created"}, taskDependencyRows(detail.Dependencies)),
				renderHumanTable("Task Runs", []string{"ID", "Status", "Attempt", "Session", "Claimed By", "Channel", "Queued", "Started", "Ended", "Error"}, taskRunRows(detail.Runs)),
				renderHumanTable("Task Events", []string{"ID", "Type", "Run", "Actor", "Origin", "Time"}, taskEventRows(detail.Events)),
			), nil
		},
		toon: func() (string, error) {
			taskBlock, err := taskBundle(detail.Task).toon()
			if err != nil {
				return "", err
			}
			return renderHumanBlocks(
				taskBlock,
				renderToonArray("task_children", []string{"id", "identifier", "scope", "workspace_id", "status", "owner", "title"}, taskChildToonRows(detail.Children)),
				renderToonArray("task_dependencies", []string{"task_id", "depends_on_task_id", "kind", "created_at"}, taskDependencyToonRows(detail.Dependencies)),
				renderToonArray("task_runs", []string{"id", "status", "attempt", "session_id", "claimed_by", "network_channel", "queued_at", "started_at", "ended_at", "error"}, taskRunToonRows(detail.Runs)),
				renderToonArray("task_events", []string{"id", "event_type", "run_id", "actor", "origin", "timestamp"}, taskEventToonRows(detail.Events)),
			), nil
		},
	}
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
				"id", "task_id", "status", "attempt", "claimed_by", "session_id", "origin", "idempotency_key", "network_channel", "queued_at", "claimed_at", "started_at", "ended_at", "error", "result",
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
		[]string{"ID", "Status", "Attempt", "Session", "Claimed By", "Channel", "Queued", "Started", "Ended", "Error"},
		"task_runs",
		[]string{"id", "status", "attempt", "session_id", "claimed_by", "network_channel", "queued_at", "started_at", "ended_at", "error"},
		func(item TaskRunRecord) []string {
			return []string{
				stringOrDash(item.ID),
				stringOrDash(string(item.Status)),
				intOrDash(item.Attempt),
				stringOrDash(item.SessionID),
				stringOrDash(formatTaskActorPtr(item.ClaimedBy)),
				stringOrDash(item.NetworkChannel),
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
	return firstNonEmpty(string(owner.Kind)+":"+strings.TrimSpace(owner.Ref), strings.TrimSpace(owner.Ref))
}

func formatTaskActor(actor taskpkg.ActorIdentity) string {
	return firstNonEmpty(string(actor.Kind)+":"+strings.TrimSpace(actor.Ref), strings.TrimSpace(actor.Ref))
}

func formatTaskActorPtr(actor *taskpkg.ActorIdentity) string {
	if actor == nil {
		return ""
	}
	return formatTaskActor(*actor)
}

func formatTaskOrigin(origin taskpkg.Origin) string {
	return firstNonEmpty(string(origin.Kind)+":"+strings.TrimSpace(origin.Ref), strings.TrimSpace(origin.Ref))
}
