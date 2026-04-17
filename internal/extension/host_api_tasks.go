package extensionpkg

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	apicontract "github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/network"
	taskpkg "github.com/pedronauck/agh/internal/task"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func (h *HostAPIHandler) handleTasks(ctx context.Context, raw json.RawMessage) (any, error) {
	var params hostAPITasksParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}

	manager, actor, err := h.taskManagerAndActor(ctx)
	if err != nil {
		return nil, err
	}
	query, err := h.taskQueryFromParams(ctx, params)
	if err != nil {
		return nil, err
	}

	tasks, err := manager.ListTasks(ctx, query, actor)
	if err != nil {
		return nil, mapTaskRPCError("", err)
	}
	return taskSummaryPayloadsFromSummaries(tasks), nil
}

func (h *HostAPIHandler) handleTasksGet(ctx context.Context, raw json.RawMessage) (any, error) {
	var params hostAPITaskTargetParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	taskID := strings.TrimSpace(params.ID)
	if taskID == "" {
		return nil, invalidParamsRPCError(errors.New("id is required"))
	}

	manager, actor, err := h.taskManagerAndActor(ctx)
	if err != nil {
		return nil, err
	}

	view, err := manager.GetTask(ctx, taskID, actor)
	if err != nil {
		return nil, mapTaskRPCError(taskID, err)
	}
	return taskDetailPayloadFromView(view), nil
}

func (h *HostAPIHandler) handleTasksCreate(ctx context.Context, raw json.RawMessage) (any, error) {
	var params hostAPITaskCreateParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}

	manager, actor, err := h.taskManagerAndActor(ctx)
	if err != nil {
		return nil, err
	}
	spec, err := h.createTaskSpecFromRequest(ctx, params)
	if err != nil {
		return nil, err
	}

	record, err := manager.CreateTask(ctx, spec, actor)
	if err != nil {
		return nil, mapTaskRPCError(spec.ID, err)
	}
	return taskPayloadFromTask(record), nil
}

func (h *HostAPIHandler) handleTasksUpdate(ctx context.Context, raw json.RawMessage) (any, error) {
	var params hostAPITaskUpdateParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	taskID := strings.TrimSpace(params.ID)
	if taskID == "" {
		return nil, invalidParamsRPCError(errors.New("id is required"))
	}
	if !params.HasChanges() {
		return nil, invalidParamsRPCError(errors.New("task update must include at least one mutable field"))
	}

	manager, actor, err := h.taskManagerAndActor(ctx)
	if err != nil {
		return nil, err
	}
	patch, err := taskPatchFromRequest(params.UpdateTaskRequest)
	if err != nil {
		return nil, err
	}

	record, err := manager.UpdateTask(ctx, taskID, patch, actor)
	if err != nil {
		return nil, mapTaskRPCError(taskID, err)
	}
	return taskPayloadFromTask(record), nil
}

func (h *HostAPIHandler) handleTasksCancel(ctx context.Context, raw json.RawMessage) (any, error) {
	var params hostAPITaskCancelParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	taskID := strings.TrimSpace(params.ID)
	if taskID == "" {
		return nil, invalidParamsRPCError(errors.New("id is required"))
	}

	manager, actor, err := h.taskManagerAndActor(ctx)
	if err != nil {
		return nil, err
	}
	cancelReq, err := cancelTaskFromRequest(params.CancelTaskRequest)
	if err != nil {
		return nil, err
	}

	record, err := manager.CancelTask(ctx, taskID, cancelReq, actor)
	if err != nil {
		return nil, mapTaskRPCError(taskID, err)
	}
	return taskPayloadFromTask(record), nil
}

func (h *HostAPIHandler) handleTasksRuns(ctx context.Context, raw json.RawMessage) (any, error) {
	var params hostAPITaskRunsParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	taskID := strings.TrimSpace(params.ID)
	if taskID == "" {
		return nil, invalidParamsRPCError(errors.New("id is required"))
	}

	manager, actor, err := h.taskManagerAndActor(ctx)
	if err != nil {
		return nil, err
	}
	query, err := taskRunQueryFromParams(params.TaskRunListQuery)
	if err != nil {
		return nil, err
	}

	runs, err := manager.ListTaskRuns(ctx, taskID, query, actor)
	if err != nil {
		return nil, mapTaskRPCError(taskID, err)
	}
	return taskRunPayloadsFromRuns(runs), nil
}

func (h *HostAPIHandler) handleTasksRunsEnqueue(ctx context.Context, raw json.RawMessage) (any, error) {
	var params hostAPITaskRunEnqueueParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	taskID := strings.TrimSpace(params.TaskID)
	if taskID == "" {
		return nil, invalidParamsRPCError(errors.New("task_id is required"))
	}

	manager, actor, err := h.taskManagerAndActor(ctx)
	if err != nil {
		return nil, err
	}
	spec, err := enqueueTaskRunFromRequest(taskID, params.EnqueueTaskRunRequest)
	if err != nil {
		return nil, err
	}

	run, err := manager.EnqueueRun(ctx, spec, actor)
	if err != nil {
		return nil, mapTaskRPCError(taskID, err)
	}
	return taskRunPayloadFromRun(run), nil
}

func (h *HostAPIHandler) handleTasksRunsClaim(ctx context.Context, raw json.RawMessage) (any, error) {
	var params hostAPITaskRunClaimParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	runID := strings.TrimSpace(params.ID)
	if runID == "" {
		return nil, invalidParamsRPCError(errors.New("id is required"))
	}

	manager, actor, err := h.taskManagerAndActor(ctx)
	if err != nil {
		return nil, err
	}
	claim, err := claimTaskRunFromRequest(params.ClaimTaskRunRequest)
	if err != nil {
		return nil, err
	}

	run, err := manager.ClaimRun(ctx, runID, claim, actor)
	if err != nil {
		return nil, mapTaskRPCError(runID, err)
	}
	return taskRunPayloadFromRun(run), nil
}

func (h *HostAPIHandler) handleTasksRunsStart(ctx context.Context, raw json.RawMessage) (any, error) {
	var params hostAPITaskRunStartParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	runID := strings.TrimSpace(params.ID)
	if runID == "" {
		return nil, invalidParamsRPCError(errors.New("id is required"))
	}

	manager, actor, err := h.taskManagerAndActor(ctx)
	if err != nil {
		return nil, err
	}
	startReq, err := startTaskRunFromRequest(params.StartTaskRunRequest)
	if err != nil {
		return nil, err
	}

	run, err := manager.StartRun(ctx, runID, startReq, actor)
	if err != nil {
		return nil, mapTaskRPCError(runID, err)
	}
	return taskRunPayloadFromRun(run), nil
}

func (h *HostAPIHandler) handleTasksRunsAttachSession(ctx context.Context, raw json.RawMessage) (any, error) {
	var params hostAPITaskRunAttachSessionParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	runID := strings.TrimSpace(params.ID)
	if runID == "" {
		return nil, invalidParamsRPCError(errors.New("id is required"))
	}

	manager, actor, err := h.taskManagerAndActor(ctx)
	if err != nil {
		return nil, err
	}
	sessionID, err := attachTaskRunSessionIDFromRequest(params.AttachTaskRunSessionRequest)
	if err != nil {
		return nil, err
	}

	run, err := manager.AttachRunSession(ctx, runID, sessionID, actor)
	if err != nil {
		return nil, mapTaskRPCError(runID, err)
	}
	return taskRunPayloadFromRun(run), nil
}

func (h *HostAPIHandler) handleTasksRunsComplete(ctx context.Context, raw json.RawMessage) (any, error) {
	var params hostAPITaskRunCompleteParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	runID := strings.TrimSpace(params.ID)
	if runID == "" {
		return nil, invalidParamsRPCError(errors.New("id is required"))
	}

	manager, actor, err := h.taskManagerAndActor(ctx)
	if err != nil {
		return nil, err
	}
	result, err := completeTaskRunFromRequest(params.CompleteTaskRunRequest)
	if err != nil {
		return nil, err
	}

	run, err := manager.CompleteRun(ctx, runID, result, actor)
	if err != nil {
		return nil, mapTaskRPCError(runID, err)
	}
	return taskRunPayloadFromRun(run), nil
}

func (h *HostAPIHandler) handleTasksRunsFail(ctx context.Context, raw json.RawMessage) (any, error) {
	var params hostAPITaskRunFailParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	runID := strings.TrimSpace(params.ID)
	if runID == "" {
		return nil, invalidParamsRPCError(errors.New("id is required"))
	}

	manager, actor, err := h.taskManagerAndActor(ctx)
	if err != nil {
		return nil, err
	}
	failure, err := failTaskRunFromRequest(params.FailTaskRunRequest)
	if err != nil {
		return nil, err
	}

	run, err := manager.FailRun(ctx, runID, failure, actor)
	if err != nil {
		return nil, mapTaskRPCError(runID, err)
	}
	return taskRunPayloadFromRun(run), nil
}

func (h *HostAPIHandler) handleTasksRunsCancel(ctx context.Context, raw json.RawMessage) (any, error) {
	var params hostAPITaskRunCancelParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	runID := strings.TrimSpace(params.ID)
	if runID == "" {
		return nil, invalidParamsRPCError(errors.New("id is required"))
	}

	manager, actor, err := h.taskManagerAndActor(ctx)
	if err != nil {
		return nil, err
	}
	cancelReq, err := cancelTaskRunFromRequest(params.CancelTaskRunRequest)
	if err != nil {
		return nil, err
	}

	run, err := manager.CancelRun(ctx, runID, cancelReq, actor)
	if err != nil {
		return nil, mapTaskRPCError(runID, err)
	}
	return taskRunPayloadFromRun(run), nil
}

func (h *HostAPIHandler) taskManager() (hostAPITaskManager, error) {
	if h == nil {
		return nil, errors.New("extension: host api handler is required")
	}
	if h.tasks == nil {
		return nil, errors.New("extension: task manager is not configured")
	}
	return h.tasks, nil
}

func (h *HostAPIHandler) taskManagerAndActor(ctx context.Context) (hostAPITaskManager, taskpkg.ActorContext, error) {
	manager, err := h.taskManager()
	if err != nil {
		return nil, taskpkg.ActorContext{}, err
	}
	actor, err := h.taskActorContext(ctx)
	if err != nil {
		return nil, taskpkg.ActorContext{}, err
	}
	return manager, actor, nil
}

func (h *HostAPIHandler) taskActorContext(ctx context.Context) (taskpkg.ActorContext, error) {
	extName := hostAPIExtensionNameFromContext(ctx)
	if extName == "" {
		return taskpkg.ActorContext{}, unavailableRPCError(errors.New("extension name is not available"))
	}

	actor, err := taskpkg.DeriveExtensionActorContext(extName, "")
	if err != nil {
		return taskpkg.ActorContext{}, invalidParamsRPCError(err)
	}
	return actor, nil
}

func (h *HostAPIHandler) taskQueryFromParams(
	ctx context.Context,
	params hostAPITasksParams,
) (taskpkg.Query, error) {
	query := taskpkg.Query{
		Scope:        params.Scope.Normalize(),
		Status:       params.Status.Normalize(),
		OwnerKind:    params.OwnerKind.Normalize(),
		OwnerRef:     strings.TrimSpace(params.OwnerRef),
		ParentTaskID: strings.TrimSpace(params.ParentTaskID),
		Limit:        params.Limit,
	}
	if query.Scope.Normalize() != "" {
		if err := query.Scope.Validate("task_query.scope"); err != nil {
			return taskpkg.Query{}, invalidParamsRPCError(err)
		}
	}
	if workspaceRef := strings.TrimSpace(params.Workspace); workspaceRef != "" {
		if query.Scope.Normalize() == taskpkg.ScopeGlobal {
			if err := taskpkg.ValidateScopeBinding(query.Scope, workspaceRef, "task_query", "workspace"); err != nil {
				return taskpkg.Query{}, invalidParamsRPCError(err)
			}
		}
		workspaceID, err := h.resolveTaskWorkspaceID(ctx, workspaceRef)
		if err != nil {
			return taskpkg.Query{}, err
		}
		query.WorkspaceID = workspaceID
	}
	if err := validateTaskChannel("task_query.network_channel", params.NetworkChannel); err != nil {
		return taskpkg.Query{}, err
	}
	query.NetworkChannel = strings.TrimSpace(params.NetworkChannel)
	if err := query.Validate("task_query"); err != nil {
		return taskpkg.Query{}, invalidParamsRPCError(err)
	}
	return query, nil
}

func taskRunQueryFromParams(params apicontract.TaskRunListQuery) (taskpkg.RunQuery, error) {
	query := taskpkg.RunQuery{
		Status:    params.Status.Normalize(),
		SessionID: strings.TrimSpace(params.SessionID),
		Limit:     params.Limit,
	}
	if err := query.Validate("task_run_query"); err != nil {
		return taskpkg.RunQuery{}, invalidParamsRPCError(err)
	}
	return query, nil
}

func (h *HostAPIHandler) createTaskSpecFromRequest(
	ctx context.Context,
	req apicontract.CreateTaskRequest,
) (taskpkg.CreateTask, error) {
	scope := req.Scope.Normalize()
	if err := scope.Validate("create_task.scope"); err != nil {
		return taskpkg.CreateTask{}, invalidParamsRPCError(err)
	}
	workspaceID, err := h.resolveTaskWorkspaceBinding(ctx, scope, strings.TrimSpace(req.Workspace), "create_task")
	if err != nil {
		return taskpkg.CreateTask{}, err
	}
	if err := validateTaskChannel("create_task.network_channel", req.NetworkChannel); err != nil {
		return taskpkg.CreateTask{}, err
	}

	spec := taskpkg.CreateTask{
		ID:             strings.TrimSpace(req.ID),
		Identifier:     strings.TrimSpace(req.Identifier),
		Scope:          scope,
		WorkspaceID:    workspaceID,
		NetworkChannel: strings.TrimSpace(req.NetworkChannel),
		Title:          strings.TrimSpace(req.Title),
		Description:    strings.TrimSpace(req.Description),
		Owner:          cloneOwnership(req.Owner),
		Metadata:       cloneRawMessage(req.Metadata),
	}
	if err := spec.Validate("create_task"); err != nil {
		return taskpkg.CreateTask{}, invalidParamsRPCError(err)
	}
	return spec, nil
}

func taskPatchFromRequest(req apicontract.UpdateTaskRequest) (taskpkg.Patch, error) {
	if req.NetworkChannel != nil {
		if err := validateTaskChannel("task_patch.network_channel", *req.NetworkChannel); err != nil {
			return taskpkg.Patch{}, err
		}
	}

	patch := taskpkg.Patch{
		Title:          trimStringPtr(req.Title),
		Description:    trimStringPtr(req.Description),
		Metadata:       cloneRawMessagePtr(req.Metadata),
		NetworkChannel: trimStringPtr(req.NetworkChannel),
		Owner:          cloneOwnership(req.Owner),
		ClearOwner:     req.ClearOwner,
	}
	if err := patch.Validate("task_patch"); err != nil {
		return taskpkg.Patch{}, invalidParamsRPCError(err)
	}
	return patch, nil
}

func cancelTaskFromRequest(req apicontract.CancelTaskRequest) (taskpkg.CancelTask, error) {
	cancelReq := taskpkg.CancelTask{
		Reason:   strings.TrimSpace(req.Reason),
		Metadata: cloneRawMessage(req.Metadata),
	}
	if err := cancelReq.Validate("cancel_task"); err != nil {
		return taskpkg.CancelTask{}, invalidParamsRPCError(err)
	}
	return cancelReq, nil
}

func enqueueTaskRunFromRequest(taskID string, req apicontract.EnqueueTaskRunRequest) (taskpkg.EnqueueRun, error) {
	if err := validateTaskChannel("enqueue_run.network_channel", req.NetworkChannel); err != nil {
		return taskpkg.EnqueueRun{}, err
	}

	spec := taskpkg.EnqueueRun{
		TaskID:         strings.TrimSpace(taskID),
		IdempotencyKey: strings.TrimSpace(req.IdempotencyKey),
		NetworkChannel: strings.TrimSpace(req.NetworkChannel),
	}
	if err := spec.Validate("enqueue_run"); err != nil {
		return taskpkg.EnqueueRun{}, invalidParamsRPCError(err)
	}
	return spec, nil
}

func claimTaskRunFromRequest(req apicontract.ClaimTaskRunRequest) (taskpkg.ClaimRun, error) {
	claim := taskpkg.ClaimRun{IdempotencyKey: strings.TrimSpace(req.IdempotencyKey)}
	if err := claim.Validate("claim_run"); err != nil {
		return taskpkg.ClaimRun{}, invalidParamsRPCError(err)
	}
	return claim, nil
}

func startTaskRunFromRequest(req apicontract.StartTaskRunRequest) (taskpkg.StartRun, error) {
	startReq := taskpkg.StartRun{IdempotencyKey: strings.TrimSpace(req.IdempotencyKey)}
	if err := startReq.Validate("start_run"); err != nil {
		return taskpkg.StartRun{}, invalidParamsRPCError(err)
	}
	return startReq, nil
}

func attachTaskRunSessionIDFromRequest(req apicontract.AttachTaskRunSessionRequest) (string, error) {
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return "", invalidParamsRPCError(errors.New("session_id is required"))
	}
	return sessionID, nil
}

func completeTaskRunFromRequest(req apicontract.CompleteTaskRunRequest) (taskpkg.RunResult, error) {
	result := taskpkg.RunResult{Value: cloneRawMessage(req.Result)}
	if err := result.Validate("run_result"); err != nil {
		return taskpkg.RunResult{}, invalidParamsRPCError(err)
	}
	return result, nil
}

func failTaskRunFromRequest(req apicontract.FailTaskRunRequest) (taskpkg.RunFailure, error) {
	failure := taskpkg.RunFailure{
		Error:    strings.TrimSpace(req.Error),
		Metadata: cloneRawMessage(req.Metadata),
	}
	if err := failure.Validate("run_failure"); err != nil {
		return taskpkg.RunFailure{}, invalidParamsRPCError(err)
	}
	return failure, nil
}

func cancelTaskRunFromRequest(req apicontract.CancelTaskRunRequest) (taskpkg.CancelRun, error) {
	cancelReq := taskpkg.CancelRun{
		Reason:   strings.TrimSpace(req.Reason),
		Metadata: cloneRawMessage(req.Metadata),
	}
	if err := cancelReq.Validate("cancel_run"); err != nil {
		return taskpkg.CancelRun{}, invalidParamsRPCError(err)
	}
	return cancelReq, nil
}

func (h *HostAPIHandler) resolveTaskWorkspaceBinding(
	ctx context.Context,
	scope taskpkg.Scope,
	workspaceRef string,
	path string,
) (string, error) {
	trimmed := strings.TrimSpace(workspaceRef)
	if err := taskpkg.ValidateScopeBinding(scope, trimmed, path, "workspace"); err != nil {
		return "", invalidParamsRPCError(err)
	}
	if scope.Normalize() != taskpkg.ScopeWorkspace {
		return "", nil
	}
	return h.resolveTaskWorkspaceID(ctx, trimmed)
}

func (h *HostAPIHandler) resolveTaskWorkspaceID(ctx context.Context, workspaceRef string) (string, error) {
	trimmed := strings.TrimSpace(workspaceRef)
	if trimmed == "" {
		return "", nil
	}
	if h.workspaces == nil {
		return trimmed, nil
	}

	resolved, err := h.workspaces.Resolve(ctx, trimmed)
	if err != nil {
		if errors.Is(err, workspacepkg.ErrWorkspaceNotFound) {
			return "", notFoundRPCError("workspace", trimmed, err)
		}
		return "", err
	}
	return strings.TrimSpace(resolved.ID), nil
}

func validateTaskChannel(path string, channel string) error {
	trimmed := strings.TrimSpace(channel)
	if trimmed == "" {
		return nil
	}
	if err := network.ValidateChannel(trimmed); err != nil {
		return invalidParamsRPCError(fmt.Errorf("%s: %w", path, err))
	}
	return nil
}

func mapTaskRPCError(id string, err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, workspacepkg.ErrWorkspaceNotFound):
		return notFoundRPCError("workspace", id, err)
	case errors.Is(err, taskpkg.ErrTaskNotFound):
		return notFoundRPCError("task", id, err)
	case errors.Is(err, taskpkg.ErrTaskRunNotFound):
		return notFoundRPCError("task_run", id, err)
	case errors.Is(err, taskpkg.ErrTaskDependencyNotFound):
		return notFoundRPCError("task_dependency", id, err)
	case errors.Is(err, taskpkg.ErrValidation),
		errors.Is(err, taskpkg.ErrImmutableField),
		errors.Is(err, taskpkg.ErrInvalidScopeBinding),
		errors.Is(err, taskpkg.ErrPayloadTooLarge),
		errors.Is(err, taskpkg.ErrGraphLimitExceeded),
		errors.Is(err, taskpkg.ErrCycleDetected),
		errors.Is(err, taskpkg.ErrInvalidStatusTransition),
		errors.Is(err, taskpkg.ErrSessionAlreadyBound),
		errors.Is(err, taskpkg.ErrSessionAttachNotAllowed),
		errors.Is(err, taskpkg.ErrStaleNetworkChannel),
		errors.Is(err, taskpkg.ErrPermissionDenied):
		return invalidParamsRPCError(err)
	default:
		return err
	}
}

func taskSummaryPayloadsFromSummaries(tasks []taskpkg.Summary) []apicontract.TaskSummaryPayload {
	payloads := make([]apicontract.TaskSummaryPayload, 0, len(tasks))
	for _, record := range tasks {
		payloads = append(payloads, taskSummaryPayloadFromSummary(record))
	}
	return payloads
}

func taskSummaryPayloadFromSummary(record taskpkg.Summary) apicontract.TaskSummaryPayload {
	return apicontract.TaskSummaryPayload{
		ID:             record.ID,
		Identifier:     record.Identifier,
		Scope:          record.Scope,
		WorkspaceID:    record.WorkspaceID,
		ParentTaskID:   record.ParentTaskID,
		NetworkChannel: record.NetworkChannel,
		Title:          record.Title,
		Status:         record.Status,
		Owner:          cloneOwnership(record.Owner),
		CreatedBy:      record.CreatedBy,
		Origin:         record.Origin,
		CreatedAt:      record.CreatedAt,
		UpdatedAt:      record.UpdatedAt,
		ClosedAt:       optionalTime(record.ClosedAt),
	}
}

func taskPayloadFromTask(record *taskpkg.Task) apicontract.TaskPayload {
	if record == nil {
		return apicontract.TaskPayload{}
	}

	return apicontract.TaskPayload{
		ID:             record.ID,
		Identifier:     record.Identifier,
		Scope:          record.Scope,
		WorkspaceID:    record.WorkspaceID,
		ParentTaskID:   record.ParentTaskID,
		NetworkChannel: record.NetworkChannel,
		Title:          record.Title,
		Description:    record.Description,
		Status:         record.Status,
		Owner:          cloneOwnership(record.Owner),
		CreatedBy:      record.CreatedBy,
		Origin:         record.Origin,
		CreatedAt:      record.CreatedAt,
		UpdatedAt:      record.UpdatedAt,
		ClosedAt:       optionalTime(record.ClosedAt),
		Metadata:       cloneRawMessage(record.Metadata),
	}
}

func taskDependencyPayloadsFromDependencies(dependencies []taskpkg.Dependency) []apicontract.TaskDependencyPayload {
	payloads := make([]apicontract.TaskDependencyPayload, 0, len(dependencies))
	for _, dependency := range dependencies {
		payloads = append(payloads, apicontract.TaskDependencyPayload{
			TaskID:          dependency.TaskID,
			DependsOnTaskID: dependency.DependsOnTaskID,
			Kind:            dependency.Kind,
			CreatedAt:       dependency.CreatedAt,
		})
	}
	return payloads
}

func taskRunPayloadsFromRuns(runs []taskpkg.Run) []apicontract.TaskRunPayload {
	payloads := make([]apicontract.TaskRunPayload, 0, len(runs))
	for _, run := range runs {
		payloads = append(payloads, taskRunPayloadFromRun(&run))
	}
	return payloads
}

func taskRunPayloadFromRun(run *taskpkg.Run) apicontract.TaskRunPayload {
	if run == nil {
		return apicontract.TaskRunPayload{}
	}

	return apicontract.TaskRunPayload{
		ID:             run.ID,
		TaskID:         run.TaskID,
		Status:         run.Status,
		Attempt:        run.Attempt,
		ClaimedBy:      cloneActorIdentity(run.ClaimedBy),
		SessionID:      run.SessionID,
		Origin:         run.Origin,
		IdempotencyKey: run.IdempotencyKey,
		NetworkChannel: run.NetworkChannel,
		QueuedAt:       run.QueuedAt,
		ClaimedAt:      optionalTime(run.ClaimedAt),
		StartedAt:      optionalTime(run.StartedAt),
		EndedAt:        optionalTime(run.EndedAt),
		Error:          run.Error,
		Result:         cloneRawMessage(run.Result),
	}
}

func optionalTime(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	cloned := value
	return &cloned
}

func taskEventPayloadsFromEvents(events []taskpkg.Event) []apicontract.TaskEventPayload {
	payloads := make([]apicontract.TaskEventPayload, 0, len(events))
	for _, event := range events {
		payloads = append(payloads, apicontract.TaskEventPayload{
			ID:        event.ID,
			TaskID:    event.TaskID,
			RunID:     event.RunID,
			EventType: event.EventType,
			Actor:     event.Actor,
			Origin:    event.Origin,
			Payload:   cloneRawMessage(event.Payload),
			Timestamp: event.Timestamp,
		})
	}
	return payloads
}

func taskDetailPayloadFromView(view *taskpkg.View) apicontract.TaskDetailPayload {
	if view == nil {
		return apicontract.TaskDetailPayload{}
	}

	return apicontract.TaskDetailPayload{
		Task:         taskPayloadFromTask(&view.Task),
		Children:     taskSummaryPayloadsFromSummaries(view.Children),
		Dependencies: taskDependencyPayloadsFromDependencies(view.Dependencies),
		Runs:         taskRunPayloadsFromRuns(view.Runs),
		Events:       taskEventPayloadsFromEvents(view.Events),
	}
}

func filterTaskRuns(runs []taskpkg.Run, query taskpkg.RunQuery) []taskpkg.Run {
	filtered := make([]taskpkg.Run, 0, len(runs))
	for _, run := range runs {
		if query.Status.Normalize() != "" && run.Status.Normalize() != query.Status.Normalize() {
			continue
		}
		if strings.TrimSpace(query.SessionID) != "" &&
			strings.TrimSpace(run.SessionID) != strings.TrimSpace(query.SessionID) {
			continue
		}
		filtered = append(filtered, run)
	}
	if query.Limit > 0 && len(filtered) > query.Limit {
		return filtered[:query.Limit]
	}
	return filtered
}

func cloneOwnership(source *taskpkg.Ownership) *taskpkg.Ownership {
	if source == nil {
		return nil
	}
	cloned := *source
	return &cloned
}

func cloneActorIdentity(source *taskpkg.ActorIdentity) *taskpkg.ActorIdentity {
	if source == nil {
		return nil
	}
	cloned := *source
	return &cloned
}

func trimStringPtr(source *string) *string {
	if source == nil {
		return nil
	}
	value := strings.TrimSpace(*source)
	return &value
}

func cloneRawMessagePtr(source *json.RawMessage) *json.RawMessage {
	if source == nil {
		return nil
	}
	cloned := cloneRawMessage(*source)
	return &cloned
}

func cloneRawMessage(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	return append(json.RawMessage(nil), raw...)
}
