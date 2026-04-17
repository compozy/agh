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
	observepkg "github.com/pedronauck/agh/internal/observe"
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
	tasks = filterTaskListDrafts(tasks, params)
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

func (h *HostAPIHandler) handleTasksTimeline(ctx context.Context, raw json.RawMessage) (any, error) {
	var params hostAPITaskTimelineParams
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
	query, err := taskTimelineQueryFromParams(params.TaskTimelineQuery)
	if err != nil {
		return nil, err
	}

	items, err := manager.Timeline(ctx, taskID, query, actor)
	if err != nil {
		return nil, mapTaskRPCError(taskID, err)
	}
	return taskTimelineItemPayloadsFromItems(items), nil
}

func (h *HostAPIHandler) handleTasksTree(ctx context.Context, raw json.RawMessage) (any, error) {
	var params hostAPITaskTreeParams
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

	view, err := manager.Tree(ctx, taskID, actor)
	if err != nil {
		return nil, mapTaskRPCError(taskID, err)
	}
	return taskTreePayloadFromView(view), nil
}

func (h *HostAPIHandler) handleTasksDashboard(ctx context.Context, raw json.RawMessage) (any, error) {
	var params hostAPITaskDashboardParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}

	observer, err := h.taskObserver()
	if err != nil {
		return nil, err
	}
	query, err := h.taskDashboardQueryFromParams(ctx, params)
	if err != nil {
		return nil, err
	}

	view, err := observer.QueryTaskDashboard(ctx, query)
	if err != nil {
		return nil, mapTaskRPCError("", err)
	}
	return taskDashboardPayloadFromView(&view), nil
}

func (h *HostAPIHandler) handleTasksInbox(ctx context.Context, raw json.RawMessage) (any, error) {
	var params hostAPITaskInboxParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}

	observer, err := h.taskObserver()
	if err != nil {
		return nil, err
	}
	actor, err := h.taskActorContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("extension: derive task actor context: %w", err)
	}
	query, err := h.taskInboxQueryFromParams(ctx, params)
	if err != nil {
		return nil, err
	}

	view, err := observer.QueryTaskInbox(ctx, query, actor.Actor)
	if err != nil {
		return nil, mapTaskRPCError("", err)
	}
	return taskInboxPayloadFromView(view), nil
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

func (h *HostAPIHandler) handleTasksRunsGet(ctx context.Context, raw json.RawMessage) (any, error) {
	var params hostAPITaskRunGetParams
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

	view, err := manager.RunDetail(ctx, runID, actor)
	if err != nil {
		return nil, mapTaskRPCError(runID, err)
	}
	return taskRunDetailPayloadFromView(view), nil
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

func (h *HostAPIHandler) taskObserver() (hostAPIObserver, error) {
	if h == nil {
		return nil, errors.New("extension: host api handler is required")
	}
	if h.observer == nil {
		return nil, errors.New("extension: task observer is not configured")
	}
	return h.observer, nil
}

func (h *HostAPIHandler) taskManagerAndActor(ctx context.Context) (hostAPITaskManager, taskpkg.ActorContext, error) {
	manager, err := h.taskManager()
	if err != nil {
		return nil, taskpkg.ActorContext{}, fmt.Errorf("extension: resolve task manager: %w", err)
	}
	actor, err := h.taskActorContext(ctx)
	if err != nil {
		return nil, taskpkg.ActorContext{}, fmt.Errorf("extension: derive task actor context: %w", err)
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
		Scope:         params.Scope.Normalize(),
		Status:        params.Status.Normalize(),
		Priority:      params.Priority.Normalize(),
		ApprovalState: params.ApprovalState.Normalize(),
		OwnerKind:     params.OwnerKind.Normalize(),
		OwnerRef:      strings.TrimSpace(params.OwnerRef),
		ParentTaskID:  strings.TrimSpace(params.ParentTaskID),
		Search:        strings.TrimSpace(params.Query),
		Limit:         params.Limit,
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

func taskTimelineQueryFromParams(params apicontract.TaskTimelineQuery) (taskpkg.TimelineQuery, error) {
	query := taskpkg.TimelineQuery{
		AfterSequence: params.AfterSequence,
		Limit:         params.Limit,
	}
	if err := query.Validate("task_timeline_query"); err != nil {
		return taskpkg.TimelineQuery{}, invalidParamsRPCError(err)
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

func (h *HostAPIHandler) taskDashboardQueryFromParams(
	ctx context.Context,
	params apicontract.TaskDashboardQuery,
) (observepkg.TaskDashboardQuery, error) {
	query := observepkg.TaskDashboardQuery{
		Scope:          params.Scope.Normalize(),
		OwnerKind:      params.OwnerKind.Normalize(),
		OwnerRef:       strings.TrimSpace(params.OwnerRef),
		NetworkChannel: strings.TrimSpace(params.NetworkChannel),
		OriginKind:     params.OriginKind.Normalize(),
	}
	if workspaceRef := strings.TrimSpace(params.Workspace); workspaceRef != "" {
		if err := taskpkg.ValidateScopeBinding(
			query.Scope,
			workspaceRef,
			"task_dashboard_query",
			"workspace",
		); err != nil {
			return observepkg.TaskDashboardQuery{}, invalidParamsRPCError(err)
		}
		if query.Scope.Normalize() == taskpkg.ScopeWorkspace {
			workspaceID, err := h.resolveTaskWorkspaceID(ctx, workspaceRef)
			if err != nil {
				return observepkg.TaskDashboardQuery{}, err
			}
			query.WorkspaceID = workspaceID
		}
	}
	if err := validateTaskChannel("task_dashboard_query.network_channel", query.NetworkChannel); err != nil {
		return observepkg.TaskDashboardQuery{}, err
	}
	if err := query.Validate(); err != nil {
		return observepkg.TaskDashboardQuery{}, invalidParamsRPCError(err)
	}
	return query, nil
}

func (h *HostAPIHandler) taskInboxQueryFromParams(
	ctx context.Context,
	params apicontract.TaskInboxQuery,
) (observepkg.TaskInboxQuery, error) {
	query := observepkg.TaskInboxQuery{
		Scope:     params.Scope.Normalize(),
		OwnerKind: params.OwnerKind.Normalize(),
		OwnerRef:  strings.TrimSpace(params.OwnerRef),
		Lane:      observepkg.TaskInboxLane(params.Lane).Normalize(),
		Unread:    params.Unread,
		Search:    strings.TrimSpace(params.Query),
		Limit:     params.Limit,
	}
	if workspaceRef := strings.TrimSpace(params.Workspace); workspaceRef != "" {
		if err := taskpkg.ValidateScopeBinding(query.Scope, workspaceRef, "task_inbox_query", "workspace"); err != nil {
			return observepkg.TaskInboxQuery{}, invalidParamsRPCError(err)
		}
		if query.Scope.Normalize() == taskpkg.ScopeWorkspace {
			workspaceID, err := h.resolveTaskWorkspaceID(ctx, workspaceRef)
			if err != nil {
				return observepkg.TaskInboxQuery{}, err
			}
			query.WorkspaceID = workspaceID
		}
	}
	if err := query.Validate(); err != nil {
		return observepkg.TaskInboxQuery{}, invalidParamsRPCError(err)
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
		Priority:       req.Priority.Normalize(),
		MaxAttempts:    req.MaxAttempts,
		Draft:          req.Draft,
		ApprovalPolicy: req.ApprovalPolicy.Normalize(),
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
		Priority:       normalizePriorityPtr(req.Priority),
		MaxAttempts:    req.MaxAttempts,
		ApprovalPolicy: normalizeApprovalPolicyPtr(req.ApprovalPolicy),
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
		ID:              record.ID,
		Identifier:      record.Identifier,
		Scope:           record.Scope,
		WorkspaceID:     record.WorkspaceID,
		ParentTaskID:    record.ParentTaskID,
		NetworkChannel:  record.NetworkChannel,
		Title:           record.Title,
		Priority:        record.Priority,
		MaxAttempts:     record.MaxAttempts,
		Status:          record.Status,
		ApprovalPolicy:  record.ApprovalPolicy,
		ApprovalState:   record.ApprovalState,
		Draft:           record.Draft,
		Owner:           cloneOwnership(record.Owner),
		CreatedBy:       record.CreatedBy,
		Origin:          record.Origin,
		CreatedAt:       record.CreatedAt,
		UpdatedAt:       record.UpdatedAt,
		ClosedAt:        optionalTime(record.ClosedAt),
		ChildCount:      record.ChildCount,
		DependencyCount: record.DependencyCount,
		Dependencies:    taskDependencyReferencePayloadsFromReferences(record.Dependencies),
		ActiveRun:       taskRunSummaryPayloadFromSummary(record.ActiveRun),
		LastActivityAt:  optionalTime(record.LastActivityAt),
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
		Priority:       record.Priority,
		MaxAttempts:    record.MaxAttempts,
		Status:         record.Status,
		ApprovalPolicy: record.ApprovalPolicy,
		ApprovalState:  record.ApprovalState,
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

func taskReferencePayloadFromReference(record taskpkg.Reference) apicontract.TaskReferencePayload {
	return apicontract.TaskReferencePayload{
		ID:          record.ID,
		Identifier:  record.Identifier,
		Title:       record.Title,
		Status:      record.Status,
		Priority:    record.Priority,
		Owner:       cloneOwnership(record.Owner),
		Scope:       record.Scope,
		WorkspaceID: record.WorkspaceID,
	}
}

func taskRunSummaryPayloadFromSummary(summary *taskpkg.RunSummary) *apicontract.TaskRunSummaryPayload {
	if summary == nil {
		return nil
	}

	return &apicontract.TaskRunSummaryPayload{
		ID:          summary.ID,
		TaskID:      summary.TaskID,
		Status:      summary.Status,
		Attempt:     summary.Attempt,
		MaxAttempts: summary.MaxAttempts,
		SessionID:   summary.SessionID,
		ClaimedBy:   cloneActorIdentity(summary.ClaimedBy),
		QueuedAt:    summary.QueuedAt,
		ClaimedAt:   optionalTime(summary.ClaimedAt),
		StartedAt:   optionalTime(summary.StartedAt),
		EndedAt:     optionalTime(summary.EndedAt),
		Error:       summary.Error,
	}
}

func taskDependencyReferencePayloadsFromReferences(
	dependencies []taskpkg.DependencyReference,
) []apicontract.TaskDependencyReferencePayload {
	payloads := make([]apicontract.TaskDependencyReferencePayload, 0, len(dependencies))
	for _, dependency := range dependencies {
		payloads = append(payloads, apicontract.TaskDependencyReferencePayload{
			TaskID:          dependency.TaskID,
			DependsOnTaskID: dependency.DependsOnTaskID,
			Kind:            dependency.Kind,
			CreatedAt:       dependency.CreatedAt,
			DependsOn:       taskReferencePayloadFromReference(dependency.DependsOn),
		})
	}
	return payloads
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
		Summary:              taskSummaryPayloadFromSummary(view.Summary),
		Task:                 taskPayloadFromTask(&view.Task),
		Children:             taskSummaryPayloadsFromSummaries(view.Children),
		Dependencies:         taskDependencyPayloadsFromDependencies(view.Dependencies),
		DependencyReferences: taskDependencyReferencePayloadsFromReferences(view.DependencyReferences),
		Runs:                 taskRunPayloadsFromRuns(view.Runs),
		Events:               taskEventPayloadsFromEvents(view.Events),
	}
}

func taskTimelineItemPayloadsFromItems(items []taskpkg.TimelineItem) []apicontract.TaskTimelineItemPayload {
	payloads := make([]apicontract.TaskTimelineItemPayload, 0, len(items))
	for _, item := range items {
		payloads = append(payloads, taskTimelineItemPayloadFromItem(item))
	}
	return payloads
}

func taskTimelineItemPayloadFromItem(item taskpkg.TimelineItem) apicontract.TaskTimelineItemPayload {
	return apicontract.TaskTimelineItemPayload{
		Sequence:  item.Sequence,
		EventID:   item.EventID,
		Task:      taskReferencePayloadFromReference(item.Task),
		Run:       taskRunSummaryPayloadFromSummary(item.Run),
		EventType: item.EventType,
		Actor:     item.Actor,
		Origin:    item.Origin,
		Payload:   cloneRawMessage(item.Payload),
		Timestamp: item.Timestamp,
	}
}

func taskTreePayloadFromView(view *taskpkg.TreeView) apicontract.TaskTreePayload {
	if view == nil {
		return apicontract.TaskTreePayload{}
	}

	payload := apicontract.TaskTreePayload{
		Root: taskTreeNodePayloadFromNode(view.Root),
	}
	if len(view.Descendants) == 0 {
		return payload
	}

	payload.Descendants = make([]apicontract.TaskTreeNodePayload, 0, len(view.Descendants))
	for _, node := range view.Descendants {
		payload.Descendants = append(payload.Descendants, taskTreeNodePayloadFromNode(node))
	}
	return payload
}

func taskTreeNodePayloadFromNode(node taskpkg.TreeNode) apicontract.TaskTreeNodePayload {
	return apicontract.TaskTreeNodePayload{
		Task:           taskReferencePayloadFromReference(node.Task),
		ParentTaskID:   node.ParentTaskID,
		Depth:          node.Depth,
		ChildCount:     node.ChildCount,
		ActiveRun:      taskRunSummaryPayloadFromSummary(node.ActiveRun),
		LastActivityAt: node.LastActivityAt,
	}
}

func taskRunDetailPayloadFromView(view *taskpkg.RunDetailView) apicontract.TaskRunDetailPayload {
	if view == nil {
		return apicontract.TaskRunDetailPayload{}
	}

	return apicontract.TaskRunDetailPayload{
		Run:     taskRunPayloadFromRun(&view.Run),
		Task:    taskReferencePayloadFromReference(view.Task),
		Session: taskRunSessionPayloadFromSession(view.Session),
		Summary: taskRunOperationalSummaryPayloadFromSummary(view.Summary),
	}
}

func taskRunSessionPayloadFromSession(session *taskpkg.RunSessionRef) *apicontract.TaskRunSessionPayload {
	if session == nil {
		return nil
	}

	return &apicontract.TaskRunSessionPayload{
		SessionID:   session.SessionID,
		WorkspaceID: session.WorkspaceID,
		AgentName:   session.AgentName,
		Name:        session.Name,
		Channel:     session.Channel,
		State:       session.State,
		CreatedAt:   session.CreatedAt,
		UpdatedAt:   session.UpdatedAt,
	}
}

func taskRunOperationalSummaryPayloadFromSummary(
	summary taskpkg.RunOperationalSummary,
) apicontract.TaskRunOperationalSummaryPayload {
	return apicontract.TaskRunOperationalSummaryPayload{
		LastActivityAt: summary.LastActivityAt,
		LastEventType:  summary.LastEventType,
		ToolCallCount:  summary.ToolCallCount,
		TurnCount:      summary.TurnCount,
		InputTokens:    summary.InputTokens,
		OutputTokens:   summary.OutputTokens,
		TotalTokens:    summary.TotalTokens,
		TotalCost:      summary.TotalCost,
		CostCurrency:   summary.CostCurrency,
	}
}

func taskDashboardPayloadFromView(view *observepkg.TaskDashboardView) apicontract.TaskDashboardPayload {
	if view == nil {
		return apicontract.TaskDashboardPayload{}
	}

	return apicontract.TaskDashboardPayload{
		Totals:          taskDashboardTotalsPayload(view.Totals),
		Cards:           taskDashboardCardsPayload(view.Cards),
		StatusBreakdown: taskDashboardStatusBreakdownPayloads(view.StatusBreakdown),
		Queue:           taskDashboardQueuePayload(view.Queue),
		Health:          taskDashboardHealthPayload(view.Health),
		ActiveRuns:      taskDashboardActiveRunsPayload(view.ActiveRuns),
		Freshness:       taskDashboardFreshnessPayload(view.Freshness),
	}
}

func taskDashboardTotalsPayload(totals observepkg.TaskDashboardTotals) apicontract.TaskDashboardTotalsPayload {
	return apicontract.TaskDashboardTotalsPayload{
		TasksTotal:             totals.TasksTotal,
		RunsTotal:              totals.RunsTotal,
		DraftTasks:             totals.DraftTasks,
		PendingTasks:           totals.PendingTasks,
		ReadyTasks:             totals.ReadyTasks,
		InProgressTasks:        totals.InProgressTasks,
		BlockedTasks:           totals.BlockedTasks,
		CompletedTasks:         totals.CompletedTasks,
		FailedTasks:            totals.FailedTasks,
		CanceledTasks:          totals.CanceledTasks,
		AwaitingApprovalTasks:  totals.AwaitingApprovalTasks,
		DependencyBlockedTasks: totals.DependencyBlockedTasks,
		QueuedRuns:             totals.QueuedRuns,
		ClaimedRuns:            totals.ClaimedRuns,
		StartingRuns:           totals.StartingRuns,
		RunningRuns:            totals.RunningRuns,
		CompletedRuns:          totals.CompletedRuns,
		FailedRuns:             totals.FailedRuns,
		CanceledRuns:           totals.CanceledRuns,
		ActiveRuns:             totals.ActiveRuns,
	}
}

func taskDashboardCardsPayload(cards observepkg.TaskDashboardCards) apicontract.TaskDashboardCardsPayload {
	return apicontract.TaskDashboardCardsPayload{
		InProgress: apicontract.TaskDashboardInProgressCardPayload{
			Tasks:        cards.InProgress.Tasks,
			ActiveRuns:   cards.InProgress.ActiveRuns,
			RunningRuns:  cards.InProgress.RunningRuns,
			StartingRuns: cards.InProgress.StartingRuns,
			ClaimedRuns:  cards.InProgress.ClaimedRuns,
			QueuedRuns:   cards.InProgress.QueuedRuns,
			HealthStatus: cards.InProgress.HealthStatus,
		},
		Blocked: apicontract.TaskDashboardBlockedCardPayload{
			Tasks:                cards.Blocked.Tasks,
			AwaitingApproval:     cards.Blocked.AwaitingApproval,
			AwaitingDependencies: cards.Blocked.AwaitingDependencies,
			HealthStatus:         cards.Blocked.HealthStatus,
		},
		Failed: apicontract.TaskDashboardFailedCardPayload{
			Tasks:        cards.Failed.Tasks,
			FailedRuns:   cards.Failed.FailedRuns,
			ForcedStops:  cards.Failed.ForcedStops,
			HealthStatus: cards.Failed.HealthStatus,
		},
		Latency: apicontract.TaskDashboardLatencyCardPayload{
			ClaimLatencyMillis: taskLatencyMetricPayload(cards.Latency.ClaimLatencyMillis),
			StartLatencyMillis: taskLatencyMetricPayload(cards.Latency.StartLatencyMillis),
		},
	}
}

func taskLatencyMetricPayload(metric observepkg.LatencyMetric) apicontract.TaskLatencyMetricPayload {
	return apicontract.TaskLatencyMetricPayload{
		Samples:       metric.Samples,
		AverageMillis: metric.AverageMillis,
		MaximumMillis: metric.MaximumMillis,
	}
}

func taskDashboardStatusBreakdownPayloads(
	items []observepkg.TaskDashboardStatusBreakdown,
) []apicontract.TaskDashboardStatusBreakdownPayload {
	if len(items) == 0 {
		return nil
	}

	payloads := make([]apicontract.TaskDashboardStatusBreakdownPayload, 0, len(items))
	for _, item := range items {
		payloads = append(payloads, apicontract.TaskDashboardStatusBreakdownPayload{
			Status:       item.Status,
			Count:        item.Count,
			SharePercent: item.SharePercent,
		})
	}
	return payloads
}

func taskDashboardQueuePayload(queue observepkg.TaskDashboardQueue) apicontract.TaskDashboardQueuePayload {
	payload := apicontract.TaskDashboardQueuePayload{
		Total:                 queue.Total,
		OldestQueuedAt:        queue.OldestQueuedAt,
		OldestQueueAgeMilli:   queue.OldestQueueAgeMilli,
		BacklogWarning:        queue.BacklogWarning,
		BacklogStatus:         queue.BacklogStatus,
		BacklogThresholdMilli: queue.BacklogThresholdMilli,
	}
	if len(queue.Depth) == 0 {
		return payload
	}

	payload.Depth = make([]apicontract.TaskDashboardQueueDepthPayload, 0, len(queue.Depth))
	for _, item := range queue.Depth {
		payload.Depth = append(payload.Depth, apicontract.TaskDashboardQueueDepthPayload{
			NetworkChannel:      item.NetworkChannel,
			Count:               item.Count,
			OldestQueuedAt:      item.OldestQueuedAt,
			OldestQueueAgeMilli: item.OldestQueueAgeMilli,
		})
	}
	return payload
}

func taskDashboardHealthPayload(health observepkg.TaskDashboardHealth) apicontract.TaskDashboardHealthPayload {
	return apicontract.TaskDashboardHealthPayload{
		Status:           health.Status,
		StuckRuns:        health.StuckRuns,
		ActiveOrphanRuns: health.ActiveOrphanRuns,
		QueueBacklog:     health.QueueBacklog,
	}
}

func taskDashboardActiveRunsPayload(
	activeRuns observepkg.TaskDashboardActiveRuns,
) apicontract.TaskDashboardActiveRunsPayload {
	payload := apicontract.TaskDashboardActiveRunsPayload{
		Total:    activeRuns.Total,
		Running:  activeRuns.Running,
		Starting: activeRuns.Starting,
		Claimed:  activeRuns.Claimed,
		Queued:   activeRuns.Queued,
	}
	if len(activeRuns.Items) == 0 {
		return payload
	}

	payload.Items = make([]apicontract.TaskDashboardActiveRunPayload, 0, len(activeRuns.Items))
	for _, item := range activeRuns.Items {
		payload.Items = append(payload.Items, apicontract.TaskDashboardActiveRunPayload{
			TaskID:         item.TaskID,
			TaskIdentifier: item.TaskIdentifier,
			TaskTitle:      item.TaskTitle,
			TaskStatus:     item.TaskStatus,
			TaskPriority:   item.TaskPriority,
			TaskOwner:      cloneOwnership(item.TaskOwner),
			Scope:          item.Scope,
			WorkspaceID:    item.WorkspaceID,
			RunID:          item.RunID,
			RunStatus:      item.RunStatus,
			Attempt:        item.Attempt,
			MaxAttempts:    item.MaxAttempts,
			SessionID:      item.SessionID,
			NetworkChannel: item.NetworkChannel,
			LastActivityAt: item.LastActivityAt,
			AgeMilli:       item.AgeMilli,
			HealthStatus:   item.HealthStatus,
			Stuck:          item.Stuck,
			Error:          item.Error,
		})
	}
	return payload
}

func taskDashboardFreshnessPayload(
	freshness observepkg.TaskDashboardFreshness,
) apicontract.TaskDashboardFreshnessPayload {
	return apicontract.TaskDashboardFreshnessPayload{
		ObservedAt:       freshness.ObservedAt,
		LatestActivityAt: freshness.LatestActivityAt,
		AgeMilli:         freshness.AgeMilli,
		StaleAfterMilli:  freshness.StaleAfterMilli,
		HasLiveWork:      freshness.HasLiveWork,
		Status:           freshness.Status,
		Stale:            freshness.Stale,
	}
}

func taskInboxPayloadFromView(view observepkg.TaskInboxView) apicontract.TaskInboxPayload {
	payload := apicontract.TaskInboxPayload{
		Total:         view.Total,
		UnreadTotal:   view.UnreadTotal,
		ArchivedTotal: view.ArchivedTotal,
	}
	if len(view.Groups) == 0 {
		return payload
	}

	payload.Groups = make([]apicontract.TaskInboxLaneGroupPayload, 0, len(view.Groups))
	for _, group := range view.Groups {
		groupPayload := apicontract.TaskInboxLaneGroupPayload{
			Lane:        apicontract.TaskInboxLane(group.Lane),
			Count:       group.Count,
			UnreadCount: group.UnreadCount,
		}
		if len(group.Items) > 0 {
			groupPayload.Items = make([]apicontract.TaskInboxItemPayload, 0, len(group.Items))
			for _, item := range group.Items {
				groupPayload.Items = append(groupPayload.Items, apicontract.TaskInboxItemPayload{
					Task:             taskReferencePayloadFromReference(item.Task),
					Lane:             apicontract.TaskInboxLane(item.Lane),
					ApprovalPolicy:   item.ApprovalPolicy,
					ApprovalState:    item.ApprovalState,
					BlockingReason:   item.BlockingReason,
					LatestActivityAt: item.LatestActivityAt,
					Run:              taskRunSummaryPayloadFromSummary(item.Run),
					Triage:           taskTriageStatePayloadFromState(item.Triage),
				})
			}
		}
		payload.Groups = append(payload.Groups, groupPayload)
	}
	return payload
}

func taskTriageStatePayloadFromState(state taskpkg.TriageState) apicontract.TaskTriageStatePayload {
	return apicontract.TaskTriageStatePayload{
		TaskID:             state.TaskID,
		Actor:              state.Actor,
		Read:               state.Read,
		Archived:           state.Archived,
		Dismissed:          state.Dismissed,
		LastSeenActivityAt: optionalTime(state.LastSeenActivityAt),
		UpdatedAt:          state.UpdatedAt,
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

func filterTaskListDrafts(tasks []taskpkg.Summary, query apicontract.TaskListQuery) []taskpkg.Summary {
	if query.IncludeDrafts || query.Status.Normalize() != "" {
		return tasks
	}

	filtered := make([]taskpkg.Summary, 0, len(tasks))
	for _, task := range tasks {
		if task.Draft || task.Status.Normalize() == taskpkg.TaskStatusDraft {
			continue
		}
		filtered = append(filtered, task)
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
	return &taskpkg.Ownership{
		Kind: source.Kind.Normalize(),
		Ref:  strings.TrimSpace(source.Ref),
	}
}

func cloneActorIdentity(source *taskpkg.ActorIdentity) *taskpkg.ActorIdentity {
	if source == nil {
		return nil
	}
	return &taskpkg.ActorIdentity{
		Kind: source.Kind.Normalize(),
		Ref:  strings.TrimSpace(source.Ref),
	}
}

func trimStringPtr(source *string) *string {
	if source == nil {
		return nil
	}
	value := strings.TrimSpace(*source)
	return &value
}

func normalizePriorityPtr(source *taskpkg.Priority) *taskpkg.Priority {
	if source == nil {
		return nil
	}
	normalized := source.Normalize()
	return &normalized
}

func normalizeApprovalPolicyPtr(source *taskpkg.ApprovalPolicy) *taskpkg.ApprovalPolicy {
	if source == nil {
		return nil
	}
	normalized := source.Normalize()
	return &normalized
}

func optionalTime(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	cloned := value
	return &cloned
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
