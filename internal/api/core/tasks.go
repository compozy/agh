package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/network"
	taskpkg "github.com/pedronauck/agh/internal/task"
)

const (
	defaultTaskActorRef        = "local-user"
	taskActionList             = "list"
	taskActionCreate           = "create"
	taskActionGet              = "get"
	taskActionUpdate           = "update"
	taskActionCancel           = "cancel"
	taskActionCreateChild      = "create_child"
	taskActionAddDependency    = "add_dependency"
	taskActionRemoveDependency = "remove_dependency"
	taskActionListRuns         = "list_runs"
	taskActionEnqueueRun       = "enqueue_run"
	taskActionClaimRun         = "claim_run"
	taskActionStartRun         = "start_run"
	taskActionAttachRun        = "attach_run_session"
	taskActionCompleteRun      = "complete_run"
	taskActionFailRun          = "fail_run"
	taskActionCancelRun        = "cancel_run"
)

func (h *BaseHandlers) requireTaskManager(c *gin.Context) (TaskService, bool) {
	if h.Tasks == nil {
		h.respondError(c, http.StatusServiceUnavailable, fmt.Errorf("%s: task service is not configured", h.transportName()))
		return nil, false
	}
	return h.Tasks, true
}

func (h *BaseHandlers) taskActorContext(c *gin.Context, action string) (taskpkg.ActorContext, error) {
	if h.TaskActorContextResolver != nil {
		return h.TaskActorContextResolver(c, action)
	}
	return taskpkg.DeriveHumanActorContext(
		defaultTaskActorRef,
		taskOriginKindForTransport(h.transportName()),
		"tasks."+strings.TrimSpace(action),
	)
}

func taskOriginKindForTransport(name string) taskpkg.OriginKind {
	normalized := strings.ToLower(strings.TrimSpace(name))
	switch {
	case strings.Contains(normalized, "uds"):
		return taskpkg.OriginKindUDS
	case strings.Contains(normalized, "web"):
		return taskpkg.OriginKindWeb
	case strings.Contains(normalized, "cli"):
		return taskpkg.OriginKindCLI
	default:
		return taskpkg.OriginKindHTTP
	}
}

// ListTasks returns the filtered task list.
func (h *BaseHandlers) ListTasks(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	actor, err := h.taskActorContext(c, taskActionList)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	query, err := h.parseTaskListQuery(c.Request.Context(), c)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	tasks, err := manager.ListTasks(c.Request.Context(), query, actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.TasksResponse{Tasks: TaskSummaryPayloadsFromSummaries(tasks)})
}

// CreateTask creates one new task.
func (h *BaseHandlers) CreateTask(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	var req contract.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, NewTaskValidationError(fmt.Errorf("%s: decode create task request: %w", h.transportName(), err)))
		return
	}

	actor, err := h.taskActorContext(c, taskActionCreate)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	spec, err := h.createTaskSpecFromRequest(c.Request.Context(), req)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	record, err := manager.CreateTask(c.Request.Context(), spec, actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	c.JSON(http.StatusCreated, contract.TaskResponse{Task: TaskPayloadFromTask(record)})
}

// GetTask returns one expanded task view.
func (h *BaseHandlers) GetTask(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	taskID, err := requiredPathID(c.Param("id"), "task id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	actor, err := h.taskActorContext(c, taskActionGet)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	view, err := manager.GetTask(c.Request.Context(), taskID, actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.TaskDetailResponse{Task: TaskDetailPayloadFromView(view)})
}

// UpdateTask patches one mutable task surface.
func (h *BaseHandlers) UpdateTask(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	taskID, err := requiredPathID(c.Param("id"), "task id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	var req contract.UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, NewTaskValidationError(fmt.Errorf("%s: decode update task request: %w", h.transportName(), err)))
		return
	}
	if !req.HasChanges() {
		err := NewTaskValidationError(errors.New("task update must include at least one mutable field"))
		h.respondError(c, http.StatusBadRequest, err)
		return
	}

	actor, err := h.taskActorContext(c, taskActionUpdate)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	patch, err := taskPatchFromRequest(req)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	record, err := manager.UpdateTask(c.Request.Context(), taskID, patch, actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.TaskResponse{Task: TaskPayloadFromTask(record)})
}

// CancelTask requests cancellation for one task tree.
func (h *BaseHandlers) CancelTask(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	taskID, err := requiredPathID(c.Param("id"), "task id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	var req contract.CancelTaskRequest
	if err := decodeOptionalJSON(c, &req); err != nil {
		h.respondError(c, http.StatusBadRequest, NewTaskValidationError(fmt.Errorf("%s: decode cancel task request: %w", h.transportName(), err)))
		return
	}

	actor, err := h.taskActorContext(c, taskActionCancel)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	cancelReq, err := cancelTaskFromRequest(req)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	record, err := manager.CancelTask(c.Request.Context(), taskID, cancelReq, actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.TaskResponse{Task: TaskPayloadFromTask(record)})
}

// CreateChildTask creates one child task beneath the supplied parent.
func (h *BaseHandlers) CreateChildTask(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	parentTaskID, err := requiredPathID(c.Param("id"), "task id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	var req contract.CreateTaskChildRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, NewTaskValidationError(fmt.Errorf("%s: decode create child task request: %w", h.transportName(), err)))
		return
	}

	actor, err := h.taskActorContext(c, taskActionCreateChild)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	spec, err := h.createChildTaskSpecFromRequest(c.Request.Context(), req)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	record, err := manager.CreateChildTask(c.Request.Context(), parentTaskID, spec, actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	c.JSON(http.StatusCreated, contract.TaskResponse{Task: TaskPayloadFromTask(record)})
}

// AddTaskDependency adds one blocking dependency edge.
func (h *BaseHandlers) AddTaskDependency(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	taskID, err := requiredPathID(c.Param("id"), "task id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	var req contract.AddTaskDependencyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, NewTaskValidationError(fmt.Errorf("%s: decode add dependency request: %w", h.transportName(), err)))
		return
	}

	actor, err := h.taskActorContext(c, taskActionAddDependency)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	spec, err := addTaskDependencyFromRequest(taskID, req)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	if err := manager.AddDependency(c.Request.Context(), spec, actor); err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	view, err := manager.GetTask(c.Request.Context(), taskID, actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.TaskDetailResponse{Task: TaskDetailPayloadFromView(view)})
}

// RemoveTaskDependency removes one blocking dependency edge.
func (h *BaseHandlers) RemoveTaskDependency(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	taskID, err := requiredPathID(c.Param("id"), "task id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}
	dependsOnID, err := requiredPathID(c.Param("depends_on_id"), "depends_on_id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	actor, err := h.taskActorContext(c, taskActionRemoveDependency)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	if err := manager.RemoveDependency(c.Request.Context(), taskID, dependsOnID, actor); err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	view, err := manager.GetTask(c.Request.Context(), taskID, actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.TaskDetailResponse{Task: TaskDetailPayloadFromView(view)})
}

// ListTaskRuns returns the filtered run list for one task.
func (h *BaseHandlers) ListTaskRuns(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	taskID, err := requiredPathID(c.Param("id"), "task id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	actor, err := h.taskActorContext(c, taskActionListRuns)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	query, err := parseTaskRunListQuery(c)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	runs, err := manager.ListTaskRuns(c.Request.Context(), taskID, query, actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.TaskRunsResponse{Runs: TaskRunPayloadsFromRuns(runs)})
}

// EnqueueTaskRun creates one new queue-first run for the supplied task.
func (h *BaseHandlers) EnqueueTaskRun(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	taskID, err := requiredPathID(c.Param("id"), "task id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	var req contract.EnqueueTaskRunRequest
	if err := decodeOptionalJSON(c, &req); err != nil {
		h.respondError(c, http.StatusBadRequest, NewTaskValidationError(fmt.Errorf("%s: decode enqueue run request: %w", h.transportName(), err)))
		return
	}

	actor, err := h.taskActorContext(c, taskActionEnqueueRun)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	spec, err := enqueueTaskRunFromRequest(taskID, req)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	run, err := manager.EnqueueRun(c.Request.Context(), spec, actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	c.JSON(http.StatusCreated, contract.TaskRunResponse{Run: TaskRunPayloadFromRun(run)})
}

// ClaimTaskRun claims one queued run.
func (h *BaseHandlers) ClaimTaskRun(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	runID, err := requiredPathID(c.Param("id"), "run id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	var req contract.ClaimTaskRunRequest
	if err := decodeOptionalJSON(c, &req); err != nil {
		h.respondError(c, http.StatusBadRequest, NewTaskValidationError(fmt.Errorf("%s: decode claim run request: %w", h.transportName(), err)))
		return
	}

	actor, err := h.taskActorContext(c, taskActionClaimRun)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	claim, err := claimTaskRunFromRequest(req)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	run, err := manager.ClaimRun(c.Request.Context(), runID, claim, actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.TaskRunResponse{Run: TaskRunPayloadFromRun(run)})
}

// StartTaskRun starts one claimed run.
func (h *BaseHandlers) StartTaskRun(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	runID, err := requiredPathID(c.Param("id"), "run id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	var req contract.StartTaskRunRequest
	if err := decodeOptionalJSON(c, &req); err != nil {
		h.respondError(c, http.StatusBadRequest, NewTaskValidationError(fmt.Errorf("%s: decode start run request: %w", h.transportName(), err)))
		return
	}

	actor, err := h.taskActorContext(c, taskActionStartRun)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	startReq, err := startTaskRunFromRequest(req)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	run, err := manager.StartRun(c.Request.Context(), runID, startReq, actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.TaskRunResponse{Run: TaskRunPayloadFromRun(run)})
}

// AttachTaskRunSession binds one existing session to a run.
func (h *BaseHandlers) AttachTaskRunSession(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	runID, err := requiredPathID(c.Param("id"), "run id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	var req contract.AttachTaskRunSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, NewTaskValidationError(fmt.Errorf("%s: decode attach run session request: %w", h.transportName(), err)))
		return
	}

	actor, err := h.taskActorContext(c, taskActionAttachRun)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	sessionID, err := attachTaskRunSessionIDFromRequest(req)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	run, err := manager.AttachRunSession(c.Request.Context(), runID, sessionID, actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.TaskRunResponse{Run: TaskRunPayloadFromRun(run)})
}

// CompleteTaskRun marks one running run as completed.
func (h *BaseHandlers) CompleteTaskRun(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	runID, err := requiredPathID(c.Param("id"), "run id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	var req contract.CompleteTaskRunRequest
	if err := decodeOptionalJSON(c, &req); err != nil {
		h.respondError(c, http.StatusBadRequest, NewTaskValidationError(fmt.Errorf("%s: decode complete run request: %w", h.transportName(), err)))
		return
	}

	actor, err := h.taskActorContext(c, taskActionCompleteRun)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	result, err := completeTaskRunFromRequest(req)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	run, err := manager.CompleteRun(c.Request.Context(), runID, result, actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.TaskRunResponse{Run: TaskRunPayloadFromRun(run)})
}

// FailTaskRun marks one run as failed.
func (h *BaseHandlers) FailTaskRun(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	runID, err := requiredPathID(c.Param("id"), "run id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	var req contract.FailTaskRunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, NewTaskValidationError(fmt.Errorf("%s: decode fail run request: %w", h.transportName(), err)))
		return
	}

	actor, err := h.taskActorContext(c, taskActionFailRun)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	failure, err := failTaskRunFromRequest(req)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	run, err := manager.FailRun(c.Request.Context(), runID, failure, actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.TaskRunResponse{Run: TaskRunPayloadFromRun(run)})
}

// CancelTaskRun cancels one non-terminal run.
func (h *BaseHandlers) CancelTaskRun(c *gin.Context) {
	manager, ok := h.requireTaskManager(c)
	if !ok {
		return
	}

	runID, err := requiredPathID(c.Param("id"), "run id")
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	var req contract.CancelTaskRunRequest
	if err := decodeOptionalJSON(c, &req); err != nil {
		h.respondError(c, http.StatusBadRequest, NewTaskValidationError(fmt.Errorf("%s: decode cancel run request: %w", h.transportName(), err)))
		return
	}

	actor, err := h.taskActorContext(c, taskActionCancelRun)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	cancelReq, err := cancelTaskRunFromRequest(req)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	run, err := manager.CancelRun(c.Request.Context(), runID, cancelReq, actor)
	if err != nil {
		h.respondError(c, StatusForTaskError(err), err)
		return
	}

	c.JSON(http.StatusOK, contract.TaskRunResponse{Run: TaskRunPayloadFromRun(run)})
}

func (h *BaseHandlers) parseTaskListQuery(ctx context.Context, c *gin.Context) (taskpkg.TaskQuery, error) {
	limit, err := ParseOptionalInt(c.Query("limit"))
	if err != nil {
		return taskpkg.TaskQuery{}, NewTaskValidationError(err)
	}

	query := taskpkg.TaskQuery{
		Scope:        taskpkg.Scope(strings.TrimSpace(c.Query("scope"))).Normalize(),
		Status:       taskpkg.TaskStatus(strings.TrimSpace(c.Query("status"))).Normalize(),
		OwnerKind:    taskpkg.OwnerKind(strings.TrimSpace(c.Query("owner_kind"))).Normalize(),
		OwnerRef:     strings.TrimSpace(c.Query("owner_ref")),
		ParentTaskID: strings.TrimSpace(c.Query("parent_task_id")),
		Limit:        limit,
	}

	if workspaceRef := strings.TrimSpace(c.Query("workspace")); workspaceRef != "" {
		if query.Scope.Normalize() == taskpkg.ScopeGlobal {
			return taskpkg.TaskQuery{}, taskpkg.ValidateScopeBinding(query.Scope, workspaceRef, "task_query", "workspace")
		}
		workspaceID, err := h.lookupWorkspaceID(ctx, workspaceRef)
		if err != nil {
			return taskpkg.TaskQuery{}, err
		}
		query.WorkspaceID = workspaceID
	}

	if networkChannel := strings.TrimSpace(c.Query("network_channel")); networkChannel != "" {
		if err := validateTaskChannel("task_query.network_channel", networkChannel); err != nil {
			return taskpkg.TaskQuery{}, err
		}
		query.NetworkChannel = networkChannel
	}

	if err := query.Validate("task_query"); err != nil {
		return taskpkg.TaskQuery{}, err
	}
	return query, nil
}

func parseTaskRunListQuery(c *gin.Context) (taskpkg.TaskRunQuery, error) {
	limit, err := ParseOptionalInt(c.Query("limit"))
	if err != nil {
		return taskpkg.TaskRunQuery{}, NewTaskValidationError(err)
	}

	query := taskpkg.TaskRunQuery{
		Status:    taskpkg.TaskRunStatus(strings.TrimSpace(c.Query("status"))).Normalize(),
		SessionID: strings.TrimSpace(c.Query("session_id")),
		Limit:     limit,
	}
	if err := query.Validate("task_run_query"); err != nil {
		return taskpkg.TaskRunQuery{}, err
	}
	return query, nil
}

func (h *BaseHandlers) createTaskSpecFromRequest(ctx context.Context, req contract.CreateTaskRequest) (taskpkg.CreateTask, error) {
	scope := req.Scope.Normalize()
	workspaceID, err := h.resolveTaskWorkspaceBinding(ctx, scope, req.Workspace, "create_task")
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
		return taskpkg.CreateTask{}, err
	}
	return spec, nil
}

func (h *BaseHandlers) createChildTaskSpecFromRequest(ctx context.Context, req contract.CreateTaskChildRequest) (taskpkg.CreateTask, error) {
	scope := req.Scope.Normalize()
	workspaceID, err := h.resolveTaskWorkspaceBinding(ctx, scope, req.Workspace, "create_child_task")
	if err != nil {
		return taskpkg.CreateTask{}, err
	}
	if err := validateTaskChannel("create_child_task.network_channel", req.NetworkChannel); err != nil {
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
	if err := spec.Validate("create_child_task"); err != nil {
		return taskpkg.CreateTask{}, err
	}
	return spec, nil
}

func taskPatchFromRequest(req contract.UpdateTaskRequest) (taskpkg.TaskPatch, error) {
	if req.NetworkChannel != nil {
		if err := validateTaskChannel("task_patch.network_channel", *req.NetworkChannel); err != nil {
			return taskpkg.TaskPatch{}, err
		}
	}

	patch := taskpkg.TaskPatch{
		Title:          trimStringPtr(req.Title),
		Description:    trimStringPtr(req.Description),
		Metadata:       cloneRawMessagePtr(req.Metadata),
		NetworkChannel: trimStringPtr(req.NetworkChannel),
		Owner:          cloneOwnership(req.Owner),
		ClearOwner:     req.ClearOwner,
	}
	if err := patch.Validate("task_patch"); err != nil {
		return taskpkg.TaskPatch{}, err
	}
	return patch, nil
}

func cancelTaskFromRequest(req contract.CancelTaskRequest) (taskpkg.CancelTask, error) {
	cancelReq := taskpkg.CancelTask{
		Reason:   strings.TrimSpace(req.Reason),
		Metadata: cloneRawMessage(req.Metadata),
	}
	if err := cancelReq.Validate("cancel_task"); err != nil {
		return taskpkg.CancelTask{}, err
	}
	return cancelReq, nil
}

func addTaskDependencyFromRequest(taskID string, req contract.AddTaskDependencyRequest) (taskpkg.AddDependency, error) {
	kind := req.Kind.Normalize()
	if kind == "" {
		kind = taskpkg.DependencyKindBlocks
	}

	spec := taskpkg.AddDependency{
		TaskID:          strings.TrimSpace(taskID),
		DependsOnTaskID: strings.TrimSpace(req.DependsOnTaskID),
		Kind:            kind,
	}
	if err := spec.Validate("add_dependency"); err != nil {
		return taskpkg.AddDependency{}, err
	}
	return spec, nil
}

func enqueueTaskRunFromRequest(taskID string, req contract.EnqueueTaskRunRequest) (taskpkg.EnqueueRun, error) {
	if err := validateTaskChannel("enqueue_run.network_channel", req.NetworkChannel); err != nil {
		return taskpkg.EnqueueRun{}, err
	}

	spec := taskpkg.EnqueueRun{
		TaskID:         strings.TrimSpace(taskID),
		IdempotencyKey: strings.TrimSpace(req.IdempotencyKey),
		NetworkChannel: strings.TrimSpace(req.NetworkChannel),
	}
	if err := spec.Validate("enqueue_run"); err != nil {
		return taskpkg.EnqueueRun{}, err
	}
	return spec, nil
}

func claimTaskRunFromRequest(req contract.ClaimTaskRunRequest) (taskpkg.ClaimRun, error) {
	claim := taskpkg.ClaimRun{IdempotencyKey: strings.TrimSpace(req.IdempotencyKey)}
	if err := claim.Validate("claim_run"); err != nil {
		return taskpkg.ClaimRun{}, err
	}
	return claim, nil
}

func startTaskRunFromRequest(req contract.StartTaskRunRequest) (taskpkg.StartRun, error) {
	startReq := taskpkg.StartRun{IdempotencyKey: strings.TrimSpace(req.IdempotencyKey)}
	if err := startReq.Validate("start_run"); err != nil {
		return taskpkg.StartRun{}, err
	}
	return startReq, nil
}

func attachTaskRunSessionIDFromRequest(req contract.AttachTaskRunSessionRequest) (string, error) {
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return "", NewTaskValidationError(errors.New("session_id is required"))
	}
	return sessionID, nil
}

func completeTaskRunFromRequest(req contract.CompleteTaskRunRequest) (taskpkg.RunResult, error) {
	result := taskpkg.RunResult{Value: cloneRawMessage(req.Result)}
	if err := result.Validate("run_result"); err != nil {
		return taskpkg.RunResult{}, err
	}
	return result, nil
}

func failTaskRunFromRequest(req contract.FailTaskRunRequest) (taskpkg.RunFailure, error) {
	failure := taskpkg.RunFailure{
		Error:    strings.TrimSpace(req.Error),
		Metadata: cloneRawMessage(req.Metadata),
	}
	if err := failure.Validate("run_failure"); err != nil {
		return taskpkg.RunFailure{}, err
	}
	return failure, nil
}

func cancelTaskRunFromRequest(req contract.CancelTaskRunRequest) (taskpkg.CancelRun, error) {
	cancelReq := taskpkg.CancelRun{
		Reason:   strings.TrimSpace(req.Reason),
		Metadata: cloneRawMessage(req.Metadata),
	}
	if err := cancelReq.Validate("cancel_run"); err != nil {
		return taskpkg.CancelRun{}, err
	}
	return cancelReq, nil
}

func (h *BaseHandlers) resolveTaskWorkspaceBinding(ctx context.Context, scope taskpkg.Scope, workspaceRef string, path string) (string, error) {
	trimmed := strings.TrimSpace(workspaceRef)
	if err := taskpkg.ValidateScopeBinding(scope, trimmed, path, "workspace"); err != nil {
		return "", err
	}
	if scope.Normalize() != taskpkg.ScopeWorkspace {
		return "", nil
	}
	return h.lookupWorkspaceID(ctx, trimmed)
}

func validateTaskChannel(path string, channel string) error {
	trimmed := strings.TrimSpace(channel)
	if trimmed == "" {
		return nil
	}
	if err := network.ValidateChannel(trimmed); err != nil {
		return NewTaskValidationError(fmt.Errorf("%s: %w", path, err))
	}
	return nil
}

func requiredPathID(raw string, field string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", NewTaskValidationError(fmt.Errorf("%s is required", field))
	}
	return trimmed, nil
}

func decodeOptionalJSON(c *gin.Context, dest any) error {
	if err := c.ShouldBindJSON(dest); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

// TaskSummaryPayloadsFromSummaries converts task summaries into shared payloads.
func TaskSummaryPayloadsFromSummaries(tasks []taskpkg.TaskSummary) []contract.TaskSummaryPayload {
	payloads := make([]contract.TaskSummaryPayload, 0, len(tasks))
	for _, record := range tasks {
		payloads = append(payloads, TaskSummaryPayloadFromSummary(record))
	}
	return payloads
}

// TaskSummaryPayloadFromSummary converts one task summary into the shared payload.
func TaskSummaryPayloadFromSummary(record taskpkg.TaskSummary) contract.TaskSummaryPayload {
	return contract.TaskSummaryPayload{
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

// TaskPayloadFromTask converts one task record into the shared payload.
func TaskPayloadFromTask(record *taskpkg.Task) contract.TaskPayload {
	if record == nil {
		return contract.TaskPayload{}
	}

	return contract.TaskPayload{
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

// TaskDependencyPayloadsFromDependencies converts dependency records into shared payloads.
func TaskDependencyPayloadsFromDependencies(dependencies []taskpkg.TaskDependency) []contract.TaskDependencyPayload {
	payloads := make([]contract.TaskDependencyPayload, 0, len(dependencies))
	for _, dependency := range dependencies {
		payloads = append(payloads, contract.TaskDependencyPayload{
			TaskID:          dependency.TaskID,
			DependsOnTaskID: dependency.DependsOnTaskID,
			Kind:            dependency.Kind,
			CreatedAt:       dependency.CreatedAt,
		})
	}
	return payloads
}

// TaskRunPayloadsFromRuns converts task runs into shared payloads.
func TaskRunPayloadsFromRuns(runs []taskpkg.TaskRun) []contract.TaskRunPayload {
	payloads := make([]contract.TaskRunPayload, 0, len(runs))
	for _, run := range runs {
		payloads = append(payloads, TaskRunPayloadFromRun(&run))
	}
	return payloads
}

// TaskRunPayloadFromRun converts one task run into the shared payload.
func TaskRunPayloadFromRun(run *taskpkg.TaskRun) contract.TaskRunPayload {
	if run == nil {
		return contract.TaskRunPayload{}
	}

	return contract.TaskRunPayload{
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

// TaskEventPayloadsFromEvents converts task events into shared payloads.
func TaskEventPayloadsFromEvents(events []taskpkg.TaskEvent) []contract.TaskEventPayload {
	payloads := make([]contract.TaskEventPayload, 0, len(events))
	for _, event := range events {
		payloads = append(payloads, contract.TaskEventPayload{
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

// TaskDetailPayloadFromView converts one expanded task view into the shared payload.
func TaskDetailPayloadFromView(view *taskpkg.TaskView) contract.TaskDetailPayload {
	if view == nil {
		return contract.TaskDetailPayload{}
	}

	return contract.TaskDetailPayload{
		Task:         TaskPayloadFromTask(&view.Task),
		Children:     TaskSummaryPayloadsFromSummaries(view.Children),
		Dependencies: TaskDependencyPayloadsFromDependencies(view.Dependencies),
		Runs:         TaskRunPayloadsFromRuns(view.Runs),
		Events:       TaskEventPayloadsFromEvents(view.Events),
	}
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
	trimmed := strings.TrimSpace(*source)
	return &trimmed
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
	copyValue := cloneRawMessage(*source)
	return &copyValue
}
